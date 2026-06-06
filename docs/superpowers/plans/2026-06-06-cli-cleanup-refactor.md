# cc-env CLI 清理与重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除死代码、把 1261 行的 `internal/cli/app.go` 拆成职责单一的文件、并去除交互界面的文案冗余，全程不改变对外行为（UI 去重除外）。

**Architecture:** 纯整理性改动。第一、二部分删代码并以 `go build/vet/test` 验证不回归；第三部分拆分 `app.go`（同包内搬移函数，签名不变，测试零改动）；第四、五部分修改交互渲染并同步更新断言。

**Tech Stack:** Go 1.23，标准库，零外部依赖。测试为 `go test`，部分交互测试依赖系统 `script` 命令分配 PTY（无 `script` 时自动 skip）。

对应设计文档：`docs/superpowers/specs/2026-06-06-cli-cleanup-refactor-design.md`

---

## 前置说明（重要）

当前工作区在分支 `chore/cli-cleanup` 上，存在**先前遗留的未提交改动**，它们完成了 `cc-env use <profile>` → `cc-env <profile>` 的重命名（`app.go`、`app_test.go`、`parse.go`、`CLAUDE.md`、`README.md`）。这些改动不是本计划产生的，但测试通过、逻辑自洽。

`Task 1` 会再次修改 `CLAUDE.md`，`Task 2/3` 会再次修改 `app.go`，若不先提交这些遗留改动，后续清理 commit 会与之纠缠。因此 `Task 0` 先把它们固化为基线 commit。**执行 `Task 0` 前需向用户确认这些遗留改动确实是预期的**（它们实现的是 `cc-env <profile>` 直接启动语义）。

每个 Task 结束时的统一验证命令：

```bash
go build ./... && go vet ./... && go test ./... -count=1
```

---

## Task 0: 固化遗留重命名改动为基线 commit

**Files:**
- Modify（仅提交，不编辑）：`internal/cli/app.go`、`internal/cli/app_test.go`、`internal/cli/parse.go`、`CLAUDE.md`、`README.md`

- [ ] **Step 1: 确认遗留改动内容**

Run: `git status --short && git diff --stat`
Expected: 列出上述 5 个文件为已修改（`docs/` 下的 spec/plan 已是干净的提交）。向用户确认这些改动是预期的 `cc-env <profile>` 重命名收尾。

- [ ] **Step 2: 验证当前工作区测试通过**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: 全部 PASS（`cc-env/internal/cli`、`/output`、`/profile`、`/settings` 均 ok）。

- [ ] **Step 3: 提交基线**

```bash
git add internal/cli/app.go internal/cli/app_test.go internal/cli/parse.go CLAUDE.md README.md
git commit -m "refactor(cli): 完成 use→<profile> 直接启动语义的收尾改动"
```

- [ ] **Step 4: 确认工作区干净**

Run: `git status --short`
Expected: 无输出（工作区干净）。

---

## Task 1: 删除死代码包 `internal/settings`

`internal/settings` 自 b03a41a 改为子进程启动后已无任何生产代码引用，CLAUDE.md 也标注其为 legacy。

**Files:**
- Delete: `internal/settings/store.go`
- Delete: `internal/settings/backup.go`
- Delete: `internal/settings/store_test.go`
- Modify: `CLAUDE.md`（架构图中删除 `settings/` 一行）

- [ ] **Step 1: 再次确认无引用**

Run: `grep -rn "internal/settings" --include="*.go" .`
Expected: 无输出（确认零引用）。

- [ ] **Step 2: 删除整个包目录**

```bash
git rm internal/settings/store.go internal/settings/backup.go internal/settings/store_test.go
```

- [ ] **Step 3: 更新 CLAUDE.md 架构图**

在 `CLAUDE.md` 的架构图中删除这一行：

```
  settings/          → Legacy settings.json integration, no longer called by CLI launch flow
```

- [ ] **Step 4: 验证构建与测试**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: 全部 PASS；输出中不再出现 `cc-env/internal/settings`。

- [ ] **Step 5: 提交**

```bash
git add -A
git commit -m "chore: 删除无人引用的 legacy settings 包"
```

---

## Task 2: 删除零散死代码

