# neurolink

[English](README.md) | 简体中文

`neurolink` 是一个 Go 终端 UI 工具，用来在开玩或排查问题前观察 Apex Legends 相关服务是否健康，并支持显式触发的 Apex 玩家数据查询。界面主题是 Crypto 风格的“监控无人机”，重点展示服务可用性和开玩前的快速信息检查，而不是把占位 ICMP ping 伪装成真实 Apex 服务器监控，也不是后台跟踪玩家。

它关注这些服务和信号：

- Crossplay Auth
- Lobby / Matchmaking
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status API 健康度
- 如果状态 payload 提供，则展示最近用户反馈

玩家视图可以按玩家名和平台（`PC`、`PS4`、`X1`）通过 Apex Legends Status `/bridge` 查询玩家数据。查询只会在你在玩家视图中按下 Enter 等明确操作后执行。PC 查询通常使用 Origin 账号名，即使该账号通过 Steam 游玩。`neurolink` 不会连续跟踪玩家、后台查询比赛历史或隐藏遥测。

## 当前数据来源

主数据源是 Apex Legends Status 生态的公开站点和 API：

- 网站：`https://apexlegendsstatus.com/`
- API 文档与入口：`https://apexlegendsapi.com/`
- API 基础地址：`https://api.apexlegendsstatus.com/`

状态仪表盘会轮询：

```text
GET https://api.apexlegendsstatus.com/servers
```

玩家视图只在显式触发时查询：

```text
GET https://api.apexlegendsstatus.com/bridge?player=PLAYER_NAME&platform=PLATFORM
```

两个端点的实时数据都需要 Apex Legends Status API key。程序使用 `Authorization` header 传递 key，不会硬编码或记录示例真实 token。

注意：`/servers` 端点可能返回 `Content-Type: text/plain;charset=UTF-8`，但 body 内容是 JSON。因此客户端使用 `Accept: */*`，然后按 JSON 解码 body，避免因为严格要求 `application/json` 而被上游拒绝。

界面会把 `/servers` 响应归一化为核心服务卡片。顶部 Overall 只代表“可玩核心服务”的汇总状态；`Apex Legends Status API` 是第三方状态站/API 自身健康度，会单独显示，不会把状态站问题直接当成 Apex 核心服务宕机。

玩家查询会把 `/bridge` 响应归一化为身份、平台、UID、等级、排位、当前传奇和 trackers。它只做显式查询，不展示连续在线/大厅跟踪。

## 安装

在当前代码目录中安装：

```bash
go install .
```

也可以按模块路径安装：

```bash
go install github.com/Nesoriel/neurolink@latest
```

## 运行

运行 demo：

```bash
neurolink
```

使用真实 API 做一次性覆盖：

```bash
neurolink --api-key "$YOUR_APEX_STATUS_API_KEY" --poll-interval 1m
```

本地开发时也可以继续用 `go run .`：

```bash
go run . --demo
go run . --lang zh-Hans
```

## TUI 操作

- `tab` / `shift+tab`：切换视图
- `1`：状态仪表盘
- `2`：玩家查询
- `3`：配置
- `?`：帮助
- `/`：打开玩家查询并聚焦玩家名输入框
- `enter`：在玩家视图中执行查询
- `p`：玩家输入框未聚焦时切换 `PC` / `PS4` / `X1`
- `r`：立即请求刷新服务状态
- `q` / `ctrl+c`：退出

## 持久化配置

`neurolink` 会把可选的持久化配置保存到 Go `os.UserConfigDir()` 返回的跨平台用户配置目录：

```text
<user-config-dir>/neurolink/config.json
```

常见位置：Linux 上是 `~/.config/neurolink/config.json`，macOS 上是 `~/Library/Application Support/neurolink/config.json`，Windows 上是 `%AppData%\neurolink\config.json`。

查看当前配置文件路径：

```bash
neurolink config path
```

保存配置：

```bash
neurolink config set api-key "$YOUR_APEX_STATUS_API_KEY"
neurolink config set language en
neurolink config set language zh-Hans
neurolink config set poll-interval 30s
```

也可以一次保存多个配置项：

```bash
neurolink config set --api-key "$YOUR_APEX_STATUS_API_KEY" --lang zh-Hans --poll-interval 30s
```

查看已保存配置。API key 永远会被遮罩：

```bash
neurolink config show
```

删除已保存配置：

```bash
neurolink config unset api-key
neurolink config unset language poll-interval
neurolink config unset all
```

需要时可以指定自定义配置文件路径：

```bash
neurolink --config ./local.config.json config show
neurolink --config ./local.config.json
```

普通运行时配置优先级是：

```text
默认值 < 配置文件 < 环境变量 < 命令行参数
```

## 环境变量

- `NEUROLINK_APEX_API_KEY`：一次性 API key 覆盖
- `NEUROLINK_LANG`：一次性语言覆盖，支持 `en` 或 `zh-Hans`
- `NEUROLINK_POLL_INTERVAL`：一次性轮询间隔覆盖，例如 `30s` 或 `1m`
- `NEUROLINK_CONFIG`：配置文件路径覆盖

命令行参数仍然拥有最高优先级：

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" neurolink --poll-interval 1m
NEUROLINK_LANG=zh-Hans neurolink --lang en
```

## 无 API Key 时的行为

如果配置文件、环境变量和 `--api-key` 都没有提供 API key，程序会进入 demo 模式。也可以显式强制使用 demo：

```bash
neurolink --demo
```

Demo 模式使用确定性的示例数据，只用于展示界面和本地开发。仪表盘和玩家查询结果都会显示 `DEMO` 来源和演示提示，不会把示例数据伪装成实时 Apex 服务或玩家数据。

## 构建和测试

构建：

```bash
go build ./...
```

测试：

```bash
go test ./...
go test -race ./...
```

格式化：

```bash
gofmt -w .
```

## 用户反馈面板

右上角面板在英文界面中显示为 `Community Pulse`，在中文界面中显示为 `用户反馈`。如果 `/servers` payload 包含最近用户反馈，界面会展示简短摘要；如果没有相关字段，界面会说明当前 `/servers` 响应不一定包含 report feed，并保留 Apex Legends Status 的数据来源说明。

这个面板只提供服务健康参考。玩家数据只通过明确的玩家查询视图获取。

## 可选 ping 诊断

旧的 ICMP ping 代码保留为显式诊断模块，但不再是主监控数据源，也不会默认使用 TEST-NET 占位地址。需要做网络层诊断时，应显式提供目标，并把结果当作辅助信息，而不是 Apex 服务健康状态。

Linux 上运行 ICMP 诊断可能需要允许非特权 ping：

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

## 当前限制和 TODO

- Apex Legends Status API 的字段可能变化；当前 normalizer 对常见字段和核心服务别名做了兼容处理。
- 没有 API key 时只能看到 demo 数据。
- 最近用户反馈只有在 `/servers` payload 提供相关字段时才会展示。
- 玩家查询第一版支持玩家名 + 平台（`PC`、`PS4`、`X1`）；上游也有 UID 查询，但本轮 TUI 尚未暴露。
- 玩家视图汇总 `/bridge` 返回的资料，不做连续在线/大厅跟踪。
- ping 诊断还没有作为独立 TUI 面板暴露。
- 后续可以增加更多服务卡片、更丰富的状态历史、保存默认查询玩家/平台、更细粒度的桌面通知。
