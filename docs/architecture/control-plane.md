# Control Plane Architecture

The Control Plane is the "brain" of AgentStack. It manages the persistent state of the system, coordinates background tasks, and ensures that the actual state of the Kubernetes cluster matches the desired state defined by the user.

## Components

### 1. State Management (PostgreSQL)
PostgreSQL is the single source of truth for all metadata.
- **Schema Management**: Managed via SQL migrations in `agentstack/migrations`.
- **Type-Safe Access**: Uses [sqlc](https://sqlc.dev/) to generate type-safe Go code from SQL queries.
- **Multi-Tenancy**: Implemented using a `tenant_id` column on all major tables, enforced via middleware.

### 2. Caching & Task Queue (Redis)
Redis is used for high-performance caching and asynchronous task coordination.
- **RBAC Cache**: Stores compiled permission sets for rapid access during API calls.
- **Quota Tracking**: Real-time tracking of resource usage and rate limits.
- **Evaluation Queue**: Buffers agent traces and evaluation data before processing.

### 3. Reconciliation Loop
The Control Plane implements a reconciliation pattern similar to Kubernetes controllers.
- **Watchers**: Monitors the database for changes in agent definitions.
- **Reconcilers**: Compares the database state with the Kubernetes state and performs the necessary actions (Create, Update, Delete).
- **Status Updates**: Periodically polls Kubernetes for agent health and updates the database with the latest status and URLs.

## Internal Structure (`agentstack/internal/infrastructure`)

- **`database/`**: Contains the PostgreSQL connection pool, repository implementations, and `sqlc` generated code.
- **`cache/`**: Contains the Redis client and specialized cache implementations (Quota, RBAC).
- **`worker/`**: Implements background workers for tasks like cleanup, usage aggregation, and evaluation processing.

## Data Consistency

AgentStack follows an **Eventual Consistency** model for deployments:
1. The API writes the desired state to PostgreSQL and returns a `202 Accepted`.
2. A background worker or the reconciliation loop detects the change.
3. The worker interacts with the Kubernetes API to apply the changes.
4. Once Kubernetes confirms the resources are ready, the worker updates the agent status in PostgreSQL to `Ready`.

## Scalability

The Control Plane is designed to be stateless and can be scaled horizontally.
- **Leader Election**: For tasks that must only run once (like certain cleanup jobs), the Control Plane uses Kubernetes-native leader election.
- **Database Connection Pooling**: Uses `pgx` with optimized pooling settings to handle high concurrency.
