# Auto-Spawn Workers

Auto-spawn enables the Manager to automatically create worker containers when jobs are submitted. Workers are created on-demand, process the job, and terminate when finished. No idle workers means efficient resource usage.

## Supported Spawner Types

| Type | Flag | Description |
|------|------|-------------|
| **none** | `-spawner none` | Disabled (default) - workers must be started manually |
| **docker** | `-spawner docker` | Local Docker containers |
| **swarm** | `-spawner swarm` | Docker Swarm services (for Dokploy) |
| **lambda** | `-spawner lambda` | AWS Lambda functions |

## Quick Start

### Docker Spawner (Local Development)

```bash
# Start Manager with Docker auto-spawn
./gmaps-scraper -manager \
  -dsn 'postgres://user:pass@localhost/gmaps' \
  -rabbitmq-url 'amqp://guest:guest@localhost:5672/' \
  -redis-addr localhost:6379 \
  -spawner docker \
  -spawner-image gmaps-scraper:latest \
  -spawner-network gmaps-network
```

When a job is created, Manager will:
1. Create a new Docker container
2. Connect it to the specified network
3. Pass RabbitMQ/Redis connection info
4. Container processes the job and terminates

### Docker Swarm (Dokploy)

For Dokploy deployments using Docker Swarm:

```bash
# Start Manager with Swarm auto-spawn
./gmaps-scraper -manager \
  -dsn 'postgres://user:pass@postgres:5432/gmaps' \
  -rabbitmq-url 'amqp://guest:guest@rabbitmq:5672/' \
  -redis-addr redis:6379 \
  -spawner swarm \
  -spawner-image gmaps-scraper:latest \
  -spawner-network dokploy-network \
  -spawner-manager-url http://manager:8080
```

**PENTING untuk Dokploy:**
- `-spawner-manager-url` HARUS diset ke nama service Docker (e.g., `http://manager:8080`)
- Jangan gunakan `localhost` karena spawned workers tidak bisa mengaksesnya
- Network harus overlay dan attachable

#### Dokploy docker-compose.yml

```yaml
version: '3.8'

services:
  manager:
    image: gmaps-scraper:latest
    command: >
      -manager
      -dsn postgres://gmaps:password@postgres:5432/gmaps?sslmode=disable
      -rabbitmq-url amqp://guest:guest@rabbitmq:5672/
      -redis-addr redis:6379
      -spawner swarm
      -spawner-image gmaps-scraper:latest
      -spawner-network gmaps-network
      -spawner-manager-url http://manager:8080
      -spawner-max-workers 10
    ports:
      - "8080:8080"
    networks:
      - gmaps-network
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: replicated
      replicas: 1
      placement:
        constraints:
          - node.role == manager

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: gmaps
      POSTGRES_PASSWORD: password
      POSTGRES_DB: gmaps
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - gmaps-network
    deploy:
      mode: replicated
      replicas: 1

  redis:
    image: redis:7-alpine
    networks:
      - gmaps-network
    deploy:
      mode: replicated
      replicas: 1

  rabbitmq:
    image: rabbitmq:3-management-alpine
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    networks:
      - gmaps-network
    deploy:
      mode: replicated
      replicas: 1

networks:
  gmaps-network:
    driver: overlay
    attachable: true

volumes:
  postgres_data:
```

**Checklist untuk Dokploy:**
- [x] Mount Docker socket: `/var/run/docker.sock:/var/run/docker.sock`
- [x] Use overlay network with `attachable: true`
- [x] Manager runs on Swarm manager node (`node.role == manager`)
- [x] Set `-spawner-manager-url` to service name (NOT localhost)
- [x] Image `gmaps-scraper:latest` accessible from all nodes

### AWS Lambda

For serverless deployments using AWS Lambda:

```bash
# Start Manager with Lambda auto-spawn
./gmaps-scraper -manager \
  -dsn 'postgres://user:pass@rds.amazonaws.com/gmaps' \
  -rabbitmq-url 'amqps://user:pass@rabbitmq.amazonaws.com:5671/' \
  -redis-addr elasticache.amazonaws.com:6379 \
  -spawner lambda \
  -spawner-lambda-function gmaps-worker \
  -spawner-lambda-region us-east-1 \
  -spawner-lambda-max-conc 100
```

#### Lambda Worker Setup

1. Build Lambda worker binary:
```bash
GOOS=linux GOARCH=amd64 go build -o bootstrap -tags lambda.norpc
zip function.zip bootstrap
```

2. Create Lambda function with the following handler:
```go
// The Lambda function receives job info and connects back to Manager
type LambdaPayload struct {
    JobID       string `json:"job_id"`
    Priority    int    `json:"priority"`
    ManagerURL  string `json:"manager_url"`
    RabbitMQURL string `json:"rabbitmq_url,omitempty"`
    RedisAddr   string `json:"redis_addr,omitempty"`
    Concurrency int    `json:"concurrency,omitempty"`
}
```

