# Google Maps Scraper

A high-performance, open-source Google Maps scraper written in Go. Extracts detailed business information from Google Maps search results at scale.

## ⚠️ IMPORTANT: Repository Info

**This is a FORK repository: `sadewadee/google-scraper`**

- **ALWAYS create issues in THIS repo**: `sadewadee/google-scraper`
- **NEVER create issues in upstream**: `gosom/google-maps-scraper`
- When using `gh issue create`, always use: `--repo sadewadee/google-scraper`

```bash
# CORRECT
gh issue create --repo sadewadee/google-scraper --title "..." --body "..."

# WRONG - DO NOT DO THIS
gh issue create --repo gosom/google-maps-scraper --title "..." --body "..."
```

## Quick Reference

```bash
# Build (standard with Playwright)
go build -o gmaps-scraper

# Build (alternative with Rod - faster startup)
go build -tags rod -o gmaps-scraper-rod

# Run CLI mode
./gmaps-scraper -input queries.txt -results results.csv -c 4 -depth 10

# Run Web UI
./gmaps-scraper -web -addr :8080

# Docker
docker run -v $PWD/queries.txt:/queries.txt -v $PWD/results.csv:/results.csv \
  gosom/google-maps-scraper -input /queries.txt -results /results.csv
```

## Architecture

```
main.go                 # Entry point, config parsing, runner initialization
├── runner/             # Core execution logic
│   ├── runner.go       # Config struct and CLI flags
│   ├── filerunner/     # CLI mode (input file → output file)
│   ├── managerrunner/  # Web UI server and job management (Manager)
│   ├── workerrunner/   # Worker mode (connects to Manager)
│   ├── databaserunner/ # Distributed scraping with PostgreSQL
│   └── lambdaaws/      # AWS Lambda execution
├── gmaps/              # Google Maps domain logic
│   ├── entry.go        # Entry struct (business data model)
│   ├── job.go          # Search job (processing results pages)
│   └── place.go        # Place job (extracting single listing details)
├── internal/           # Internal packages
│   ├── api/            # HTTP API handlers and router
│   ├── cache/          # Redis cache layer
│   ├── domain/         # Domain models (Job, Worker, Result)
│   ├── mq/             # RabbitMQ publisher/consumer
│   ├── queue/          # Redis/Asynq job queue (fallback)
│   ├── repository/     # PostgreSQL/SQLite data access
│   ├── service/        # Business logic services
│   └── worker/         # Worker runner implementation
└── deduper/            # Deduplication logic
```

## Data Flow

1. **Input**: Search queries (keywords) or geo-coordinates
2. **Seeding**: Runner creates "Seed Jobs" (Google Maps search URLs)
3. **Discovery**: Browser loads results, scrolls to find all listings
4. **Queuing**: Listings converted to "Place Jobs"
5. **Extraction**: Visit each place, parse DOM or JSON state (`APP_INITIALIZATION_STATE`)
6. **Email crawl** (optional): Visits business websites to find emails
7. **Output**: Data sent to configured writer (CSV, JSON, PostgreSQL, S3, LeadsDB)

## Key Technologies

- **Language**: Go 1.25+
- **Scraping Engine**: `scrapemate`
- **Browser Automation**: Playwright-Go (default) or Go-Rod (build tag: `rod`)
- **Storage**: PostgreSQL (distributed), SQLite (Web UI)
- **Job Queue**: RabbitMQ (preferred) or Redis/Asynq (fallback)
- **Cache**: Redis (for dashboard performance)
- **Infrastructure**: Docker, AWS Lambda

## Operation Modes

| Mode | Flag | Description |
|------|------|-------------|
| **Manager** | `-manager` | API server + Web UI (no scraping) - **RECOMMENDED** |
| **Worker** | `-worker` | Scraper that connects to Manager via RabbitMQ/Redis |
| CLI (deprecated) | `-input` | Input file → Output file |
| Distributed (deprecated) | `-dsn` | PostgreSQL-coordinated instances |
| Serverless | `-aws-lambda` | AWS Lambda deployment |

