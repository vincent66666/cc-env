# 统一交互 TUI 重设计

- 日期：2026-06-06
- 状态：已确认，待实现
- 范围：cc-env 交互层全面重构

## 背景与目标

cc-env 当前有**两套并行的交互式 TUI**：`cc-env status`（只读选择 + 切换）和 `cc-env list`（完整 CRUD）。两者都渲染配置列表、都在 Enter 时切换并启动 `claude`，概念高度重叠；新增/编辑配置时还会**跳出 alt-screen、退回 cooked-mode 逐行读取输入**（token 明文回显），体验割裂、代码分散。

本次目标：

1. **合并为单一 TUI**。输入 `cc-env`（无参数）进入一个页面，在同一页完成 选择 / 切换 / 新建 / 编辑 / 重命名 / 删除。
2. **删除** `list / add / edit / remove / rename / status` 子命令。
3. 保留快捷直达：`cc-env <profile>` 与 `cc-env official` 仍直接切换并启动 `claude`。
4. 保留 `cc-env current` 作为唯一的非交互、可脚本化命令。
5. 引入 **Bubble Tea + Bubbles + Lip Gloss** 作为 TUI 框架。

## 非目标

- 不改动 profile 的数据结构、env 字段白名单、持久化格式（`profiles.json` 不变）。
- 不改动 `claude` 启动时的环境变量清理/覆盖逻辑（`buildClaudeEnv`）。
- 不引入除 Bubble Tea 全家桶以外的额外依赖。
- 不做配置导入/导出、多语言、远程同步等新功能。

## 关键决策

| 决策 | 结论 |
|---|---|
| TUI 框架 | **Bubble Tea + Bubbles + Lip Gloss**，放弃 zero-dependency 原则 |
| `cc-env`（无参） | **进入 TUI**，不再以 official 启动 `claude`（破坏性变更） |
| TUI 内 Enter | 切换并**直接 exec `claude`** |
| 子命令 | `list/add/edit/remove/rename/status` **直接删除**，无兼容过渡 |
| `cc-env current` | **保留**（脚本/状态栏用途，TUI 无法替代） |
| `cc-env <profile>` / `official` | **保留**直达启动，`--` 透传 claude 参数保留 |
| 重命名 | **并入编辑表单**（名称字段可改），不再有独立 rename 动作 |
| 非 TTY 跑 `cc-env` | 打印非交互状态（当前 + 可用配置）后退出 0，不启动 claude |

### 放弃 zero-dependency 的理由

本次要做"单页 + 内建多字段表单 + token 遮罩 + 模糊搜索 + 预览"，这恰是 stdlib 最吃亏、Bubbles 现成组件最省事的场景：`list`（过滤/分页）、`textinput`（光标/`EchoPassword` 遮罩）、`viewport`（长内容）直接可用，省掉项目里最易出 bug 的手写输入/遮罩代码。代价是引入 `bubbletea + bubbles + lipgloss` 及其约 6–8 个间接依赖、二进制体积 +2~4MB。README / CLAUDE.md 需移除 zero-dependency 卖点。

## 架构分层

```
main.go                      → cli.Run()
internal/
  cli/        ← 瘦身为「参数分发 + 直达启动 + current + 非 TTY 兜底」
    app.go        Run() 分发：无参→tui.Run；<profile>/official→switch+launch；current→print
    launch.go     switchProfile / buildClaudeEnv / runClaude   （保留，TUI 与直达复用）
    parse.go      （保留）
  tui/        ← 新增：Bubble Tea 应用，单页交互
    app.go        tui.Run(paths) → 建 program、运行、返回用户选中的启动目标
    model.go      根 Model + 状态机（state 枚举）
    list.go       列表视图（bubbles/list，自带过滤）
    form.go       新建/编辑表单（bubbles/textinput，token 遮罩）
    preview.go    右侧预览面板（lipgloss 渲染）
    confirm.go    删除确认（内联 yes/no）
    keys.go       键位绑定
    styles.go     lipgloss 主题
  profile/    ← 几乎不动，作为持久化/校验核心被 tui 复用
  output/     ← 保留 current 与非 TTY 兜底所需的渲染
```

