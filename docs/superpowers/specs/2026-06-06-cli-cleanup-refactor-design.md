# cc-env 清理与重构设计

日期：2026-06-06
状态：已确认，待生成实施计划

## 背景

`cc-env` 的核心设计是合理的：通过以「清理后的环境变量 + profile 覆盖」启动 `claude` 子进程（提交 b03a41a），而不是改写 `~/.claude/settings.json`，因此切换是无状态、可逆的，不会污染官方登录态；`profile.Save` 采用临时文件 + `rename` 的原子写入；包边界清晰、零外部依赖、测试覆盖充分。

本次工作**不涉及核心设计的重写**，只做三类清理与可维护性改进。这是对现有代码的整理，不新增任何功能、不改变任何对外行为（UI 文案的去重除外，见第三部分）。

## 目标与范围

仅包含以下三部分：

1. 删除死代码
2. 拆分过大的 `internal/cli/app.go`（1261 行）
3. 修复交互界面的文案冗余

**总体验证标准**：每一部分完成后，`go build ./... && go vet ./... && go test ./... -count=1` 全部通过。

## 不在本次范围内（Non-Goals）

- **环境变量键的扩散问题**（新增一个受管变量需改动约 5 处：`ManagedEnvKeys`、`profileFlags`、`buildProfileEnv`、`promptAddFields`、`promptEditFields` 及 flag 注册）。这是更深的重构，本次不做。
- **拆分测试文件** `app_test.go`（3687 行）。为保持 diff 可审查，本次保留为单文件；如需镜像拆分，后续单独进行。
- `internal/profile/store.go` 与 `internal/cli/app.go` 中重复的 `normalizeProfileName`（均为 `strings.TrimSpace`）——保持现状，不合并。
- `buildClaudeEnv` 与 `profile.SupportedEnvKeys` 之间受管键集合的轻微重复——保持现状。

---

## 第一部分 — 删除死代码

不改变任何行为。

1. **删除整个 `internal/settings/` 包**：`store.go`、`backup.go`、`store_test.go`（约 350 行）。
   - 已通过 `grep -rn "internal/settings"` 确认无任何生产代码引用，是 b03a41a 之前的遗留物。
   - CLAUDE.md 中描述此包为「legacy, no longer called」，删除后需同步更新 CLAUDE.md 架构图，去掉 `settings/` 一行。
2. **删除 `internal/output/style.go` 中的 `styler.muted()`**：已定义但从未被调用。
3. **去掉 `promptAddValue` 的 `sensitive` 参数**：该参数被三个调用方（`promptAddName`、`promptRenameName`、`promptAddFields`）恒为 `false` 传入，且函数体内 `_ = sensitive` 直接忽略。脱敏仅作用于展示已有值（`maskValue(current)`），而 `add` 场景没有已有值，因此该参数无意义。一并更新签名与三个调用方。
4. **去掉 `app.go:1072` 处被丢弃的 `_ = keepCurrent`**：`promptEditValue` 在描述字段的分支返回了 `keepCurrent` 但未使用；改为以 `_` 接收返回值，删除多余的赋值与丢弃语句。

**验证**：删除与改动后 `go build ./... && go vet ./... && go test ./... -count=1` 全绿。

---

## 第二部分 — 拆分 `app.go`

将 `internal/cli/app.go`（1261 行）按职责拆为 6 个文件，全部仍属 `package cli`。这是纯代码搬移，函数签名不变，因此 `app_test.go` 无需改动即可通过。