三处：未被调用的 `styler.muted()`（及其专用常量 `ansiMuted`）、`promptAddValue` 恒为 `false` 且被忽略的 `sensitive` 参数、被丢弃的 `_ = keepCurrent`。

**Files:**
- Modify: `internal/output/style.go`
- Modify: `internal/cli/app.go`

- [ ] **Step 1: 删除 `styler.muted()` 与 `ansiMuted`**

在 `internal/output/style.go` 中删除整个方法：

```go
func (s styler) muted(text string) string {
	if !s.enabled {
		return text
	}

	return ansiMuted + text + ansiReset
}
```

并在常量块中删除这一行：

```go
	ansiMuted   = "\x1b[90m"
```

- [ ] **Step 2: 删除 `promptAddValue` 的 `sensitive` 参数**

在 `internal/cli/app.go` 中，将函数签名与首行从：

```go
func promptAddValue(reader *bufio.Reader, label, suffix string, required, sensitive bool) (string, error) {
	_ = sensitive
	_, _ = fmt.Fprintf(promptWriter, "%s%s：", label, suffix)
```

改为：

```go
func promptAddValue(reader *bufio.Reader, label, suffix string, required bool) (string, error) {
	_, _ = fmt.Fprintf(promptWriter, "%s%s：", label, suffix)
```

- [ ] **Step 3: 更新 `promptAddValue` 的全部调用方**

`promptAddName`：

```go
func promptAddName(reader *bufio.Reader) (string, error) {
	return promptAddValue(reader, "名称", "", true)
}
```

`promptRenameName`：

```go
func promptRenameName(reader *bufio.Reader) (string, error) {
	return promptAddValue(reader, "新名称", "", true)
}
```

`promptAddFields` 中 7 处调用各去掉末尾的 `, false`（即去掉 `sensitive` 实参），改为：

```go
	if input.description == "" {
		input.description, err = promptAddValue(reader, "描述", "（可选）", false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.token == "" {
		input.token, err = promptAddValue(reader, profile.EnvAuthToken, "", true)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.baseURL == "" {
		input.baseURL, err = promptAddValue(reader, profile.EnvBaseURL, "", true)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.model == "" {
		input.model, err = promptAddValue(reader, "ANTHROPIC_MODEL", "（可选）", false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultOpus == "" {
		input.defaultOpus, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_OPUS_MODEL", "（可选）", false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultSonnet == "" {
		input.defaultSonnet, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_SONNET_MODEL", "（可选）", false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultHaiku == "" {
		input.defaultHaiku, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_HAIKU_MODEL", "（可选）", false)
		if err != nil {
			return profileFlags{}, err
		}
	}
```

- [ ] **Step 4: 删除被丢弃的 `keepCurrent`**

在 `promptEditFields` 的描述分支，将：

```go
	if input.description == "" {
		var keepCurrent bool
		existing.Description, keepCurrent, err = promptEditValue(reader, "描述", existing.Description, false, false)
		if err != nil {
			return profile.Profile{}, err
		}
		_ = keepCurrent
	} else {
```

改为：

```go
	if input.description == "" {
		existing.Description, _, err = promptEditValue(reader, "描述", existing.Description, false, false)
		if err != nil {
			return profile.Profile{}, err
		}
	} else {
```

