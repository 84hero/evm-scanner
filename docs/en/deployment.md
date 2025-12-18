# Deployment Guide

Best practices for deploying and operating EVM Scanner in production.

## Deployment Options

### 1. Docker Compose (Recommended)

The easiest way to get started with a persistent database.

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

### 2. Systemd (Direct Binary)

For traditional VM deployments. Create `/etc/systemd/system/evm-scanner.service`:

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

[Install]
WantedBy=multi-user.target
```

## Operations

### 1. Monitoring

Key metrics to watch:
- **Scan Lag**: Difference between chain tip and current scanned height.
- **RPC Error Rate**: Frequency of failed node requests.
- **Sink Failures**: Webhook timeouts or DB write errors.
- **Resource Usage**: CPU and Memory consumption.

### 2. Logging

Use `json` format in production for easier log aggregation:
```yaml
log:
  format: "json"
  level: "info"
```

### 3. High Availability RPC

Always configure at least 2 nodes from different providers:
- **Primary**: Paid node (Alchemy, Infura) for performance.
- **Backup**: Public node or secondary provider for failover.

## Sizing Recommendations

| Scale | Network | Recommendation |
| :--- | :--- | :--- |
| Small | Ethereum | 1 CPU / 2G RAM |
| Medium | BSC/Polygon | 2 CPU / 4G RAM |
| Large | Full Indexing | 4 CPU / 8G RAM+ |
