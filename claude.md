# Google Maps Scraper

A high-performance, open-source Google Maps scraper written in Go. Extracts detailed business information from Google Maps search results at scale.

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
│   ├── webrunner/      # Web UI server and job management
│   ├── databaserunner/ # Distributed scraping with PostgreSQL
│   └── lambdaaws/      # AWS Lambda execution
├── gmaps/              # Google Maps domain logic
│   ├── entry.go        # Entry struct (business data model)
│   ├── job.go          # Search job (processing results pages)
│   └── place.go        # Place job (extracting single listing details)
├── web/                # Web server (handlers, API)
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
- **Infrastructure**: Docker, AWS Lambda

## Operation Modes

| Mode | Flag | Description |
|------|------|-------------|
| CLI | (default) | Input file → Output file |
| Web UI | `-web` | Local dashboard at specified address |
| Distributed | `-dsn` | PostgreSQL-coordinated multiple instances |
| Serverless | `-aws-lambda` | AWS Lambda deployment |

## Key Configuration Flags

| Flag | Description |
|------|-------------|
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
- **Deduplication**: In-memory or database-backed to avoid redundant processing
