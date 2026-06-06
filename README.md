# cc-env

`cc-env` 是一个独立 CLI，用来管理 Claude Code 的第三方 API profile，并通过启动 `claude` 子进程时注入环境变量来切换模式。

它不再写入 `~/.claude/settings.json`，因此不会污染官方 Claude 登录态。官方登录态通过内置的 `official` 模式表示：启动前清理所有受管理的第三方路由变量，然后直接运行 `claude`。

## 功能概览

- 用 `~/.claude/cc-env/profiles.json` 保存多个第三方 API profile
- 用 `cc-env <profile|official>` 保存当前模式并启动 `claude`
- 通过 `cc-env` 进入交互界面完成新增/编辑/删除/重命名/切换
- 支持把 profile 名后的参数原样传给 `claude`，也兼容用 `--` 显式分隔
- `official` 和 `current` 是保留名，不能作为普通 profile 使用

## 默认文件路径

- profiles 仓库：`~/.claude/cc-env/profiles.json`

也可以通过环境变量覆盖默认路径：

- `CC_ENV_PROFILES_PATH`

兼容旧变量 `CC_SWITCH_PROFILES_PATH`，但新配置优先使用 `CC_ENV_PROFILES_PATH`。

## 安装

```bash
go build -o cc-env .
```

构建完成后会得到当前目录下的 `cc-env` 可执行文件。

## 数据结构

`~/.claude/cc-env/profiles.json` 的结构如下：

```json
{
  "version": 1,
  "current": "deepseek",
  "profiles": {
    "deepseek": {
      "description": "DeepSeek",
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-demo",
        "ANTHROPIC_BASE_URL": "https://api.deepseek.com/anthropic",
        "ANTHROPIC_MODEL": "deepseek-v4-pro",
        "ANTHROPIC_DEFAULT_OPUS_MODEL": "deepseek-v4-pro",
        "ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-pro",
        "ANTHROPIC_DEFAULT_HAIKU_MODEL": "deepseek-v4-flash",
        "CLAUDE_CODE_SUBAGENT_MODEL": "deepseek-v4-flash",
        "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
        "CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK": "1"
      }
    }
  }
}
```

### 支持的字段

必填：

- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`

可选：

- `ANTHROPIC_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`
- `CLAUDE_CODE_SUBAGENT_MODEL`
- `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`
- `CLAUDE_CODE_DISABLE_NONSTREAMING_FALLBACK`

不在白名单中的字段不会被接受到 profile 中。

## 使用方法

### 交互界面

```bash
cc-env
```

无参数时进入 Bubble Tea 交互界面，可在此完成 profile 的选择、新增、编辑、删除等操作。

键位：

| 按键 | 功能 |
|------|------|
| `↑` / `↓` · `j` / `k` | 上下选择 |
| `Enter` | 切换到选中 profile 并启动 claude |
| `a` | 新建 profile |
| `e` | 编辑当前选中的 profile |
| `d` | 删除当前选中的 profile |
| `/` | 过滤 |
| `q` | 退出（不切换） |

> 非交互终端（管道/重定向）下，`cc-env` 打印当前配置状态后直接退出，不启动 claude。

### 查看当前模式

```bash
cc-env current
```

示例输出：

```text
deepseek
```

### 启动第三方 profile

```bash
cc-env deepseek
```

传参给 Claude：

```bash
cc-env deepseek --print "hello"
```

也可以显式使用 `--` 分隔：

```bash
cc-env deepseek -- --print "hello"
```

执行顺序：

1. 读取 `profiles.json`
2. 找到目标 profile
3. 更新 `profiles.json` 中的 `current`
4. 从当前进程环境中移除所有受管理变量
5. 写入目标 profile 的 env
6. 启动 `claude`

### 启动官方登录态

```bash
cc-env official
```

`official` 模式会清理第三方路由变量后启动 `claude`，不会写入 token、base URL 或 model。

## 限制

- `official` 是内置模式，不能新增、编辑、删除
- `current` 是保留名，不能作为普通 profile 名称
- 不能删除当前正在使用的普通 profile
- 第三方 profile 必须包含 token 和 base URL

## 开发与验证

运行测试：

```bash
go test ./... -count=1
```

构建：

```bash
go build ./...
```

## 文件说明

- [main.go](main.go)：CLI 入口
- [internal/cli/app.go](internal/cli/app.go)：命令分发与直达启动
- [internal/tui/](internal/tui/)：Bubble Tea 交互界面
- [internal/profile/store.go](internal/profile/store.go)：`profiles.json` 读写
- [internal/profile/validate.go](internal/profile/validate.go)：profile 校验规则
