# PostgreSQL Integration Example

This example demonstrates a production-like setup where PostgreSQL is used for both:
1.  **Persistence**: Saving the scanner's current block height (cursor).
2.  **Sink**: Storing the actual event data into a relational table.

## Prerequisites

- A running PostgreSQL instance.
- Set the `PG_URL` environment variable.

## How to Run

1.  **Start Postgres** (using Docker):
    ```bash
    docker run --name some-postgres -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres
    ```

2.  **Run the example**:
    ```bash
    export PG_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
    go run main.go
    ```

## What happens?

- The scanner will automatically create two tables:
    - `demo_cursors`: To track the last scanned block.
    - `contract_events`: To store decoded (or raw) logs with a unique constraint on `(tx_hash, log_index)` to prevent duplicates.
- You can query the data using:
    ```sql
    SELECT * FROM contract_events ORDER BY block_number DESC;
    ```
