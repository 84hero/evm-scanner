# Scanner CLI

The standard command-line interface for the EVM Scanner. It allows you to run a full-featured scanner service driven by YAML configurations.

## Features

- **Configuration Driven**: All filters and outputs are defined in `app.yaml`.
- **High Performance**: Built-in support for all Sinks (Postgres, Redis, Kafka, Webhook, etc.).
- **Environment Support**: Easily switch configurations via environment variables.

## Usage

### 1. Build
```bash
make build
# Binary will be at bin/scanner-cli
```

### 2. Configure
Copy the example files and modify them:
- `config.yaml`: Infrastructure (RPC, Storage).
- `app.yaml`: Business logic (Filters, Sinks).

### 3. Run
```bash
./bin/scanner-cli
```

## Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `CONFIG_FILE` | Path to core configuration | `config.yaml` |
| `APP_CONFIG_FILE` | Path to application filters/sinks | `app.yaml` |
| `PG_URL` | PostgreSQL connection string (Overrides storage) | - |
| `REDIS_ADDR` | Redis address (Overrides storage) | - |

## Example Deployment (Docker)

Refer to the root [docker-compose.yml](../../docker-compose.yml) for a production-ready container setup.