- [ ] **Step 5: 验证构建与测试**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: 全部 PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/output/style.go internal/cli/app.go
git commit -m "chore(cli): 清理未使用的 muted 样式、sensitive 参数与丢弃变量"
```

---

## Task 3: 拆分 `internal/cli/app.go`

把 `app.go` 按职责拆成 6 个文件，全部仍是 `package cli`。**这是逐字搬移**：被移动的函数体一字不改，只是换到新文件，并为每个新文件补上正确的 `package cli` 头与 import 块。`app_test.go` 不改动。

每个新文件创建后立即 `go build ./internal/cli/`，确认无「重复定义 / 未定义 / 未使用 import」错误，再进行下一个。

**Files:**
- Create: `internal/cli/display.go`
- Create: `internal/cli/prompt.go`
- Create: `internal/cli/commands.go`
- Create: `internal/cli/launch.go`
- Create: `internal/cli/interactive.go`
- Modify: `internal/cli/app.go`（移走上述内容后，保留 dispatch 与只读命令）

各文件应包含的标识符与所需 import：

| 新文件 | 标识符 | import |
|--------|--------|--------|
| `display.go` | `profileNames`、`modeNames`、`availableNames`、`displayNamesForProfiles`、`profileDisplayName`、`profileListDisplayName`、`profileDescriptions`、`currentDescription`、`officialProfileDescription` | `sort`、`strings`、`cc-env/internal/profile` |
| `prompt.go` | `promptAddName`、`promptRenameName`、`promptAddFields`、`promptEditFields`、`applyEditPrompt`、`promptAddValue`、`promptEditValue`、`maskValue` | `bufio`、`fmt`、`strings`、`cc-env/internal/profile` |
| `commands.go` | `profileFlags`、`runAdd`、`runEdit`、`runEditWithPromptReader`、`parseProfileFlags`、`buildProfileEnv`、`runRemove`、`runRename`、`isReservedProfileName`、`isReservedCommandName` | `bufio`、`flag`、`fmt`、`io`、`os`、`strings`、`cc-env/internal/profile` |
| `launch.go` | `runProfileCommand`、`parseProfileCommandArgs`、`switchProfile`、`runClaude`、`buildClaudeEnv` | `errors`、`fmt`、`io`、`os`、`os/exec`、`strings`、`cc-env/internal/profile` |
| `interactive.go` | 包级变量（`promptReader`、`promptWriter`、`promptInteractive`、`startInteractiveSession`、`launchClaude`）、`selectorAction` 类型与其常量、屏幕控制常量（`clearScreenSequence`、`enterAlternateScreenMode`、`exitAlternateScreenMode`）、`selectorInteractive`、`runInteractiveStatus`、`runInteractiveList`、`executeListAction`、`executeListDelete`、`resumeListSession`、`startInteractiveTerminalSession`、`reloadListMenu`、`readSelectorAction`、`menuHasMissingCurrentProfile` | `bufio`、`errors`、`fmt`、`io`、`os`、`cc-env/internal/profile` |
| `app.go`（保留） | `Paths`、`Run`、`defaultPaths`、`runCurrent`、`runList`、`runStatus`、`currentStatus`、`shouldRenderUnknownForProfileLoadError`、`normalizeProfileName`、`formatCLIError` | `errors`、`fmt`、`io`、`os`、`strings`、`cc-env/internal/output`、`cc-env/internal/profile` |

- [ ] **Step 1: 创建 `display.go`**

新建文件，写入 `package cli` 头 + 上表所列 import + 从 `app.go` 剪切的 9 个展示辅助函数（逐字）。随后从 `app.go` 删除这 9 个函数。

Run: `go build ./internal/cli/`
Expected: 编译通过，无重复定义。

- [ ] **Step 2: 创建 `prompt.go`**

新建文件，`package cli` + import + 从 `app.go` 剪切的 8 个 prompt 相关函数（逐字，含 Task 2 已清理后的版本）。随后从 `app.go` 删除它们。

Run: `go build ./internal/cli/`
Expected: 编译通过。

- [ ] **Step 3: 创建 `commands.go`**

新建文件，`package cli` + import + 从 `app.go` 剪切 `profileFlags` 结构体及 add/edit/remove/rename 相关函数（逐字）。随后从 `app.go` 删除它们。

Run: `go build ./internal/cli/`
Expected: 编译通过。

- [ ] **Step 4: 创建 `launch.go`**

新建文件，`package cli` + import + 从 `app.go` 剪切 5 个启动相关函数（逐字）。随后从 `app.go` 删除它们。

Run: `go build ./internal/cli/`
Expected: 编译通过。

- [ ] **Step 5: 创建 `interactive.go`**

新建文件，`package cli` + import + 从 `app.go` 剪切包级变量块、`selectorAction` 类型与常量、屏幕控制常量、以及全部交互循环函数（逐字）。随后从 `app.go` 删除它们。

此时 `app.go` 应仅剩 `Paths`、`Run`、`defaultPaths`、`runCurrent`、`runList`、`runStatus`、`currentStatus`、`shouldRenderUnknownForProfileLoadError`、`normalizeProfileName`、`formatCLIError`，并将 import 块精简为上表 `app.go` 行所列（删除 `bufio`、`flag`、`os/exec`、`sort`）。

Run: `go build ./internal/cli/`
Expected: 编译通过。

- [ ] **Step 6: 全量验证**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: 全部 PASS。`app_test.go` 未改动仍通过，证明行为未变。

- [ ] **Step 7: 检查每个文件无遗漏/无重复**

Run: `gofmt -l internal/cli/ && grep -rn "^func \|^type \|^var \|^const " internal/cli/*.go | grep -v _test | wc -l`
Expected: `gofmt -l` 无输出（格式正确）；声明计数与拆分前一致（无函数丢失或重复）。

- [ ] **Step 8: 提交**

```bash
git add internal/cli/
git commit -m "refactor(cli): 拆分 app.go 为 launch/commands/prompt/interactive/display"
```

---

## Task 4: 去除 status_selector 文案冗余

交互式状态选择器同时打印 `可用配置：` 和 `选择配置：` 两个相邻表头，去掉前者、保留后者（与 list_menu 一致）。非交互式 `output.RenderStatus` 不动。

**Files:**
- Modify: `internal/cli/status_selector.go`
- Test: `internal/cli/status_selector_test.go`
- Test: `internal/cli/app_test.go`

- [ ] **Step 1: 先改单元测试，制造失败（红）**

在 `internal/cli/status_selector_test.go` 的 `TestStatusSelectorRender` 片段切片中删除 `"可用配置：",` 这一行，并在该测试的断言循环之后追加一条反向断言：

```go
	if strings.Contains(rendered, "可用配置：") {
		t.Fatalf("expected selector to drop the redundant 可用配置 header, got %q", rendered)
	}
```

在 `TestStatusSelectorRenderShowsStructuredHeaderAndHints` 片段切片中同样删除 `"可用配置：",` 这一行。

- [ ] **Step 2: 同步更新 app_test.go 中三处交互断言（仍为红）**

在 `internal/cli/app_test.go` 中，把以下三处的 `"可用配置："` 改为 `"选择配置："`（均为交互式 PTY 流程的正向断言）：

- `TestRun_StatusInteractive...`（约第 1783 行）：
  `if !strings.Contains(output, "可用配置：") || !strings.Contains(output, "> prod") {` → 将 `"可用配置："` 改为 `"选择配置："`
- 约第 1828 行：`if !strings.Contains(output, "可用配置：") {` → `"选择配置："`
- 约第 1948 行：`if !strings.Contains(output, "可用配置：") {` → `"选择配置："`

**不要改**约第 1982 行 `if strings.Contains(output, "可用配置：")`（该用例为「无其他配置」的非交互空列表负向断言，保持原样）。

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/cli/ -run 'TestStatusSelector' -count=1`
Expected: FAIL —新增的反向断言失败，因为生产代码仍输出 `可用配置：`。

- [ ] **Step 4: 修改生产渲染代码**

在 `internal/cli/status_selector.go` 的 `render()` 中，将：

```go
	out.WriteString("\n可用配置：\n")
	out.WriteString("选择配置：\n")
```

改为：

```go
	out.WriteString("\n选择配置：\n")
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/cli/ -count=1`
Expected: PASS（单元测试 + PTY 集成测试均通过；若机器无 `script` 命令，相关集成用例 skip）。

- [ ] **Step 6: 提交**

```bash
git add internal/cli/status_selector.go internal/cli/status_selector_test.go internal/cli/app_test.go
git commit -m "fix(cli): 去除状态选择器重复的可用配置表头"
```

---

## Task 5: 去除 list_menu 文案冗余

actions 模式同时打印 `操作：X` 和 `目标配置：X`、delete-confirm 模式同时打印 `确认删除：X` 和 `目标配置：X`，均去掉前者、保留 `目标配置：X`。`list_menu_test.go` 已只断言 `目标配置：`（用 `Contains`），无需改动；需更新的是 `app_test.go` 中断言 `操作：`/`确认删除：` 的集成用例。

**Files:**
- Modify: `internal/cli/list_menu.go`
- Test: `internal/cli/list_menu_test.go`
- Test: `internal/cli/app_test.go`

- [ ] **Step 1: 先在单元测试加反向断言，制造失败（红）**

在 `internal/cli/list_menu_test.go` 的 `TestListMenuEnterActionsAndBack` 断言循环之后追加：

```go
	if strings.Contains(rendered, "操作：beta") {
		t.Fatalf("expected actions view to drop the redundant 操作 line, got %q", rendered)
	}
```

在 `TestListMenuEnterDeleteConfirmAndCancel` 断言循环之后追加：

```go
	if strings.Contains(rendered, "确认删除：beta") {
		t.Fatalf("expected delete view to drop the redundant 确认删除 line, got %q", rendered)
	}
```

- [ ] **Step 2: 运行单元测试确认失败**

Run: `go test ./internal/cli/ -run 'TestListMenu(EnterActionsAndBack|EnterDeleteConfirmAndCancel)' -count=1`
Expected: FAIL —两条反向断言失败，因为生产代码仍输出 `操作：beta` 与 `确认删除：beta`。

- [ ] **Step 3: 修改生产渲染代码**

在 `internal/cli/list_menu.go` 的 `render()` 中，delete-confirm 分支由：

```go
		out.WriteString("删除配置\n")
		out.WriteString("确认删除：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("此操作不可恢复，请再次确认。\n\n")
```

改为：

```go
		out.WriteString("删除配置\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("此操作不可恢复，请再次确认。\n\n")
```

actions 分支由：

```go
		out.WriteString("配置操作\n")
		out.WriteString("操作：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n\n")
		out.WriteString("可执行操作：\n")
```

改为：

```go
		out.WriteString("配置操作\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n\n")
		out.WriteString("可执行操作：\n")
```

- [ ] **Step 4: 更新 app_test.go 集成断言**

在 `internal/cli/app_test.go` 中：

- 约第 2764 行：`if !strings.Contains(output, "配置列表：") || !strings.Contains(output, "操作：prod") {` → 将 `"操作：prod"` 改为 `"目标配置：prod"`
- 约第 2814 行（负向，验证降级流程不进入操作菜单）：`if strings.Contains(output, "操作：") {` → 改为 `if strings.Contains(output, "可执行操作：") {`（`可执行操作：` 是操作菜单专有表头，去除冗余后仍存在，能继续表达「未进入操作菜单」）
- 约第 2858 行：`if !strings.Contains(output, "操作：demo") {` → `"目标配置：demo"`
- 约第 2974 行：`if !strings.Contains(output, "操作：demo") {` → `"目标配置：demo"`
- 约第 3103 行：`if !strings.Contains(output, "操作：beta") || !strings.Contains(output, "已更新配置：beta") {` → 将 `"操作：beta"` 改为 `"目标配置：beta"`
- 约第 3298 行：`if !strings.Contains(output, "确认删除：beta") || !strings.Contains(output, "已删除配置：beta") {` → 将 `"确认删除：beta"` 改为 `"目标配置：beta"`

- [ ] **Step 5: 全量验证**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: 全部 PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/cli/list_menu.go internal/cli/list_menu_test.go internal/cli/app_test.go
git commit -m "fix(cli): 去除配置操作/删除确认界面重复的主题行"
```

---

## 完成标准

- `go build ./... && go vet ./... && go test ./... -count=1` 全绿。
- `internal/settings/` 不复存在；`grep -rn "internal/settings" .` 仅可能命中文档/历史，不命中 `.go`。
- `internal/cli/app.go` 显著变小（约 150 行），其余职责分散在 launch/commands/prompt/interactive/display 各文件。
- `styler.muted`、`ansiMuted`、`promptAddValue` 的 `sensitive` 参数、`_ = keepCurrent` 均已移除。
- 交互式状态/列表界面不再出现重复表头与重复名称行。

## 自查结果（Self-Review）

- **Spec 覆盖**：spec 第一部分 → Task 1+2；第二部分 → Task 3；第三部分 → Task 4（status_selector）+ Task 5（list_menu）。遗留改动处理 → Task 0。无遗漏。
- **占位符**：无 TBD/TODO；所有改动均给出确切代码或确切 old→new 文本。行号标注为「约第 N 行」，因 Task 0–3 不修改 `app_test.go`，其行号在进入 Task 4/5 时保持稳定；最终以匹配文本为准。
- **类型/命名一致性**：`promptAddValue` 去参后签名 `(reader, label, suffix, required)` 在定义与全部调用处一致；新文件的函数名与 import 与现有代码一致；`可执行操作：`/`目标配置：`/`选择配置：` 字样与 list_menu/status_selector 现有渲染一致。
