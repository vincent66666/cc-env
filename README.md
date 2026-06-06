# cc-env

`cc-env` 是一个独立 CLI，用来管理 Claude Code 的第三方 API profile，并通过启动 `claude` 子进程时注入环境变量来切换模式。

它不再写入 `~/.claude/settings.json`，因此不会污染官方 Claude 登录态。官方登录态通过内置的 `official` 模式表示：启动前清理所有受管理的第三方路由变量，然后直接运行 `claude`。

## 功能概览

- 用 `~/.claude/cc-env/profiles.json` 保存多个第三方 API profile
- 用 `cc-env use <profile|official>` 保存当前模式并启动 `claude`
- 用 `cc-env use` 按当前模式启动；如果 current 为空，则默认使用 `official`
- 支持把 `--` 后的参数原样传给 `claude`
- 支持新增、编辑、删除、重命名第三方 profile
- `official` 是内置保留名，不能作为普通 profile 使用

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

### 查看状态

```bash
cc-env
```

示例输出：

```text
当前配置：deepseek - DeepSeek
接口地址：https://api.deepseek.com/anthropic
模型：deepseek-v4-pro
可用配置：official - 官方登录态
```

### 查看当前模式

```bash
cc-env current
```

示例输出：

```text
deepseek
```

### 列出模式

```bash
cc-env list
```

非交互输出按普通 profile 名称排序，并在末尾包含内置 `official`：

```text
deepseek - DeepSeek
official - 官方登录态
```

### 启动第三方 profile

```bash
cc-env use deepseek
```

传参给 Claude：

```bash
cc-env use deepseek -- --print "hello"
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
cc-env use official
```

`official` 模式会清理第三方路由变量后启动 `claude`，不会写入 token、base URL 或 model。

### 使用当前模式启动

```bash
cc-env use
```

如果 `current` 为空，默认按 `official` 启动。

## Profile 管理

新增 profile：

```bash
cc-env add deepseek \
  --description "DeepSeek" \
  --token "token-demo" \
  --base-url "https://api.deepseek.com/anthropic" \
  --model "deepseek-v4-pro" \
  --default-opus-model "deepseek-v4-pro" \
  --default-sonnet-model "deepseek-v4-pro" \
  --default-haiku-model "deepseek-v4-flash" \
  --subagent-model "deepseek-v4-flash" \
  --disable-nonessential-traffic \
  --disable-nonstreaming-fallback
```

编辑 profile：

```bash
cc-env edit deepseek --model "deepseek-v4-pro"
```

删除 profile：

```bash
cc-env remove deepseek
```

重命名 profile：

```bash
cc-env rename deepseek ds
```

限制：

- `official` 是内置模式，不能新增、编辑、删除或重命名
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
- [internal/cli/app.go](internal/cli/app.go)：命令分发、参数处理和 Claude 启动
- [internal/profile/store.go](internal/profile/store.go)：`profiles.json` 读写
- [internal/profile/validate.go](internal/profile/validate.go)：profile 校验规则
- [internal/settings/store.go](internal/settings/store.go)：旧 settings 写入实现，目前 CLI 不再调用
