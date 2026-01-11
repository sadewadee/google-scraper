# Dokploy Deployment Guide

## Quick Start dengan Dokploy

### 1. Create Application di Dokploy

1. Login ke Dokploy dashboard
2. Create new **Application** → **Docker Compose**
3. Connect repository: `https://github.com/sadewadee/google-scraper`
4. Set compose file: `docker-compose.yml`

### 2. Environment Variables

Di Dokploy, set environment variables ini:

```env
# Required
POSTGRES_USER=gmaps
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_DB=gmaps

# Scraper settings
CONCURRENCY=4
DEPTH=10
WEB_PORT=8080
```

### 3. Domain & Port

- Set domain: `scraper.yourdomain.com`
- Map port: `8080` → Web UI

### 4. Deploy

Click **Deploy** dan tunggu build selesai.

---

## Architecture di Dokploy

```
┌─────────────────────────────────────────────────────┐
│                    Dokploy                          │
├─────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  scraper-   │  │     db      │  │   migrate   │ │
│  │    web      │  │ (PostgreSQL)│  │             │ │
│  │   :8080     │  │   :5432     │  │  (one-shot) │ │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘ │
│         │                │                          │
│         └────────────────┘                          │
│              gmaps-network                          │
└─────────────────────────────────────────────────────┘
           │
           ▼
    ┌──────────────┐
    │   Traefik    │ ← Dokploy's reverse proxy
    │ (Auto HTTPS) │
    └──────────────┘
           │
           ▼
    https://scraper.yourdomain.com
```

---

## Modes

### Mode 1: Web UI Only (Default)

```bash
# Sudah termasuk di docker-compose.yml default
docker compose up -d
```

Access: `http://localhost:8080`

### Mode 2: Distributed Workers

Untuk scale 1M+ places, jalankan multiple workers:

```bash
# Start dengan worker profile
docker compose --profile distributed up -d

# Scale workers
docker compose --profile distributed up -d --scale worker=5
```

### Mode 3: CLI Only (One-off Job)

```bash
# Run single scrape job
docker compose run --rm scraper-web \
  -input /queries/my-queries.txt \
  -results /results/output.csv \
  -c 4 -depth 10
```

---

## Volume Mounts

| Volume | Path in Container | Purpose |
|--------|-------------------|---------|
| `postgres_data` | `/var/lib/postgresql/data` | Database persistence |
| `scraper_data` | `/data` | SQLite jobs.db (Web UI) |
| `./queries` | `/queries` | Input query files |
| `./results` | `/results` | Output CSV/JSON files |

---

## Resource Recommendations

| Scale | Workers | Memory/Worker | Total RAM |
|-------|---------|---------------|-----------|
| Small (<1K) | 1 | 2GB | 4GB |
| Medium (1K-10K) | 2-4 | 2GB | 8-12GB |
| Large (10K-100K) | 4-8 | 2GB | 16-20GB |
| Enterprise (100K+) | 10+ | 2GB | 32GB+ |

---

## Dokploy-Specific Settings

### Health Check
```yaml
# Sudah ada di docker-compose.yml
healthcheck:
  test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

### Restart Policy
```yaml
restart: unless-stopped
```

### Memory Limits
```yaml
deploy:
  resources:
    limits:
      memory: 4G
    reservations:
      memory: 1G
```

---

## Troubleshooting

### Container Crash (OOM)
```bash
# Increase memory limit
# Edit docker-compose.yml → deploy.resources.limits.memory
```

### Database Connection Failed
```bash
# Check if db is healthy
docker compose logs db

# Restart migration
docker compose restart migrate
```

### Browser Fails to Start
```bash
# Check chromium dependencies
docker compose exec scraper-web ls -la /root/.cache/rod

# Use Playwright image instead
# Change Dockerfile.rod → Dockerfile in docker-compose.yml
```

### View Logs
```bash
docker compose logs -f scraper-web
docker compose logs -f db
```

---

## Backup & Restore

### Backup Database
```bash
docker compose exec db pg_dump -U gmaps gmaps > backup.sql
```

### Restore Database
```bash
cat backup.sql | docker compose exec -T db psql -U gmaps gmaps
```

### Backup Results
```bash
# Results are in ./results volume
cp -r ./results ./backup-results-$(date +%Y%m%d)
```

---

## Upgrade

```bash
# Pull latest
git pull origin main

# Rebuild
docker compose build --no-cache

# Restart
docker compose up -d
```
