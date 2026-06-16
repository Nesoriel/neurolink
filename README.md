# neurolink

`neurolink` 是一个 Go 终端 UI 工具，用来在开玩或排查问题前观察 Apex Legends
相关服务是否健康。界面主题是 Crypto 风格的“监控无人机”，重点展示服务状态，而不是伪装成真实 Apex
服务器的 ICMP ping。

## 当前数据来源

主数据源是 Apex Legends Status 生态的服务状态接口：

- 网站参考：`https://apexlegendsstatus.com/`
- API 文档与入口：`https://apexlegendsapi.com/`
- API 基础地址：`https://api.apexlegendsstatus.com/`

当前实现会轮询 `GET /servers`，也就是官方文档中的服务器状态接口：

```text
https://api.apexlegendsstatus.com/servers
```

注意：该端点实际返回的是 `Content-Type: text/plain;charset=UTF-8`，但 body 内容是 JSON。因此客户端会使用 `Accept: */*` 并按 JSON 解码，避免上游返回 `406 Not Acceptable`。

程序只关注服务器/服务可用性，不查询玩家资料、战绩、UID 或用户状态。

响应会被归一化为这些服务卡片：

- Crossplay Auth
- Lobby / Matchmaking
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status API

顶部总览的 `Overall` 只代表前四项“游戏核心可玩服务”的状态。`Apex Legends Status API` 是第三方状态站/API 自身健康度，会单独显示，但不会把大厅、匹配、登录等核心服务误判为不可用。

如果 API 响应中包含最近用户报告，界面会展示简短摘要；如果没有该字段，界面会明确说明当前 payload
没有报告 feed。

## API Key 配置

Apex Legends Status API 通常需要 API key。`neurolink` 不会硬编码 key，也不会提交示例真实 token。

可以用命令行参数：

```bash
go run . --api-key "$YOUR_APEX_STATUS_API_KEY"
```

也可以用环境变量：

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" go run .
```

常用参数：

```bash
go run . --api-key "$YOUR_APEX_STATUS_API_KEY" --poll-interval 30s
go run . --demo
```

## 无 API Key 时的行为

如果没有 `--api-key`，并且没有设置 `NEUROLINK_APEX_API_KEY`，程序会进入 demo 模式。

Demo 模式使用确定性的示例数据，只用于展示界面和本地开发。界面会显示 `DEMO` 来源和 demo 提示，不会把示例数据伪装成实时 Apex 服务状态。

## 运行、构建、测试

运行：

```bash
go run .
```

使用真实 API：

```bash
NEUROLINK_APEX_API_KEY="$YOUR_APEX_STATUS_API_KEY" go run . --poll-interval 1m
```

构建：

```bash
go build ./...
```

测试：

```bash
go test ./...
```

格式化：

```bash
gofmt -w .
```

## 可选 ping 诊断

旧的 ICMP ping 代码保留为显式诊断模块，但不再是主监控数据源，也不会默认使用 TEST-NET
占位地址。需要做网络层诊断时，应显式提供目标，并把结果当作辅助信息，而不是 Apex 服务健康状态。

Linux 上运行 ICMP 诊断可能需要允许非特权 ping：

```bash
sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
```

## 当前限制和 TODO

- Apex Legends Status API 的字段可能变化；当前 normalizer 对常见字段和核心服务别名做了兼容处理。
- 没有 API key 时只能看到 demo 数据。
- 最近用户报告只有在 API payload 提供相关字段时才会展示。
- ping 诊断还没有作为独立 TUI 面板暴露。
- 后续可以增加配置文件、更多平台服务卡片、状态变化历史和更细粒度的桌面通知。

## English

`neurolink` is a Go Bubble Tea TUI for checking Apex Legends service health using the Apex Legends Status API ecosystem. It reads API keys only from `--api-key` or `NEUROLINK_APEX_API_KEY`. Without a key, or with `--demo`, it shows deterministic demo data clearly labeled as demo data.

Run:

```bash
go run . --api-key "$YOUR_APEX_STATUS_API_KEY"
```

Build and test:

```bash
go build ./...
go test ./...
```
