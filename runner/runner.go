package runner

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"

	"github.com/sadewadee/google-scraper/s3uploader"
	"github.com/sadewadee/google-scraper/tlmt"
	"github.com/sadewadee/google-scraper/tlmt/gonoop"
	"github.com/sadewadee/google-scraper/tlmt/goposthog"
)

const (
	RunModeFile = iota + 1
	RunModeDatabase
	RunModeDatabaseProduce
	RunModeInstallPlaywright
	RunModeAwsLambda
	RunModeAwsLambdaInvoker
	RunModeManager
	RunModeWorker
)

var (
	ErrInvalidRunMode = errors.New("invalid run mode")
)

type Runner interface {
	Run(context.Context) error
	Close(context.Context) error
}

type S3Uploader interface {
	Upload(ctx context.Context, bucketName, key string, body io.Reader) error
}

type Config struct {
	Concurrency              int
	CacheDir                 string
	MaxDepth                 int
	InputFile                string
	ResultsFile              string
	JSON                     bool
	LangCode                 string
	Debug                    bool
	Dsn                      string
	ProduceOnly              bool
	ExitOnInactivityDuration time.Duration
	Email                    bool
	CustomWriter             string
	GeoCoordinates           string
	Zoom                     int
	RunMode                  int
	DisableTelemetry         bool
	AwsLamdbaRunner          bool
	DataFolder               string
	Proxies                  []string
	AwsAccessKey             string
	AwsSecretKey             string
	AwsRegion                string
	S3Uploader               S3Uploader
	S3Bucket                 string
	AwsLambdaInvoker         bool
	FunctionName             string
	AwsLambdaChunkSize       int
	FastMode                 bool
	Radius                   float64
	Addr                     string
	DisablePageReuse         bool
	ExtraReviews             bool
	LeadsDBAPIKey            string
	// Manager/Worker mode flags
	ManagerMode bool
	WorkerMode  bool
	ManagerURL  string
	WorkerID    string
	// StaticFolder is the path to static frontend files
	StaticFolder string

	// Redis configuration for cache and deduplication
	RedisURL  string
	RedisAddr string
	RedisPass string
	RedisDB   int

	// RabbitMQ configuration for job queue
	RabbitMQURL string

	// ProxyGate flags
	ProxyGateEnabled         bool
	ProxyGateAddr            string
	ProxyGateSources         []string
	ProxyGateRefreshInterval time.Duration

	// Email validation (Moribouncer)
	EmailValidatorURL string
	EmailValidatorKey string

	// Migration flags
	Migrate       bool // Run migration only, then exit
	MigrateStatus bool // Check migration status and exit

	// Auto-spawn configuration (Manager mode)
	SpawnerType        string            // none, docker, swarm, lambda
	SpawnerImage       string            // Docker image for worker containers
	SpawnerNetwork     string            // Docker network to attach workers
	SpawnerConcurrency int               // Concurrency per spawned worker
	SpawnerMaxWorkers  int               // Max concurrent workers (0 = unlimited)
	SpawnerAutoRemove  bool              // Auto-remove containers after exit
	SpawnerLabels      map[string]string // Labels for spawned containers
	SpawnerConstraints []string          // Swarm placement constraints
	SpawnerManagerURL  string            // Manager URL for spawned workers (default: auto-detect)

	// AWS Lambda spawner configuration
	SpawnerLambdaFunction   string // Lambda function name/ARN
	SpawnerLambdaRegion     string // AWS region (defaults to AwsRegion)
	SpawnerLambdaInvocation string // Event (async) or RequestResponse (sync)
	SpawnerLambdaMaxConc    int    // Max concurrent Lambda invocations
}