**删除文件**：`cli/status_selector.go`、`cli/list_menu.go`、`cli/interactive.go`、`cli/prompt.go`、`cli/commands.go`（add/edit/remove/rename 及 flag 解析整块）、`cli/term_darwin.go`、`cli/term_other.go`（raw-mode / alt-screen / resize 交由 Bubble Tea 处理）。对应测试文件（`list_menu_test.go`、`status_selector_test.go` 等）随之删除。

## TUI 设计

### 状态机

根 `Model` 用单一 `state` 枚举驱动整页：

| state | 组件 | 说明 |
|---|---|---|
| `stateList` | `bubbles/list` + 右侧 preview | 默认页：左列表、右预览 |
| `stateForm` | 一组 `bubbles/textinput` | 新建 / 编辑共用；token 用 `EchoPassword` 遮罩 |
| `stateConfirm` | 内联 yes/no | 删除确认 |

`Update` 是纯函数 `(Model, Msg) → (Model, Cmd)`：所有"列表态 / 表单态 / 确认态"切换都是改 `state` 字段，不再手动进出 raw-mode。

### 单页布局（list 态）

```
 cc-env
 ┌── 配置 ───────────────┬── 预览 ──────────────┐
 │ > deepseek (当前)     │ 名称   deepseek      │
 │   kimi                │ 描述   DeepSeek      │
 │   official            │ base   api.deep...   │
 │                       │ model  v4-pro        │
 │ filter: ___           │ opus   v4-pro        │
 │                       │ token  de******xy    │
 ├───────────────────────┴──────────────────────┤
 │ Enter 切换并启动  a 新建  e 编辑              │
 │ d 删除  / 过滤  q 退出                        │
 └──────────────────────────────────────────────┘
```

- 列表顺序：当前配置置顶（沿用 `prioritizeCurrentProfile`），其余 profile，再 `official`。
- 预览面板展示高亮项的字段；token 经遮罩（沿用 `maskValue` 等价逻辑）。
- 新建/编辑时右侧预览区切换为 `textinput` 表单视图。

### 键位（list 态）

`↑/↓`·`j/k` 移动 ｜ `/` 过滤 ｜ `Enter` 切换并启动 ｜ `a` 新建 ｜ `e` 编辑（含改名） ｜ `d` 删除 ｜ `q`·`Ctrl+C` 退出。

### 行为规则（沿用现状）

- `official` 不可编辑 / 删除 / 重命名（只能被选中切换）。
- 当前正在使用的配置不可删除（`profile.Remove` 已内置拦截，TUI 同时在 UI 上禁用 `d`）。
- 表单字段：名称、描述、`ANTHROPIC_AUTH_TOKEN`(遮罩)、`ANTHROPIC_BASE_URL`、`ANTHROPIC_MODEL`、`ANTHROPIC_DEFAULT_OPUS_MODEL`、`ANTHROPIC_DEFAULT_SONNET_MODEL`、`ANTHROPIC_DEFAULT_HAIKU_MODEL`、`CLAUDE_CODE_SUBAGENT_MODEL`，以及 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` / `CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK` 两个布尔开关。
- 必填：名称、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_BASE_URL`。
- 编辑保存时若名称改变：按 `profile.Rename` 的等价语义迁移（map key 迁移 + 若是当前配置则同步 `current` 指针），并校验新名称非 `official`/`current`/重复。

## 数据流

```
cc-env（无参）
  └─ cli.Run → tui.Run(paths)
       ├─ profile.LoadForList → 构建列表（文件缺失则空列表 + official）
       ├─ 用户操作：
       │    a / e → 表单 → profile.Save → 重载列表（留在 TUI）
       │    d     → 确认 → profile.Remove → 重载列表
       │    Enter → 记录 target，tea.Quit
       └─ 返回 (target, launch=true)
  └─ cli 拿到 target → switchProfile(target) → 持久化 current + exec claude
```