| 文件 | 包含的标识符 | 约行数 |
|------|--------------|--------|
| `app.go` | `Paths`、`Run`、`defaultPaths`、`runCurrent`、`runList`、`runStatus`、`currentStatus`、`shouldRenderUnknownForProfileLoadError`、共享小工具 `normalizeProfileName`、`formatCLIError` | ~160 |
| `launch.go` | `runProfileCommand`、`parseProfileCommandArgs`、`switchProfile`、`runClaude`、`buildClaudeEnv` | ~90 |
| `commands.go` | `profileFlags`、`runAdd`、`runEdit`、`runEditWithPromptReader`、`parseProfileFlags`、`buildProfileEnv`、`runRemove`、`runRename`、`isReservedProfileName`、`isReservedCommandName` | ~200 |
| `prompt.go` | `promptAddName`、`promptRenameName`、`promptAddFields`、`promptEditFields`、`applyEditPrompt`、`promptAddValue`、`promptEditValue`、`maskValue` | ~170 |
| `interactive.go` | 注入用包级变量（`promptReader`、`promptWriter`、`promptInteractive`、`startInteractiveSession`、`launchClaude`）、`selectorAction` 类型与常量、ANSI 常量、`selectorInteractive`、`runInteractiveStatus`、`runInteractiveList`、`executeListAction`、`executeListDelete`、`resumeListSession`、`startInteractiveTerminalSession`、`reloadListMenu`、`readSelectorAction`、`menuHasMissingCurrentProfile` | ~330 |
| `display.go` | `profileNames`、`modeNames`、`availableNames`、`displayNamesForProfiles`、`profileDisplayName`、`profileListDisplayName`、`profileDescriptions`、`currentDescription`、`officialProfileDescription` | ~80 |

约束：
- 所有标识符必须恰好定义一次（搬移后不得出现重复定义或遗漏）。
- 不改任何函数签名、不改任何逻辑。
- `status_selector.go`、`list_menu.go`、`term_*.go`、`parse.go` 不在本次搬移内，保持原样。

**验证**：`app_test.go` 零改动；`go build ./... && go vet ./... && go test ./... -count=1` 全绿。

---

## 第三部分 — 修复 UI 文案冗余

统一以 `目标配置：` / `选择配置：` 作为唯一主题行，去掉重复展示同一名称的行。

**重要边界**：只改交互式渲染（`statusSelector.render()` 与 `listMenu.render()`）。非交互式输出 `output.RenderStatus` / `RenderList` 中的 `可用配置：` 保持不变——这是有意为之，非交互路径本就只有单个表头，不存在冗余。

### 3.1 status_selector（`internal/cli/status_selector.go`）

删除 `可用配置：` 行，保留 `选择配置：`（与 list_menu 一致）。

改动前：
```
...
可用配置：
选择配置：
> demo（当前）...
```
改动后：
```
...
选择配置：
> demo（当前）...
```

### 3.2 list_menu（`internal/cli/list_menu.go`）

actions 模式删除 `操作：X` 行、delete-confirm 模式删除 `确认删除：X` 行，两种模式均保留 `目标配置：X` 作为唯一主题行。

actions 改动后：
```
配置操作
目标配置：beta - 测试环境

可执行操作：
> 切换
...
```
delete-confirm 改动后：
```
删除配置
目标配置：beta - 测试环境
此操作不可恢复，请再次确认。

> 确认删除
  取消
```

### 3.3 需要同步更新的测试

只更新「断言交互式渲染」的用例；**非交互式 `RenderStatus`/`RenderList` 的整串期望值不动**（如 `app_test.go` 的 2071、2147、3461 行与 `output/print_test.go`）。

以下为起始检查清单，最终以运行测试后的失败项为准：
- `internal/cli/status_selector_test.go`：第 29–30、150–151 行——去掉对 `可用配置：` 的断言，保留 `选择配置：`。
- `internal/cli/app_test.go`：交互式状态流中断言 `可用配置：` 的用例（约 1783、1828、1948、1982 行）——改为 `选择配置：`（其中 1982 行为「断言不存在」的负向用例，需确认改为对 `选择配置：` 的相应断言）。
- `internal/cli/list_menu_test.go`：保留 `目标配置：` 的断言不变。
- `internal/cli/app_test.go`：断言 `操作：X` 的用例（约 2764、2858、2974、3103 行）改为 `目标配置：X`；断言 `确认删除：beta`（约 3298 行）改为 `目标配置：beta`。

**验证**：`go test ./... -count=1` 全绿。

---

## 实施顺序

按第一 → 第二 → 第三部分依次进行，每部分独立提交、独立验证（`go build && go vet && go test` 全绿）后再进入下一部分。重构（一、二部分）遵循「改动前后测试均通过」；UI 改动（三部分）先改渲染再同步断言。