func ParseConfig() *Config {
	cfg := Config{}

	if os.Getenv("PLAYWRIGHT_INSTALL_ONLY") == "1" {
		cfg.RunMode = RunModeInstallPlaywright

		return &cfg
	}

	var (
		proxies          string
		proxyGateSources string
	)

	flag.IntVar(&cfg.Concurrency, "c", min(runtime.NumCPU()/2, 1), "sets the concurrency [default: half of CPU cores]")
	flag.StringVar(&cfg.CacheDir, "cache", "cache", "sets the cache directory [no effect at the moment]")
	flag.IntVar(&cfg.MaxDepth, "depth", 10, "maximum scroll depth in search results [default: 10]")
	flag.StringVar(&cfg.ResultsFile, "results", "stdout", "path to the results file [default: stdout]")
	flag.StringVar(&cfg.InputFile, "input", "", "path to the input file with queries (one per line) [default: empty]")
	flag.StringVar(&cfg.LangCode, "lang", "en", "language code for Google (e.g., 'de' for German) [default: en]")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable headful crawl (opens browser window) [default: false]")
	flag.StringVar(&cfg.Dsn, "dsn", "", "database connection string [only valid with database provider]")
	flag.BoolVar(&cfg.ProduceOnly, "produce", false, "produce seed jobs only (requires dsn)")
	flag.DurationVar(&cfg.ExitOnInactivityDuration, "exit-on-inactivity", 0, "exit after inactivity duration (e.g., '5m')")
	flag.BoolVar(&cfg.JSON, "json", false, "produce JSON output instead of CSV")
	flag.BoolVar(&cfg.Email, "email", false, "extract emails from websites")
	flag.StringVar(&cfg.CustomWriter, "writer", "", "use custom writer plugin (format: 'dir:pluginName')")
	flag.StringVar(&cfg.GeoCoordinates, "geo", "", "set geo coordinates for search (e.g., '37.7749,-122.4194')")
	flag.IntVar(&cfg.Zoom, "zoom", 15, "set zoom level (0-21) for search")
	flag.StringVar(&cfg.DataFolder, "data-folder", "webdata", "data folder for web runner")
	flag.StringVar(&proxies, "proxies", "", "comma separated list of proxies to use in the format protocol://user:pass@host:port example: socks5://localhost:9050 or http://user:pass@localhost:9050")
	flag.BoolVar(&cfg.AwsLamdbaRunner, "aws-lambda", false, "run as AWS Lambda function")
	flag.BoolVar(&cfg.AwsLambdaInvoker, "aws-lambda-invoker", false, "run as AWS Lambda invoker")
	flag.StringVar(&cfg.FunctionName, "function-name", "", "AWS Lambda function name")
	flag.StringVar(&cfg.AwsAccessKey, "aws-access-key", "", "AWS access key")
	flag.StringVar(&cfg.AwsSecretKey, "aws-secret-key", "", "AWS secret key")
	flag.StringVar(&cfg.AwsRegion, "aws-region", "", "AWS region")
	flag.StringVar(&cfg.S3Bucket, "s3-bucket", "", "S3 bucket name")
	flag.IntVar(&cfg.AwsLambdaChunkSize, "aws-lambda-chunk-size", 100, "AWS Lambda chunk size")
	flag.BoolVar(&cfg.FastMode, "fast-mode", false, "fast mode (reduced data collection)")
	flag.Float64Var(&cfg.Radius, "radius", 10000, "search radius in meters. Default is 10000 meters")
	flag.StringVar(&cfg.Addr, "addr", ":8080", "address to listen on for web server")
	flag.BoolVar(&cfg.DisablePageReuse, "disable-page-reuse", false, "disable page reuse in playwright")
	flag.BoolVar(&cfg.ExtraReviews, "extra-reviews", false, "enable extra reviews collection")
	flag.StringVar(&cfg.LeadsDBAPIKey, "leadsdb-api-key", "", "LeadsDB API key for exporting results to LeadsDB")
	flag.BoolVar(&cfg.ManagerMode, "manager", false, "run as manager (API only, no scraping)")
	flag.BoolVar(&cfg.WorkerMode, "worker", false, "run as worker (connects to manager)")
	flag.StringVar(&cfg.ManagerURL, "manager-url", "http://localhost:8080", "manager API URL for worker mode")
	flag.StringVar(&cfg.WorkerID, "worker-id", "", "worker ID (auto-generated if empty)")
	flag.StringVar(&cfg.StaticFolder, "static-folder", "", "path to static frontend files")

	// Redis flags
	flag.StringVar(&cfg.RedisURL, "redis-url", "", "Redis connection URL (redis://user:pass@host:port/db)")
	flag.StringVar(&cfg.RedisAddr, "redis-addr", "", "Redis address (host:port)")
	flag.StringVar(&cfg.RedisPass, "redis-pass", "", "Redis password")
	flag.IntVar(&cfg.RedisDB, "redis-db", 0, "Redis database number")

	// RabbitMQ flags
	flag.StringVar(&cfg.RabbitMQURL, "rabbitmq-url", "", "RabbitMQ connection URL (amqp://user:pass@host:port/vhost)")

	// ProxyGate flags
	flag.BoolVar(&cfg.ProxyGateEnabled, "proxygate", false, "enable embedded proxy gateway")
	flag.StringVar(&cfg.ProxyGateAddr, "proxygate-addr", "localhost:8081", "proxy gateway listen address")
	flag.StringVar(&proxyGateSources, "proxygate-sources", "", "comma-separated proxy source URLs (uses defaults if empty)")
	flag.DurationVar(&cfg.ProxyGateRefreshInterval, "proxygate-refresh", 10*time.Minute, "proxy refresh interval")

	// Email validation flags
	flag.StringVar(&cfg.EmailValidatorURL, "email-validator-url", "", "Email validation API URL (e.g., https://api.moribouncer.com/v1)")
	flag.StringVar(&cfg.EmailValidatorKey, "email-validator-key", "", "Email validation API key (Moribouncer)")

	// Migration flags
	flag.BoolVar(&cfg.Migrate, "migrate", false, "Run auto-migration and exit")
	flag.BoolVar(&cfg.MigrateStatus, "migrate-status", false, "Check migration status and exit")

	// Auto-spawn flags (Manager mode)
	flag.StringVar(&cfg.SpawnerType, "spawner", "none", "Worker spawner type: none, docker, swarm, lambda")
	flag.StringVar(&cfg.SpawnerImage, "spawner-image", "gmaps-scraper:latest", "Docker image for spawned workers")
	flag.StringVar(&cfg.SpawnerNetwork, "spawner-network", "gmaps-network", "Docker network for spawned workers")
	flag.IntVar(&cfg.SpawnerConcurrency, "spawner-concurrency", 4, "Concurrency per spawned worker")
	flag.IntVar(&cfg.SpawnerMaxWorkers, "spawner-max-workers", 0, "Max concurrent workers (0 = unlimited)")
	flag.BoolVar(&cfg.SpawnerAutoRemove, "spawner-auto-remove", true, "Auto-remove containers after exit")
	flag.StringVar(&cfg.SpawnerManagerURL, "spawner-manager-url", "", "Manager URL for spawned workers (e.g., http://manager:8080)")
	flag.StringVar(&cfg.SpawnerLambdaFunction, "spawner-lambda-function", "", "AWS Lambda function name/ARN")
	flag.StringVar(&cfg.SpawnerLambdaRegion, "spawner-lambda-region", "", "AWS region for Lambda (defaults to -aws-region)")
	flag.StringVar(&cfg.SpawnerLambdaInvocation, "spawner-lambda-invocation", "Event", "Lambda invocation type: Event (async) or RequestResponse (sync)")
	flag.IntVar(&cfg.SpawnerLambdaMaxConc, "spawner-lambda-max-conc", 100, "Max concurrent Lambda invocations")

	flag.Parse()

	if cfg.AwsAccessKey == "" {
		cfg.AwsAccessKey = os.Getenv("MY_AWS_ACCESS_KEY")
	}

	if cfg.AwsSecretKey == "" {
		cfg.AwsSecretKey = os.Getenv("MY_AWS_SECRET_KEY")
	}

	if cfg.AwsRegion == "" {
		cfg.AwsRegion = os.Getenv("MY_AWS_REGION")
	}

	// Email validator environment variable fallback
	if cfg.EmailValidatorKey == "" {
		cfg.EmailValidatorKey = os.Getenv("MORIBOUNCER_API_KEY")
	}

	if cfg.AwsLambdaInvoker && cfg.FunctionName == "" {
		panic("FunctionName must be provided when using AwsLambdaInvoker")
	}

	if cfg.AwsLambdaInvoker && cfg.S3Bucket == "" {
		panic("S3Bucket must be provided when using AwsLambdaInvoker")
	}

	if cfg.AwsLambdaInvoker && cfg.InputFile == "" {
		panic("InputFile must be provided when using AwsLambdaInvoker")
	}

	if cfg.Concurrency < 1 {
		panic("Concurrency must be greater than 0")
	}

	if cfg.MaxDepth < 1 {
		panic("MaxDepth must be greater than 0")
	}

	if cfg.Zoom < 0 || cfg.Zoom > 21 {
		panic("Zoom must be between 0 and 21")
	}

	if cfg.Dsn == "" && cfg.ProduceOnly {
		panic("Dsn must be provided when using ProduceOnly")
	}

	if proxies != "" {
		cfg.Proxies = strings.Split(proxies, ",")
	}

	if proxyGateSources != "" {
		cfg.ProxyGateSources = strings.Split(proxyGateSources, ",")
	}

	if cfg.AwsAccessKey != "" && cfg.AwsSecretKey != "" && cfg.AwsRegion != "" {
		cfg.S3Uploader = s3uploader.New(cfg.AwsAccessKey, cfg.AwsSecretKey, cfg.AwsRegion)
	}

	switch {
	case cfg.ManagerMode:
		cfg.RunMode = RunModeManager
	case cfg.WorkerMode:
		cfg.RunMode = RunModeWorker
	case cfg.AwsLambdaInvoker:
		cfg.RunMode = RunModeAwsLambdaInvoker
	case cfg.AwsLamdbaRunner:
		cfg.RunMode = RunModeAwsLambda
	case cfg.Dsn == "":
		cfg.RunMode = RunModeFile
	case cfg.ProduceOnly:
		cfg.RunMode = RunModeDatabaseProduce
	case cfg.Dsn != "":
		cfg.RunMode = RunModeDatabase
	default:
		panic("Invalid configuration")
	}

	return &cfg
}