**关键约束**：exec `claude` 在 **Bubble Tea 退出之后**由 cli 层执行，不在 tea 运行时内 exec，避免终端状态残留。`tui.Run` 只负责交互，返回"启动哪个 profile（或纯退出）"。

`tui.Run` 签名（草案）：返回选中的目标名与是否启动，例如 `(target string, launch bool, err error)`；`launch == false` 表示用户直接退出。

## 命令面（改造后）

| 命令 | 行为 |
|---|---|
| `cc-env` | 进入 TUI（不启动 claude；非 TTY 时打印状态） |
| `cc-env <profile>` / `cc-env official` | 直达：切换 + 启动 claude（保留 `--` 透传） |
| `cc-env current` | 打印当前 profile 名（保留，脚本用） |
| `cc-env list/add/edit/remove/rename/status` | 删除 |

**保留名收敛**：删命令后，`list/add/...` 等名字释放，可作普通 profile 名；**仅 `official` 与 `current` 仍为保留名**（`current` 因仍是命令、避免 `cc-env current` 歧义）。原 `isReservedCommandName` 收敛为只判这两个。

## 错误处理与边界

- **非 TTY**（管道 / 重定向）跑 `cc-env`：无法起 TUI，打印非交互状态（当前配置 + 可用配置列表）后退出 0，不启动 claude。复用 `output` 包既有渲染。
- **profiles.json 缺失**：`LoadForList` 返回空文件结构，TUI 正常进入，仅含 `official` 与空 profile 列表，可直接 `a` 新建。
- **表单校验失败**（缺 token/base-url、名称重复或为保留名）：`profile.ValidateProfile` 报错 → 表单内联提示，保留已输入内容，不丢失。
- **保存 / 加载 IO 错误**：TUI 顶部错误条提示，不崩溃；致命错误退出 TUI 并向 stderr 输出。
- **TTY 判定**：复用现有 `selectorInteractive` 同款判据（stdin 与 stdout 均为字符设备）。

## 测试策略

- **`profile` 包**：核心逻辑未变，测试基本不动。
- **`tui` 包**：`Update` 为纯函数，按"喂 `tea.KeyMsg` → 断言 `state`/选中项/`View()` 文本"写单测，覆盖：状态转移、键位、过滤、表单字段导航、表单校验失败、Enter 返回的 target、official/当前配置的禁用规则。
- **`cli` 包**：保留并补充 `current`、直达启动、非 TTY 兜底的测试；**删除**针对已移除子命令（add/edit/remove/rename/status/list）的大量用例（当前 `app_test.go` 约 3700 行，多数随命令删除而清理）。

## 影响面与迁移

- `go.mod` / `go.sum`：新增 `github.com/charmbracelet/bubbletea`、`.../bubbles`、`.../lipgloss` 及其间接依赖。
- `README.md`：移除 zero-dependency 表述，更新功能概览、命令清单、交互说明（无参进 TUI、子命令已删、`current` 保留）。
- `CLAUDE.md`：更新 Project 说明（去掉 stdlib-only）、Commands、Architecture 目录树（新增 `internal/tui/`，删除已移除文件）、Key Design Details。
- 删除上述 `cli/*.go` 与对应测试，新增 `internal/tui/` 全套。

## 待实现时确认的细节（非阻塞）

- `tui.Run` 的精确签名与返回结构（target / launch / err 的具体形态）在实现时定稿。
- 预览面板用 `viewport` 还是静态 lipgloss 盒子：profile 字段短，先用静态盒子，超长再换 `viewport`。
- 删除确认是内联行还是 `lipgloss` 弹窗：先内联，足够即可。
- 主题配色细节（`styles.go`）实现时定。
