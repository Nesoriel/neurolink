# neurolink

[English](README.md) | 简体中文

`neurolink` 是一个 Go 终端 UI 工具，用来在开玩或排查问题前观察 Apex Legends 相关服务是否健康。界面主题是 Crypto 风格的“监控无人机”，重点展示服务可用性，而不是把占位 ICMP ping 伪装成真实 Apex 服务器监控。

它关注这些服务和信号：

- Crossplay Auth
- Lobby / Matchmaking
- PC / Desktop Logins
- Player Accounts
- Apex Legends Status API 健康度
- 如果状态 payload 提供，则展示最近用户反馈

`neurolink` 不查询玩家资料、战绩、UID、比赛历史或玩家在线状态。

## 当前数据来源

主数据源是 Apex Legends Status 生态的公开站点和 API：

- 网站：`https://apexlegendsstatus.com/`
- API 文档与入口：`https://apexlegendsapi.com/`
- API 基础地址：`https://api.apexlegendsstatus.com/`

当前实现会轮询：

```text
GET https://api.apexlegendsstatus.com/servers
```

注意：`/servers` 端点可能返回 `Content-Type: text/plain;charset=UTF-8`，但 body 内容是 JSON。因此客户端使用 `Accept: */*`，然后按 JSON 解码 body，避免因为严格要求 `application/json` 而被上游拒绝。

界面会把响应归一化为核心服务卡片。顶部 Overall 只代表“可玩核心服务”的汇总状态；`Apex Legends Status API` 是第三方状态站/API 自身健康度，会单独显示，不会把状态站问题直接当成 Apex 核心服务宕机。

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

## 语言配置

TUI 支持英文和简体中文：

```bash
go run . --lang en
go run . --lang zh-Hans
```

也可以用环境变量：

```bash
NEUROLINK_LANG=zh-Hans go run .
```

支持值只有 `en` 和 `zh-Hans`。默认是 `en`。

## 无 API Key 时的行为

如果没有 `--api-key`，并且没有设置 `NEUROLINK_APEX_API_KEY`，程序会进入 demo 模式。

Demo 模式使用确定性的示例数据，只用于展示界面和本地开发。界面会显示 `DEMO` 来源和演示提示，不会把示例数据伪装成实时 Apex 服务状态。

## 运行、构建、测试

运行 demo：

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
go test -race ./...
```

格式化：

```bash
gofmt -w .
```

## 用户反馈面板

右上角面板在英文界面中显示为 `Community Pulse`，在中文界面中显示为 `用户反馈`。如果 `/servers` payload 包含最近用户反馈，界面会展示简短摘要；如果没有相关字段，界面会说明当前 `/servers` 响应不一定包含 report feed，并保留 Apex Legends Status 的数据来源说明。

这个面板只提供服务健康参考，不做玩家资料或玩家状态查询。

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
- ping 诊断还没有作为独立 TUI 面板暴露。
- 后续可以增加配置文件、更多服务卡片、状态变化历史和更细粒度的桌面通知。
