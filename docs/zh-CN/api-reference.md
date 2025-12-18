# API 参考

本文档提供了 EVM Scanner 的接口定义、数据格式及命令行参考。

## 命令行选项 (CLI)

`scanner-cli` 是核心运行程序，支持以下参数：

### 环境控制
- `CONFIG_FILE`: 指定 `config.yaml` 路径（默认: `./config.yaml`）
- `APP_CONFIG_FILE`: 指定 `app.yaml` 路径（默认: `./app.yaml`）
- `PG_URL`: 覆盖 Postgres 存储连接串
- `REDIS_ADDR`: 覆盖 Redis 存储地址

### 运行示例
```bash
# 使用默认配置启动
./scanner-cli

# 指定自定义配置文件
CONFIG_FILE=./prod/config.yaml APP_CONFIG_FILE=./prod/app.yaml ./scanner-cli
```

## Webhook 数据格式

当启用 Webhook 输出时，EVM Scanner 会向指定的 URL 发送 JSON 格式的 `POST` 请求。

### 请求体 (Payload)
系统会批量发送解析后的日志：

```json
[
  {
    "address": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
    "topics": [
      "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
      "0x000000000000000000000000123...",
      "0x000000000000000000000000456..."
    ],
    "data": "0x0000000000000000000000000000000000000000000000000000000005f5e100",
    "blockNumber": 18000000,
    "transactionHash": "0xabc...",
    "transactionIndex": 42,
    "blockHash": "0xdef...",
    "logIndex": 0,
    "removed": false,
    "eventName": "Transfer",
    "decoded": {
      "from": "0x123...",
      "to": "0x456...",
      "value": "100000000"
    }
  }
]
```

### 签名验证
如果配置了 `secret`，请求头中将包含 `X-Scanner-Signature`。其计算方式为：
`HMAC-SHA256(payload_body, secret)`

## 数据库结构 (Postgres)

如果启用 Postgres 输出，系统会自动维护以下表结构：

### `contract_events` 表

| 字段名 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | SERIAL | 主键 |
| `block_number` | BIGINT | 区块高度 |
| `transaction_hash` | VARCHAR(66) | 交易哈希 |
| `log_index` | INTEGER | 日志索引 |
| `address` | VARCHAR(42) | 合约地址 |
| `topics` | JSONB | 事件主题列表 |
| `data` | TEXT | 原始数据 |
| `decoded` | JSONB | 解码后的参数 |
| `created_at` | TIMESTAMP | 写入时间 |

## 消息队列结构

发送到 Kafka, RabbitMQ, Redis 的消息内容与 Webhook 数据格式一致，但通常为单条发送而非数组。

## 错误代码

| 代码 | 说明 | 建议操作 |
| :--- | :--- | :--- |
| `RPC_ERROR` | RPC 节点无法访问 | 检查网络或增加备用节点 |
| `DECODE_ERROR` | ABI 匹配失败或数据格式错误 | 检查 `app.yaml` 中的 ABI 定义 |
| `SINK_ERROR` | 下游输出目标返回错误 | 检查 Webhook 接收端或数据库状态 |
| `REORG_DETECTED` | 检测到链重组 | 系统将自动执行回退逻辑 |