3. Configure Lambda:
   - Runtime: `provided.al2023`
   - Memory: 1024MB+
   - Timeout: 15 minutes
   - VPC: Same VPC as RDS/ElastiCache

## Configuration Flags

### Common Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-spawner` | `none` | Spawner type: none, docker, swarm, lambda |
| `-spawner-image` | `gmaps-scraper:latest` | Docker image for workers |
| `-spawner-network` | `gmaps-network` | Docker network to attach workers |
| `-spawner-concurrency` | `4` | Concurrency per spawned worker |
| `-spawner-max-workers` | `0` | Max concurrent workers (0 = unlimited) |
| `-spawner-auto-remove` | `true` | Auto-remove containers after exit |
| `-spawner-manager-url` | (auto) | Manager URL for workers (REQUIRED for Dokploy) |

### Docker/Swarm Specific

| Flag | Default | Description |
|------|---------|-------------|
| `-spawner-image` | `gmaps-scraper:latest` | Docker image name |
| `-spawner-network` | `gmaps-network` | Network to attach containers |
| `-spawner-manager-url` | (auto) | **PENTING**: Untuk Dokploy, set ke `http://<service-name>:8080` |

### Lambda Specific

| Flag | Default | Description |
|------|---------|-------------|
| `-spawner-lambda-function` | `` | Lambda function name or ARN |
| `-spawner-lambda-region` | `` | AWS region |
| `-spawner-lambda-invocation` | `Event` | Invocation type: Event (async) or RequestResponse (sync) |
| `-spawner-lambda-max-conc` | `100` | Max concurrent Lambda invocations |

## How It Works

### Flow Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Dashboard     │────▶│    Manager      │────▶│    Spawner      │
│   (Job Create)  │     │   (API Server)  │     │  (Docker/Swarm/ │
└─────────────────┘     └────────┬────────┘     │    Lambda)      │
                                 │              └────────┬────────┘
                                 │                       │
                                 ▼                       ▼
                        ┌────────────────┐      ┌────────────────┐
                        │   RabbitMQ/    │◀─────│    Worker      │
                        │   Redis Queue  │      │  (Container/   │
                        └────────────────┘      │   Function)    │
                                                └───────┬────────┘
                                                        │
                                                        ▼
                                               ┌────────────────┐
                                               │   PostgreSQL   │
                                               │   (Results)    │
                                               └────────────────┘
```

### Spawn Process

1. **Job Created**: User creates job via Dashboard
2. **Job Enqueued**: Manager saves job to DB and publishes to RabbitMQ
3. **Spawn Triggered**: Manager calls spawner asynchronously (non-blocking)
4. **Worker Started**: Container/function created with:
   - Job ID in environment
   - Manager URL for API calls
   - RabbitMQ/Redis connection info
5. **Job Processed**: Worker picks up job from queue, scrapes data
6. **Results Submitted**: Worker submits results to Manager API
7. **Worker Exits**: Container terminates (auto-removed if configured)

### Container Labels

Spawned Docker containers have these labels:
- `gmaps.worker=true`
- `gmaps.job-id=<uuid>`
- `gmaps.priority=<0-10>`
- `gmaps.spawner=docker|swarm`

## Best Practices

### Resource Limits

Set maximum workers to prevent resource exhaustion:
```bash
-spawner-max-workers 10
```

### Network Configuration

Ensure the network allows:
- Workers → Manager (HTTP API, port 8080)
- Workers → RabbitMQ (port 5672)
- Workers → Redis (port 6379)
- Workers → PostgreSQL (port 5432)

### Monitoring

Monitor spawned workers:
```bash
# Docker
docker ps --filter label=gmaps.worker=true

# Swarm
docker service ls --filter label=gmaps.worker=true
```

### Cleanup

Clean up completed/failed workers:
```bash
# Remove all stopped gmaps workers
docker rm $(docker ps -a --filter label=gmaps.worker=true --filter status=exited -q)
```

## Troubleshooting

### Workers Not Starting

1. Check Docker socket permissions:
   ```bash
   ls -la /var/run/docker.sock
   ```

2. Verify image exists:
   ```bash
   docker images gmaps-scraper:latest
   ```

3. Check network exists:
   ```bash
   docker network ls | grep gmaps-network
   ```

### Workers Can't Connect

1. Verify network connectivity:
   ```bash
   docker run --rm --network gmaps-network busybox ping manager
   ```

2. Check Manager URL is accessible from worker network

### Lambda Timeouts

- Increase Lambda timeout (max 15 minutes)
- Reduce depth/keywords per job
- Use smaller batch sizes

## Architecture Comparison

| Feature | Docker | Swarm | Lambda |
|---------|--------|-------|--------|
| Startup time | ~2s | ~5s | ~1s (cold) |
| Scale limit | Host resources | Cluster resources | 1000 concurrent |
| Cost | Fixed (server) | Fixed (cluster) | Pay-per-use |
| Best for | Development | Production (Dokploy) | Burst workloads |
| Persistence | Container logs | Service logs | CloudWatch |
