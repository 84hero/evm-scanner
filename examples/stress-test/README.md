# RPC Stress Test

This example provides comprehensive stress testing for the RPC client's concurrency control and rate limiting features.

## Test Scenarios

### 1. Light Load Test
- **Workers**: 10
- **Duration**: 10 seconds
- **Request Delay**: 100ms
- **Purpose**: Verify basic functionality under normal load

### 2. Medium Load Test
- **Workers**: 50
- **Duration**: 10 seconds
- **Request Delay**: 50ms
- **Purpose**: Test moderate concurrent load

### 3. Heavy Load Test
- **Workers**: 100
- **Duration**: 10 seconds
- **Request Delay**: 10ms
- **Purpose**: Test system under heavy concurrent load

### 4. Burst Test
- **Concurrent Requests**: 200
- **Purpose**: Test handling of sudden traffic spikes

### 5. Sustained Load Test
- **Workers**: 20
- **Duration**: 30 seconds
- **Request Delay**: 50ms
- **Purpose**: Verify stability under sustained load

## How to Run

```bash
cd examples/stress-test
go run main.go
```

## Expected Output

```
🔥 RPC Stress Test Suite
============================================================

📊 Test 1: Light Load (10 workers, 10s)
  [1s] Requests: 95 | Success: 95 | Errors: 0 | QPS: 95.00
  [2s] Requests: 189 | Success: 189 | Errors: 0 | QPS: 94.50
  ...

Results:
  Total Requests:  950
  ✅ Success:      950 (100.00%)
  ❌ Errors:       0 (0.00%)
  ⏱️  Duration:     10.05s
  📈 Throughput:   94.53 req/s
  ⚡ Avg Latency:  10.58 ms

📊 Test 2: Medium Load (50 workers, 10s)
  [1s] Requests: 245 | Success: 240 | Errors: 5 | QPS: 240.00
  ...

Results:
  Total Requests:  2450
  ✅ Success:      2380 (97.14%)
  ❌ Errors:       70 (2.86%)
  ⏱️  Duration:     10.12s
  📈 Throughput:   235.18 req/s
  ⚡ Avg Latency:  4.13 ms

📊 Test 3: Heavy Load (100 workers, 10s)
...

📊 Test 4: Burst Test (200 concurrent)
Results:
  Total Requests:  200
  ✅ Success:      195 (97.50%)
  ❌ Errors:       5 (2.50%)
  ⏱️  Duration:     1.85s
  📈 Throughput:   105.41 req/s
  ⚡ Avg Latency:  9.25 ms

📊 Test 5: Sustained Load (20 workers, 30s)
...

✨ All stress tests completed!
```

## What This Tests

### Rate Limiting
- Verifies per-node QPS limits are enforced
- Tests automatic node switching when limits are hit
- Measures actual throughput vs configured limits

### Concurrency Control
- Tests max concurrent request limits
- Verifies proper resource acquisition and release
- Checks for race conditions

### Circuit Breaker
- Tests behavior under node failures
- Verifies automatic recovery
- Checks failover to backup nodes

### Load Balancing
- Tests distribution across multiple nodes
- Verifies priority-based selection
- Checks automatic switching on busy nodes

## Interpreting Results

### Success Rate
- **>95%**: Excellent - system handling load well
- **90-95%**: Good - some rate limiting occurring
- **<90%**: Review configuration or add more nodes

### Throughput (QPS)
- Should be close to sum of all node rate limits
- Lower than expected: check node health
- Higher than expected: check if limits are configured

### Error Rate
- Errors are normal under stress (rate limiting)
- High error rate (>10%): may need more capacity
- All errors: check RPC node connectivity

## Configuration Tips

### For Higher Throughput
```go
nodes := []rpc.NodeConfig{
    {
        URL:           "...",
        Priority:      10,
        RateLimit:     50,  // Increase
        MaxConcurrent: 20,  // Increase
    },
}
```

### For More Stability
```go
nodes := []rpc.NodeConfig{
    {
        URL:           "...",
        Priority:      10,
        RateLimit:     10,  // Conservative
        MaxConcurrent: 5,   // Conservative
    },
}
```

### For High Availability
```go
// Add more nodes
nodes := []rpc.NodeConfig{
    {URL: "node1", Priority: 10, RateLimit: 25, MaxConcurrent: 10},
    {URL: "node2", Priority: 9,  RateLimit: 25, MaxConcurrent: 10},
    {URL: "node3", Priority: 8,  RateLimit: 25, MaxConcurrent: 10},
    {URL: "node4", Priority: 7,  RateLimit: 10, MaxConcurrent: 5},
}
```

## Troubleshooting

### High Error Rate
1. Check RPC node health
2. Verify API keys are valid
3. Increase rate limits if using paid tier
4. Add more backup nodes

### Low Throughput
1. Increase `rate_limit` values
2. Increase `max_concurrent` values
3. Add more RPC nodes
4. Check network latency

### Timeouts
1. Increase context timeout
2. Check RPC node response times
3. Reduce concurrent load

## Running Unit Tests

```bash
# Run all tests including stress tests
go test ./pkg/rpc/... -v

# Run only stress tests
go test ./pkg/rpc/... -v -run Stress

# Skip stress tests (for quick testing)
go test ./pkg/rpc/... -v -short

# Run benchmarks
go test ./pkg/rpc/... -bench=. -benchmem
```

## Benchmark Results

Expected benchmark results on modern hardware:

```
BenchmarkNode_TryAcquire-8         500000    2500 ns/op    128 B/op    2 allocs/op
BenchmarkMultiClient_BlockNumber-8 100000   15000 ns/op    512 B/op    8 allocs/op
```

## Related Documentation

- [Configuration Guide](../../docs/en/configuration.md)
- [Advanced RPC Example](../rpc-advanced/README.md)
- [Architecture](../../docs/en/architecture.md)