### Recommended Architecture (Manager/Worker with RabbitMQ)

```bash
# Manager (API + Dashboard)
./gmaps-scraper -manager -dsn 'postgres://...' \
  -rabbitmq-url 'amqp://guest:guest@localhost:5672/' \
  -redis-addr localhost:6379 -addr :8080

# Worker (can run multiple instances)
./gmaps-scraper -worker -manager-url http://localhost:8080 \
  -rabbitmq-url 'amqp://guest:guest@localhost:5672/' \
  -redis-addr localhost:6379

# Docker Compose (starts postgres, redis, rabbitmq, manager, worker)
docker-compose up -d
docker-compose up -d --scale worker=4  # Scale to 4 workers
```

**Priority order for job queue:**
1. RabbitMQ (preferred) - durable, priority queues, better for production
2. Redis/Asynq (fallback) - if RabbitMQ unavailable
3. HTTP polling (last resort) - if neither queue is available

## Key Configuration Flags

| Flag | Description |
|------|-------------|
| `-manager` | Run as Manager (API + Web UI) |
| `-worker` | Run as Worker (connects to Manager) |
| `-manager-url` | Manager API URL for worker mode |
| `-rabbitmq-url` | RabbitMQ connection URL (preferred job queue) |
| `-redis-addr` | Redis address for cache and deduplication |
| `-dsn` | PostgreSQL connection string |
| `-input` | Input file with queries |
| `-results` | Output file path |
| `-c` | Concurrency level |
| `-depth` | Scroll depth for results |
| `-lang` | Language code |
| `-geo` | Geo-coordinates |
| `-zoom` | Map zoom level |
| `-radius` | Search radius |
| `-json` | JSON output format |
| `-email` | Enable email crawling |
| `-extra-reviews` | Collect detailed reviews |
| `-proxies` | HTTP/SOCKS5 proxy list |
| `-spawner` | Auto-spawn workers: docker, swarm, lambda, none |
| `-spawner-image` | Docker image for spawned workers |
| `-spawner-max-workers` | Max concurrent spawned workers |

## Entry Data Model

The `gmaps.Entry` struct (in `gmaps/entry.go`) contains all scraped business data:
- Name, address, phone, website
- Reviews, ratings, operating hours
- Categories, coordinates
- Email (if crawling enabled)

## Plugin System

Custom output writers can be implemented as Go plugins. See `examples/plugins` for reference.

## Development Notes

- **Hybrid Parsing**: Tries JSON embedded in page source first, falls back to DOM parsing
- **Smart Scrolling**: Custom logic to scroll through dynamic search sidebars
- **Deduplication**: Redis-backed (distributed) or in-memory to avoid redundant processing
- **Job Queue Priority**: RabbitMQ > Redis/Asynq > HTTP polling (graceful fallback)
- **Caching**: Redis cache layer for Dashboard API endpoints (30s-120s TTL)
- **Database Normalization**: `results` table auto-populates `business_listings` + `emails` via trigger

## Documentation

For detailed technical documentation, see:

- **[Architecture Documentation](docs/ARCHITECTURE.md)** - Complete technical reference including:
  - Database normalization layer (tables, triggers, views)
  - Cache implementation (Redis, TTLs, invalidation)
  - DSN Bridge mechanism (Dashboard → DSN workers)
  - Worker deduplication (Redis SETNX)
  - API endpoint reference (Jobs, Workers, Results)
  - Message queue architecture (RabbitMQ)

- **[Auto-Spawn Workers](docs/AUTO_SPAWN.md)** - On-demand worker spawning:
  - Docker spawner for local development
  - Docker Swarm spawner for Dokploy deployments
  - AWS Lambda spawner for serverless workloads
  - Configuration flags and best practices