var (
	telemetryOnce sync.Once
	telemetry     tlmt.Telemetry
)

func Telemetry() tlmt.Telemetry {
	telemetryOnce.Do(func() {
		disableTel := func() bool {
			return os.Getenv("DISABLE_TELEMETRY") == "1"
		}()

		if disableTel {
			telemetry = gonoop.New()

			return
		}

		val, err := goposthog.New("phc_CHYBGEd1eJZzDE7ZWhyiSFuXa9KMLRnaYN47aoIAY2A", "https://eu.i.posthog.com")
		if err != nil || val == nil {
			telemetry = gonoop.New()

			return
		}

		telemetry = val
	})

	return telemetry
}

func wrapText(text string, width int) []string {
	var lines []string

	currentLine := ""
	currentWidth := 0

	for _, r := range text {
		runeWidth := runewidth.RuneWidth(r)
		if currentWidth+runeWidth > width {
			lines = append(lines, currentLine)
			currentLine = string(r)
			currentWidth = runeWidth
		} else {
			currentLine += string(r)
			currentWidth += runeWidth
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func banner(messages []string, width int) string {
	if width <= 0 {
		var err error

		width, _, err = term.GetSize(0)
		if err != nil {
			width = 80
		}
	}

	if width < 20 {
		width = 20
	}

	contentWidth := width - 4

	var wrappedLines []string
	for _, message := range messages {
		wrappedLines = append(wrappedLines, wrapText(message, contentWidth)...)
	}

	var builder strings.Builder

	builder.WriteString("╔" + strings.Repeat("═", width-2) + "╗\n")

	for _, line := range wrappedLines {
		lineWidth := runewidth.StringWidth(line)
		paddingRight := contentWidth - lineWidth

		if paddingRight < 0 {
			paddingRight = 0
		}

		builder.WriteString(fmt.Sprintf("║ %s%s ║\n", line, strings.Repeat(" ", paddingRight)))
	}

	builder.WriteString("╚" + strings.Repeat("═", width-2) + "╝\n")

	return builder.String()
}

func Banner() {
	message1 := "🕷️  Scrapy Kremlit - Google Maps Scraper"
	message2 := "🚀 Powered by Kremlit Dev Team"
	message3 := fmt.Sprintf("v%s (%s)", Version, BuildDate)

	fmt.Fprintln(os.Stderr, banner([]string{message1, message2, message3}, 0))
}
