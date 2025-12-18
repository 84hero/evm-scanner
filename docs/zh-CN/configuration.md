# 配置指南

本文档详细介绍 EVM Scanner 的配置选项。

## 配置文件

EVM Scanner 使用两个主要配置文件：

- **`config.yaml`** - 基础设施配置（RPC、存储、扫描参数）
- **`app.yaml`** - 业务逻辑配置（过滤器、输出目标）

## config.yaml 详解

### 基本配置

```yaml
# 项目唯一标识符
# 用于在数据库或 Redis 中区分不同项目的扫描进度
project: "evm-scanner-service"
```

### 日志配置

```yaml
log:
  # 日志级别: debug, info, warn, error
  level: "info"
  
  # 日志格式:
  # - text: 开发模式，彩色输出，易读
  # - json: 生产模式，结构化输出，便于日志收集
  format: "text"
```

### 扫描器配置

```yaml
scanner:
  # 链标识符
  # 支持: ethereum, polygon, bsc, arbitrum, optimism 等
  chain_id: "ethereum"
  
  # === 启动策略 ===
  
  # 强制启动模式
  # true: 忽略已保存的进度，从 start_block 开始
  # false: 从上次保存的进度继续（推荐）
  force_start: false
  
  # 起始区块号
  # 仅在 force_start=true 或首次运行时生效
  start_block: 0
  
  # 初始回退区块数
  # 首次运行时，从 (当前区块 - start_rewind) 开始扫描
  # 用于避免遗漏最近的事件
  start_rewind: 1000
  
  # 重启回退区块数
  # 重启时，从 (上次保存位置 - cursor_rewind) 开始
  # 用于处理短期链重组
  cursor_rewind: 10
  
  # === 性能参数 ===
  
  # 批量大小
  # 每次 RPC 请求获取的区块数量
  # 建议值: 10-100（根据 RPC 限制调整）
  batch_size: 50
  
  # 轮询间隔
  # 检查新区块的时间间隔
  # 建议: 链的出块时间 / 2
  interval: "2s"
  
  # 确认数
  # 扫描到 (最新区块 - confirmations) 的位置
  # 用于避免链重组导致的数据不一致
  # 以太坊建议: 12-64
  confirmations: 12
  
  # 布隆过滤器
  # 启用节点级过滤，大幅提升性能
  # 需要 RPC 节点支持
  use_bloom: true
  
  # 存储前缀
  # 用于隔离不同项目的数据
  # 会添加到表名或 Redis 键前面
  storage_prefix: "evm_scan_"
```

### RPC 节点配置

```yaml
# RPC 节点池
# 支持多节点高可用，按优先级自动故障转移
rpc_nodes:
  # 主节点（优先级最高）
  - url: "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
    priority: 10
  
  # 备用节点
  - url: "https://rpc.ankr.com/eth"
    priority: 5
```

**优先级说明：**
- 数字越大，优先级越高
- 主节点故障时，自动切换到次优先级节点
- 建议配置 2-3 个节点以确保高可用

## app.yaml 详解

### 过滤器配置

```yaml
filters:
  - description: "USDT 转账事件"
    
    # 合约地址列表
    # 留空则监听所有合约
    contracts:
      - "0xdAC17F958D2ee523a2206206994597C13D831ec7"
    
    # 事件主题（Topic）
    # Topic0 是事件签名的 keccak256 哈希
    # Transfer(address,address,uint256) 的签名
    topics:
      - ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    
    # ABI 定义（可选）
    # 提供后会自动解码日志为人类可读格式
    abi: '[{"anonymous":false,"inputs":[...],"name":"Transfer","type":"event"}]'
```

**多合约示例：**
```yaml
filters:
  - description: "主流稳定币转账"
    contracts:
      - "0xdAC17F958D2ee523a2206206994597C13D831ec7"  # USDT
      - "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"  # USDC
      - "0x6B175474E89094C44Da98b954EedeAC495271d0F"  # DAI
    topics:
      - ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
```

### 输出配置

#### 1. Webhook

```yaml
outputs:
  webhook:
    enabled: true
    url: "https://your-api.com/webhook"
    secret: "your-signing-secret"
    
    # 重试策略
    retry:
      max_attempts: 3          # 最大重试次数
      initial_backoff: "1s"    # 初始退避时间
      max_backoff: "10s"       # 最大退避时间
    
    # 异步模式（推荐）
    async: true
    buffer_size: 2000  # 缓冲区大小
    workers: 5         # 并发工作线程数
```

