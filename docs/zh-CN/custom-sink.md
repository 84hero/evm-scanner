# 自定义输出组件 (Custom Sink)

EVM Scanner 不仅可以作为 CLI 工具运行，还可以作为 Go 项目的库引入，并扩展自定义的输出逻辑（Sink）。

## 接口定义

要实现一个自定义 Sink，您需要满足 `sink.Output` 接口：

```go
type Output interface {
    // Name 返回 Sink 的唯一标识
    Name() string
    
    // Send 处理扫描到的日志数据
    Send(ctx context.Context, logs []DecodedLog) error
    
    // Close 释放资源（如关闭数据库连接、网络客户端等）
    Close() error
}
```

## 实现示例：Slack 通知

以下代码展示了如何编写一个简单的 Slack 通知 Sink，并将其集成到扫描流程中。

```go
package main

import (
    "context"
    "fmt"
    "github.com/84hero/evm-scanner/pkg/sink"
)

type SlackSink struct {
    WebhookURL string
}

func (s *SlackSink) Name() string { return "slack" }

func (s *SlackSink) Send(ctx context.Context, logs []sink.DecodedLog) error {
    for _, l := range logs {
        // 实现您的逻辑，例如发送 HTTP 请求到 Slack
        fmt.Printf("[Slack] 发现新交易: %s\n", l.Log.TxHash.Hex())
    }
    return nil
}

func (s *SlackSink) Close() error {
    return nil
}
```

## 集成到扫描器

在您的 `main.go` 中，手动初始化扫描器并调用自定义 Sink：

```go
func main() {
    ctx := context.Background()
    
    // 初始化 RPC 和 存储
    client, _ := rpc.NewClient(ctx, nodes, 5)
    store := storage.NewMemoryStore("prefix_")
    
    // 创建自定义 Sink
    mySink := &SlackSink{WebhookURL: "https://..."}
    
    // 初始化扫描器
    s := scanner.New(client, store, cfg, filter)
    
    // 设置处理函数，调用自定义 Sink
    s.SetHandler(func(ctx context.Context, logs []types.Log) error {
        // 转换/包装日志
        decoded := wrapLogs(logs)
        
        // 发送到自定义 Sink
        return mySink.Send(ctx, decoded)
    })
    
    s.Start(ctx)
}
```

## 为什么使用自定义 Sink？

1. **集成现有系统**：直接调用公司内部的微服务或权限系统。
2. **特定的格式化**：将复杂的区块链数据转换为业务特定的通知模板。
3. **复合逻辑**：根据事件内容执行条件路由（例如：大额转账发钉钉，小额转账入库）。
4. **性能监控**：在 Sink 层添加特定的 Prometheus 指标。

## 相关参考

- 完整的代码示例可以参考 [examples/custom-sink](../../examples/custom-sink/main.go)。
- 默认实现的 Sinks 可以在 [pkg/sink](../../pkg/sink/) 目录下找到。
