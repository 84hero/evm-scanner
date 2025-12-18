# 快速开始

本指南将帮助您在 5 分钟内启动并运行 EVM Scanner。

## 前置要求

- Go 1.21 或更高版本（如果从源码构建）
- 一个 EVM 兼容链的 RPC 节点访问权限

## 安装

### 方式 1: 下载预编译二进制文件（推荐）

访问 [Releases](https://github.com/84hero/evm-scanner/releases) 页面，下载适合您系统的版本：

```bash
# macOS (Apple Silicon)
wget https://github.com/84hero/evm-scanner/releases/download/v0.1.0/evm-scanner_Darwin_arm64.tar.gz
tar -xzf evm-scanner_Darwin_arm64.tar.gz

# macOS (Intel)
wget https://github.com/84hero/evm-scanner/releases/download/v0.1.0/evm-scanner_Darwin_x86_64.tar.gz
tar -xzf evm-scanner_Darwin_x86_64.tar.gz

# Linux
wget https://github.com/84hero/evm-scanner/releases/download/v0.1.0/evm-scanner_Linux_x86_64.tar.gz
tar -xzf evm-scanner_Linux_x86_64.tar.gz

# 移动到系统路径
sudo mv evm-scanner /usr/local/bin/
```

### 方式 2: 使用 Go 安装

```bash
go install github.com/84hero/evm-scanner/cmd/scanner-cli@latest
```

### 方式 3: 从源码构建

```bash
git clone https://github.com/84hero/evm-scanner.git
cd evm-scanner
make build

# 二进制文件位于 ./bin/scanner-cli
```

## 配置

### 1. 创建配置文件

```bash
# 使用中文配置模板（推荐中文用户）
cp config.yaml.example.cn config.yaml
cp app.yaml.example.cn app.yaml

# 或使用英文配置模板
cp config.yaml.example config.yaml
cp app.yaml.example app.yaml
```

### 2. 配置 RPC 节点

编辑 `config.yaml`，设置您的 RPC 节点：

```yaml
rpc_nodes:
  - url: "https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY"
    priority: 10
```

**免费 RPC 节点获取：**
- [Alchemy](https://www.alchemy.com/) - 推荐，每天 300M 免费额度
- [Infura](https://infura.io/) - 每天 100k 请求
- [QuickNode](https://www.quicknode.com/) - 免费试用

### 3. 配置过滤器

编辑 `app.yaml`，定义您要监听的事件：

**示例 1: 监听 USDT 转账**
```yaml
filters:
  - description: "USDT 转账"
    contracts:
      - "0xdAC17F958D2ee523a2206206994597C13D831ec7"
    topics:
      - ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
```

**示例 2: 监听多个稳定币**
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

### 4. 配置输出

选择您希望如何接收事件数据：

**控制台输出（开发/调试）：**
```yaml
outputs:
  console:
    enabled: true
```

**Webhook（生产推荐）：**
```yaml
outputs:
  webhook:
    enabled: true
    url: "https://your-api.com/webhook"
    async: true
    workers: 5
```

## 运行

### 基础运行

```bash
./evm-scanner
# 或如果使用 go install
scanner-cli
```

### 使用环境变量

```bash
# 指定配置文件路径
CONFIG_FILE=./my-config.yaml APP_CONFIG_FILE=./my-app.yaml ./evm-scanner
```

### Docker 运行

```bash
# 构建镜像
docker build -t evm-scanner .

# 运行容器
docker run -v $(pwd)/config.yaml:/app/config.yaml \
           -v $(pwd)/app.yaml:/app/app.yaml \
           evm-scanner
```

### Docker Compose

```bash
docker-compose up -d
```

## 验证运行

成功运行后，您应该看到类似的输出：

```
INFO [12-19|05:00:00] Starting EVM Scanner                     chain=ethereum
INFO [12-19|05:00:00] Connected to RPC                         url=https://eth-mainnet.g.alchemy.com/v2/***
INFO [12-19|05:00:01] Scanner started                          from_block=18900000 current_block=18950000
INFO [12-19|05:00:02] Processing logs                          block=18900050 logs=5
```

## 常见场景

### 场景 1: 监听 NFT 铸造

```yaml
filters:
  - description: "Bored Ape NFT 铸造"
    contracts:
      - "0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"
    topics:
      - ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    abi: '[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":true,"name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"}]'
```

### 场景 2: 监听 Uniswap 交易

```yaml
filters:
  - description: "Uniswap V3 Swap"
    contracts:
      - "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640"  # USDC/ETH Pool
    topics:
      - ["0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67"]
```

### 场景 3: 发送到 Webhook

```yaml
outputs:
  webhook:
    enabled: true
    url: "https://your-api.com/events"
    secret: "your-secret-key"
    retry:
      max_attempts: 3
      initial_backoff: "1s"
      max_backoff: "10s"
    async: true
    buffer_size: 2000
    workers: 5
```

### 场景 4: 存储到 PostgreSQL

```yaml
outputs:
  postgres:
    enabled: true
    url: "postgres://user:password@localhost:5432/events?sslmode=disable"
    table: "blockchain_events"
```

## 进阶用法

### 从特定区块开始

```yaml
scanner:
  force_start: true
  start_block: 18900000  # 从这个区块开始扫描
```

### 回填历史数据

```bash
# 1. 设置起始区块
# config.yaml
scanner:
  force_start: true
  start_block: 18000000
  batch_size: 100  # 增加批量大小加速回填

# 2. 运行扫描器
./evm-scanner

# 3. 完成后，恢复正常配置
scanner:
  force_start: false
  batch_size: 50
```

### 多链部署

为不同的链创建不同的配置文件：

```bash
# Ethereum
CONFIG_FILE=config-eth.yaml APP_CONFIG_FILE=app-eth.yaml ./evm-scanner &

# BSC
CONFIG_FILE=config-bsc.yaml APP_CONFIG_FILE=app-bsc.yaml ./evm-scanner &

# Polygon
CONFIG_FILE=config-polygon.yaml APP_CONFIG_FILE=app-polygon.yaml ./evm-scanner &
```

## 故障排查

### 问题 1: RPC 连接失败

```
ERROR Failed to connect to RPC url=https://...
```

**解决方案：**
- 检查 RPC URL 是否正确
- 验证 API Key 是否有效
- 尝试使用其他 RPC 节点
- 检查网络连接

### 问题 2: 没有扫描到事件

**检查清单：**
- ✅ 合约地址是否正确
- ✅ Topic 是否正确
- ✅ 起始区块是否在事件发生之前
- ✅ 过滤器配置是否正确

### 问题 3: 内存占用过高

**优化配置：**
```yaml
scanner:
  batch_size: 20  # 减小批量大小

outputs:
  webhook:
    buffer_size: 500  # 减小缓冲区
    workers: 2        # 减少工作线程
```

### 问题 4: 扫描速度慢

**优化配置：**
```yaml
scanner:
  batch_size: 100      # 增加批量大小
  use_bloom: true      # 启用布隆过滤器
  confirmations: 3     # 减少确认数（谨慎）
```

## 下一步

- 📖 阅读[配置指南](./configuration.md)了解所有配置选项
- 🏗️ 查看[架构设计](./architecture.md)了解工作原理
- 💻 探索[示例代码](../../examples/)学习 SDK 用法
- 🔌 学习如何[自定义 Sink](./custom-sink.md)

## 获取帮助

- 💬 [GitHub Discussions](https://github.com/84hero/evm-scanner/discussions)
- 🐛 [提交 Issue](https://github.com/84hero/evm-scanner/issues)
- 📧 联系作者: [@xNew4](https://t.me/xNew4)
