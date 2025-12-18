# 部署指南

本文档介绍如何在生产环境中部署和运维 EVM Scanner。

## 部署方式

### 1. Docker Compose (推荐)

对于大多数中小型项目，使用 Docker Compose 是最简单且高效的方式。

```yaml
version: '3.8'

services:
  evm-scanner:
    image: 84hero/evm-scanner:latest
    restart: always
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./app.yaml:/app/app.yaml
    environment:
      - CONFIG_FILE=/app/config.yaml
      - APP_CONFIG_FILE=/app/app.yaml
      - PG_URL=postgres://user:pass@postgres:5432/scanner?sslmode=disable
    depends_on:
      - postgres

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: scanner
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

### 2. Systemd (传统 Linux 部署)

如果您直接在虚拟机上运行二进制文件，建议使用 Systemd 进行管理。

创建 `/etc/systemd/system/evm-scanner.service`:

```ini
[Unit]
Description=EVM Scanner Service
After=network.target

[Service]
Type=simple
User=scanner
WorkingDirectory=/home/scanner
Environment=CONFIG_FILE=/home/scanner/config.yaml
Environment=APP_CONFIG_FILE=/home/scanner/app.yaml
ExecStart=/usr/local/bin/scanner-cli
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
systemctl daemon-reload
systemctl enable evm-scanner
systemctl start evm-scanner
```

## 运维建议

### 1. 监控与告警

建议监控以下指标：
- **扫块延迟**：当前链最高高度与已扫描高度的差值。
- **RPC 错误率**：节点返回非法响应的情况。
- **输出失败数**：Webhook 或 数据库写入失败的频率。
- **进程资源**：CPU 和 内存使用情况，避免泄露。

### 2. 日志收集

在生产环境中，建议将日志格式设置为 `json`：
```yaml
log:
  format: "json"
  level: "info"
```
结合 ELK (Elasticsearch, Logstash, Kibana) 或 Grafana Loki 进行集中存储和查询。

### 3. 高可用 RPC

务必配置至少 2 个不同供应商的 RPC 节点：
- **主节点**：付费节点（如 Alchemy, Infura），保证速度和稳定性。
- **备用节点**：公共节点或第二个付费供应商，防止单点故障。

### 4. 数据库索引优化

当 `contract_events` 表数据量达到百万级以上时，建议根据业务需求增加索引：
- 如果经常查询某个合约：`CREATE INDEX idx_address ON contract_events(address);`
- 如果经常查询某个事件：`CREATE INDEX idx_topic0 ON contract_events((topics->>0));`

## 规格建议

| 规模 | 区块链 | 建议规格 | 备注 |
| :--- | :--- | :--- | :--- |
| 小规模 | Ethereum | 1 CPU / 2G RAM | 少量合约监听 |
| 中规模 | BSC/Polygon | 2 CPU / 4G RAM | 处理由于出块快带来的高频日志 |
| 大规模 | 多链全量监听 | 4 CPU / 8G RAM+ | 建议按链拆分部署 |
