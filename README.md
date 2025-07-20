# Kube Event Analyzer

Kube Event Analyzer is a standalone, lightweight tool designed to efficiently collect, store, and analyze Kubernetes events. It provides an alternative to complex and heavy solutions like the ELK stack, offering a simple, single-binary solution perfect for clusters of all sizes.

## The Problem

Monitoring Kubernetes events is crucial for maintaining cluster health. However, traditional methods often fall short:

1.  **High Latency**: Relying on centralized logging pipelines (e.g., shipping to a shared Elasticsearch) can introduce significant delays (15+ minutes), making real-time analysis impossible.
2.  **Operational Overhead**: Setting up and maintaining a full-blown logging pipeline for smaller or temporary clusters is often impractical and costly.
3.  **Lack of Standardization**: Each organization builds its own custom dashboards and queries, leading to a fragmented and inconsistent monitoring experience across the community.

## Our Solution

Kube Event Analyzer addresses these problems with a streamlined, all-in-one approach:

-   **Real-Time Collection**: It uses the Kubernetes `WATCH` API to subscribe to events directly, ensuring minimal latency.
-   **Efficient Storage**: Events are initially stored in an in-memory DuckDB database for fast ingestion. They are then periodically archived into compressed Parquet files (`zstd` compression) for long-term storage, offering a great balance between performance and disk space.
-   **Automated Data Lifecycle**: The tool automatically manages data retention. It archives recent events into Parquet files at a configurable interval and prunes the oldest files when a predefined storage limit is reached.
-   **Simplified Architecture**: It runs as a single binary, containing both the event collector and a future API server for querying data. This eliminates the need for external databases or complex pipelines.
-   **Powerful Analytics**: By leveraging DuckDB, it can query both the real-time in-memory data and the historical Parquet files simultaneously, providing a unified view for analysis.

## Architecture

The project is a single Go binary that consists of two main components:

1.  **Collector & Storage Manager**: A background service that:
    -   Connects to the Kubernetes API server.
    -   Uses a `RetryWatcher` to reliably watch for cluster events, automatically resuming from the last known `resourceVersion` upon restart.
    -   Appends incoming events to a `kube_events` table in DuckDB.
    -   At a regular interval, archives the events from the DuckDB table to a ZSTD-compressed Parquet file using a lock-free table-swapping mechanism.
    -   Enforces a storage limit by deleting the oldest Parquet files when the total size exceeds the configured capacity.

2.  **API Server & Web UI (Future Work)**:
    -   An API endpoint will be exposed to receive SQL queries.
    -   These queries will be executed by DuckDB against the collection of Parquet files and the current in-memory database.
    -   A React-based web interface will be built to provide a powerful query builder and visualization dashboards for operators.

### Data Flow Diagram

```mermaid
graph TD
    A["K8s API Server"]
		A -- "Events" --> B("Collector")    
    subgraph kube-event-analyzer
    B -- "Appends" --> C{"DuckDB Table
    kube_events"}
    
    subgraph Data Lifecycle Manager
        D["Ticker: every 'interval'"] --> E{"Archive & Retain"}
        E -- "1- Table Swap" --> C
        E -- "2- Archives to" --> F["Parquet Files (zstd compressed)"]
        E -- "3- Deletes oldest" --> F
    end
    subgraph Analysis
        G["API Server"] --> H{"DuckDB"}
        H -- "Hot Queries" --> C
        H -- "Cold Queries" --> F
    end 
    end
    J["User"] --> I
    I["React Web UI"] -- "SQL Query" --> G




```

## Project Structure

```
.
├── data/                  # Default directory for DuckDB files and Parquet archives
├── internal/
│   ├── api/               # API server logic
│   ├── collector/         # Kubernetes event collection logic
│   └── storage/           # DuckDB and Parquet storage management
├── src/                   # React frontend code
├── package.json
├── go.mod
├── go.sum
└── main.go                # Application entrypoint
```

## Getting Started

*(This section can be expanded as the project matures)*

1.  **Prerequisites**:
    -   Go 1.24+
    -   Access to a Kubernetes cluster (a valid `kubeconfig` file)

2.  **Build & Run**:
    
    ```shell
    # Run the application
    go run main.go
    ```
