# EVM Scanner 🚀

[![Go Report Card](https://goreportcard.com/badge/github.com/84hero/evm-scanner)](https://goreportcard.com/report/github.com/84hero/evm-scanner)
[![Build Status](https://github.com/84hero/evm-scanner/workflows/Test%20and%20Lint/badge.svg)](https://github.com/84hero/evm-scanner/actions)
[![GoDoc](https://godoc.org/github.com/84hero/evm-scanner?status.svg)](https://godoc.org/github.com/84hero/evm-scanner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**[English](README.md)** | **[简体中文](README_CN.md)**

一个高性能、工业级的 EVM 事件扫描和索引框架。为需要可靠、实时访问区块链数据的开发者而构建，无需复杂索引解决方案的开销。

[特性](#-特性) • [安装](#-安装) • [快速开始](#-快速开始) • [文档](#-使用示例) • [贡献](#-贡献)

---

## 🌟 特性

-   **⛓️ 原生多链支持**: 针对 Ethereum、BSC、Polygon、Arbitrum 以及任何 EVM 兼容网络进行优化。
-   **💾 可插拔存储**: 选择您的持久化层—— **Memory**（开发）、**Redis**（性能）或 **PostgreSQL**（持久性）。
-   **🚀 高性能**: 
    -   **批量处理**: 高效的 RPC 调用批处理，最小化延迟和成本。
    -   **布隆过滤器支持**: 利用节点级过滤实现大幅速度提升。
    -   **工作池**: 并行输出处理（sinks）适用于高吞吐量环境。
-   **🔌 丰富的生态系统（Sinks）**: 直接将数据流式传输到 **Webhooks**、**Kafka**、**RabbitMQ**、**Redis**、**PostgreSQL** 或平面文件。
-   **🛡️ 生产就绪**: 自动处理链重组，具有可配置的安全窗口和游标管理。
-   **💎 人类可读**: 内置 ABI 解码，自动将原始十六进制日志转换为结构化 JSON 数据。

## 📦 安装

### 二进制文件（推荐）
从 [Releases](https://github.com/84hero/evm-scanner/releases) 页面下载适合您架构的预编译二进制文件。

### 使用 Go 安装
```bash
go install github.com/84hero/evm-scanner/cmd/scanner-cli@latest
```

### 从源码构建
```bash
git clone https://github.com/84hero/evm-scanner.git
cd evm-scanner
make build
```

## 🚀 快速开始

### 1. 初始化配置
```bash
cp config.yaml.example config.yaml
cp app.yaml.example app.yaml
```

### 2. 运行 CLI
```bash
# 基于 app.yaml 过滤器开始扫描
./bin/scanner-cli
```

### 3. Docker（一键启动）
```bash
docker-compose up -d
```

## 🛠 使用示例

### CLI 模式（独立运行）
在 `app.yaml` 中定义您的过滤器：
```yaml
filters:
  - description: "USDT 转账追踪器"
    contracts: ["0xdAC17F958D2ee523a2206206994597C13D831ec7"]
    topics: ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    abi: '[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},...],{"name":"Transfer","type":"event"}]'
```

### SDK 模式（作为库使用）
探索我们精心策划的示例，了解如何将 `evm-scanner` 集成到您的技术栈中：

| 示例 | 描述 |
| :--- | :--- |
| [**基础 SDK**](./examples/basic) | 从 Go 应用程序开始扫描的最小设置。 |
| [**自定义解码器**](./examples/custom-decoder) | 如何使用 ABI 将原始日志解码为人类可读数据。 |
| [**PostgreSQL 集成**](./examples/postgres-integration) | 使用 Postgres 进行进度跟踪和数据存储的生产就绪设置。 |
| [**企业消息队列**](./examples/enterprise-mq) | 将事件数据流式传输到 **Kafka** 用于高吞吐量微服务。 |
| [**多 Sink 管道**](./examples/multi-sink) | 同时将事件分发到控制台和文件。 |
| [**自定义链预设**](./examples/custom-chain) | 为新的 L2 或 AppChain 配置参数（BlockTime、ReorgSafe）。 |
| [**自定义 Sink**](./examples/custom-sink) | 通过实现自己的输出目标（例如 Slack）来扩展框架。 |
| [**Webhook 接收器**](./examples/webhook-receiver) | 一个简单的服务器，用于通过 Webhook 接收和处理事件。 |

```go
import (
    "github.com/84hero/evm-scanner/pkg/scanner"
    "github.com/84hero/evm-scanner/pkg/rpc"
)

func main() {
    client, _ := rpc.NewClient(ctx, rpcCfg, 10)
    s := scanner.New(client, storage, scanCfg, filter)
    
    s.SetHandler(func(ctx context.Context, logs []types.Log) error {
        // 在这里编写您的自定义业务逻辑
        return nil
    })
    
    s.Start(ctx)
}
```

## ⚙️ 配置

项目使用两个主要配置文件：
| 文件 | 用途 | 关键设置 |
| :--- | :--- | :--- |
| `config.yaml` | 基础设施 | RPC 节点、数据库/Redis 连接、扫描速度 |
| `app.yaml` | 业务逻辑 | 合约、主题、ABI、输出目标 |

## 🏗 支持的 Sinks（输出）

| Sink | 状态 | 使用场景 |
| :--- | :--- | :--- |
| **Webhook** | ✅ | 实时 API 集成 |
| **PostgreSQL** | ✅ | 永久事件存储和查询 |
| **Redis** | ✅ | 快速消息传递（List/PubSub） |
| **Kafka** | ✅ | 大数据管道和流处理 |
| **RabbitMQ** | ✅ | 企业消息队列 |
| **Console/File** | ✅ | 调试和日志记录 |

## 🛠 开发

我们使用 `Makefile` 来执行常见任务：
- `make test`: 运行测试套件。
- `make lint`: 运行代码质量检查。
- `make snapshot`: 使用 GoReleaser 进行本地构建验证。

## 🤝 贡献

贡献使开源社区成为一个学习、启发和创造的绝佳场所。我们**非常感谢**您做出的任何贡献。

1. Fork 本项目
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

## 📧 联系方式

- **Telegram**: [@xNew4](https://t.me/xNew4)
- **X (Twitter)**: [@0xNew4](https://x.com/0xNew4)

## 📄 许可证

根据 MIT 许可证分发。有关更多信息，请参阅 `LICENSE`。

---
用 ❤️ 为 Web3 社区构建。
