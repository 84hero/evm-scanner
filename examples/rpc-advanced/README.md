# Advanced RPC Features Example

This example demonstrates the advanced RPC features introduced in the latest version:

## Features Demonstrated

### 1. Per-Node Rate Limiting
Each RPC node has its own QPS (queries per second) limit to prevent hitting provider rate limits.

```go
{
    URL:           "https://eth.llamarpc.com",
    Priority:      10,
    RateLimit:     25,  // 25 requests per second
    MaxConcurrent: 10,
}
```

### 2. Per-Node Concurrency Control
Limit the number of concurrent requests to each node to prevent overload.

```go
{
    URL:           "https://rpc.ankr.com/eth",
    Priority:      8,
    RateLimit:     10,
    MaxConcurrent: 5,   // Max 5 concurrent requests
}
```

### 3. Circuit Breaker
Automatically disables nodes that experience consecutive failures:
- Trips after 5 consecutive errors
- Auto-recovers after 30 seconds
- Prevents cascading failures

### 4. Automatic Node Switching
When a node is busy or rate-limited, the client automatically switches to another available node based on priority and health.

### 5. Dynamic Node Scoring
Nodes are scored based on:
- Priority level
- Response latency
- Error rate
- Block height lag

## How to Run

```bash
cd examples/rpc-advanced
go run main.go
```

## Expected Output

```
🚀 Advanced RPC Client Demo
==================================================

📊 Demo 1: Basic RPC Call
✅ Current block: 21047197

🔄 Demo 2: Concurrent Requests (20 requests)
✅ Success: 20/20 requests
⏱️  Duration: 2.5s
📈 Throughput: 8.00 req/s

⚡ Demo 3: Rate Limiting Test
Sending 30 rapid requests...
✅ Success: 30/30 requests
⏱️  Duration: 4.2s
📊 Average QPS: 7.14

🏥 Demo 4: Node Health Status
✅ All nodes operational
   - Circuit breakers: Normal
   - Rate limits: Active
   - Concurrency: Controlled

✨ Demo completed successfully!

Key Features Demonstrated:
  ✅ Per-node rate limiting
  ✅ Per-node concurrency control
  ✅ Automatic node switching
  ✅ High availability with multiple nodes
```

## Configuration Best Practices

### For Paid RPC Providers (Alchemy, Infura)
```go
{
    URL:           "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
    Priority:      10,
    RateLimit:     25-50,  // Based on your plan
    MaxConcurrent: 10-20,  // 30-50% of rate_limit
}
```

### For Free Public Nodes
```go
{
    URL:           "https://rpc.ankr.com/eth",
    Priority:      5,
    RateLimit:     5-10,   // Conservative limits
    MaxConcurrent: 3-5,    // Lower concurrency
}
```

## Key Concepts

### Rate Limit vs Max Concurrent

- **Rate Limit (QPS)**: Total requests per second
  - Example: 25 QPS = max 25 requests in any 1-second window
  - Prevents hitting provider's rate limits

- **Max Concurrent**: Simultaneous in-flight requests
  - Example: 10 concurrent = max 10 requests running at the same time
  - Prevents node overload and connection exhaustion

### Why Both?

```
Scenario: rate_limit=25, max_concurrent=10

✅ Good: 10 concurrent requests, each takes 100ms
   - All complete within 1 second
   - Total: 10 requests/second (within limit)

❌ Bad: 25 concurrent requests, each takes 1s
   - Would exceed max_concurrent (only 10 allowed)
   - Automatically switches to another node
```

## Related Documentation

- [Configuration Guide](../../docs/en/configuration.md)
- [RPC Client API](../../docs/en/api-reference.md)
- [Architecture](../../docs/en/architecture.md)

## Troubleshooting

### "Rate limit exceeded" errors
- Reduce `rate_limit` value
- Add more RPC nodes
- Increase `interval` in scanner config

### "Node busy" errors
- Increase `max_concurrent` value
- Add more RPC nodes
- Reduce concurrent scanner operations

### Circuit breaker keeps tripping
- Check RPC node health
- Verify API keys are valid
- Consider using paid RPC tier
