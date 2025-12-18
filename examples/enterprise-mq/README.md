# Enterprise Message Queue (Kafka) Example

In large-scale architectures, event data is often streamed into message brokers like **Kafka** or **RabbitMQ** for downstream consumption by microservices, analytics engines, or data warehouses.

## Features Shown

- **Kafka Integration**: Using the `sink.KafkaOutput` to produce messages.
- **Failover Logic**: Demonstration of how to handle MQ connectivity issues in the application layer.
- **Real-world Filter**: Monitoring Uniswap V2 Pair Creation events.

## Prerequisites

- A running Kafka cluster.
- (Optional) Set `KAFKA_BROKER` environment variable.

## How to Run

1.  **Start Kafka** (using Docker):
    ```bash
    # Simple single-node Kafka setup
    docker run -p 9092:9092 apache/kafka:latest
    ```

2.  **Run the example**:
    ```bash
    go run main.go
    ```

## Message Format

Events are sent as JSON-encoded messages to the specified Kafka topic. Each message includes the raw log data and any available metadata, keyed by the transaction hash for partition stability.