**Webhook 数据格式：**
```json
{
  "block_number": 12345678,
  "transaction_hash": "0x...",
  "log_index": 0,
  "address": "0x...",
  "topics": ["0x..."],
  "data": "0x...",
  "decoded": {
    "name": "Transfer",
    "params": {
      "from": "0x...",
      "to": "0x...",
      "value": "1000000"
    }
  }
}
```

#### 2. PostgreSQL

```yaml
outputs:
  postgres:
    enabled: true
    url: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
    table: "contract_events"
```

**自动创建的表结构：**
```sql
CREATE TABLE contract_events (
    id SERIAL PRIMARY KEY,
    block_number BIGINT NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    log_index INTEGER NOT NULL,
    address VARCHAR(42) NOT NULL,
    topics JSONB,
    data TEXT,
    decoded JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(transaction_hash, log_index)
);
```

#### 3. Redis

```yaml
outputs:
  redis:
    enabled: true
    addr: "localhost:6379"
    password: ""
    db: 0
    key: "evm_events_queue"
    
    # 模式选择
    # list: 使用 LPUSH，适合队列消费
    # pubsub: 使用 PUBLISH，适合广播
    mode: "list"
```

#### 4. Kafka

```yaml
outputs:
  kafka:
    enabled: true
    brokers: ["localhost:9092"]
    topic: "evm-events"
    
    # SASL 认证（可选）
    user: "kafka-user"
    password: "kafka-password"
```

#### 5. RabbitMQ

```yaml
outputs:
  rabbitmq:
    enabled: true
    url: "amqp://guest:guest@localhost:5672/"
    exchange: "evm_events_ex"
    routing_key: "eth.mainnet"
    queue_name: "evm_events_q"
    durable: true  # 消息持久化
```

#### 6. 文件输出

```yaml
outputs:
  file:
    enabled: true
    path: "./data/events.jsonl"  # JSON Lines 格式
```

#### 7. 控制台输出

```yaml
outputs:
  console:
    enabled: true  # 输出到 stdout
```

## 环境变量

配置文件路径可以通过环境变量指定：

```bash
# 指定配置文件路径
export CONFIG_FILE=/path/to/config.yaml
export APP_CONFIG_FILE=/path/to/app.yaml

# 存储配置
export PG_URL="postgres://user:pass@localhost/db"
export REDIS_ADDR="localhost:6379"
```

## 最佳实践

### 1. 生产环境配置

```yaml
log:
  level: "info"
  format: "json"

scanner:
  batch_size: 50
  interval: "2s"
  confirmations: 64  # 更保守的确认数
  use_bloom: true

# 使用多个 RPC 节点
rpc_nodes:
  - url: "https://primary-rpc.com"
    priority: 10
  - url: "https://backup-rpc.com"
    priority: 5
```

### 2. 开发环境配置

```yaml
log:
  level: "debug"
  format: "text"

scanner:
  batch_size: 10
  interval: "1s"
  confirmations: 3
  use_bloom: false

outputs:
  console:
    enabled: true
```

### 3. 性能优化

- **batch_size**: 根据 RPC 限制调整（10-100）
- **interval**: 设置为出块时间的 1/2 到 1/3
- **confirmations**: 平衡安全性和实时性
- **use_bloom**: 生产环境务必启用
- **async**: Webhook 使用异步模式

### 4. 安全建议

- 使用环境变量存储敏感信息（API Key、密码）
- 生产环境使用 `json` 格式日志
- 启用 Webhook 签名验证
- 使用 SSL/TLS 连接数据库

## 故障排查

### RPC 连接失败
```yaml
# 检查 RPC URL 是否正确
# 尝试降低 batch_size
# 增加多个备用节点
```

### 内存占用过高
```yaml
# 降低 batch_size
# 减少 webhook.buffer_size
# 减少 webhook.workers
```

### 扫描速度慢
```yaml
# 增加 batch_size
# 启用 use_bloom
# 使用更快的 RPC 节点
# 减少 confirmations（谨慎）
```

## 相关文档

- [快速开始](./quick-start.md)
- [架构设计](./architecture.md)
- [API 参考](./api-reference.md)
