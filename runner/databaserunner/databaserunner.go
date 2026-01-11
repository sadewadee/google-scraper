package databaserunner

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	// postgres driver
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sadewadee/google-scraper/postgres"
	"github.com/sadewadee/google-scraper/runner"
	"github.com/sadewadee/google-scraper/tlmt"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/scrapemateapp"
)

type dbrunner struct {
	cfg      *runner.Config
	provider scrapemate.JobProvider
	produce  bool
	app      *scrapemateapp.ScrapemateApp
	conn     *sql.DB
}

func New(cfg *runner.Config) (runner.Runner, error) {
	if cfg.RunMode != runner.RunModeDatabase && cfg.RunMode != runner.RunModeDatabaseProduce {
		return nil, fmt.Errorf("%w: %d", runner.ErrInvalidRunMode, cfg.RunMode)
	}

	conn, err := openPsqlConn(cfg.Dsn)
	if err != nil {
		return nil, err
	}

	ans := dbrunner{
		cfg:      cfg,
		provider: postgres.NewProvider(conn),
		produce:  cfg.ProduceOnly,
		conn:     conn,
	}

	if ans.produce {
		return &ans, nil
	}

	psqlWriter := postgres.NewResultWriter(conn)

	writers := []scrapemate.ResultWriter{
		psqlWriter,
	}

	opts := []func(*scrapemateapp.Config) error{
		// scrapemateapp.WithCache("leveldb", "cache"),
		scrapemateapp.WithConcurrency(cfg.Concurrency),
		scrapemateapp.WithProvider(ans.provider),
		scrapemateapp.WithExitOnInactivity(cfg.ExitOnInactivityDuration),
	}

	if len(cfg.Proxies) > 0 {
		opts = append(opts,
			scrapemateapp.WithProxies(cfg.Proxies),
		)
	}

	if !cfg.FastMode {
		if cfg.Debug {
			opts = append(opts, scrapemateapp.WithJS(
				scrapemateapp.Headfull(),
				scrapemateapp.DisableImages(),
			))
		} else {
			opts = append(opts, scrapemateapp.WithJS(scrapemateapp.DisableImages()))
		}
	} else {
		opts = append(opts, scrapemateapp.WithStealth("firefox"))
	}

	if !cfg.DisablePageReuse {
		opts = append(opts,
			scrapemateapp.WithPageReuseLimit(2),
			scrapemateapp.WithPageReuseLimit(200),
		)
	}

	matecfg, err := scrapemateapp.NewConfig(
		writers,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	ans.app, err = scrapemateapp.NewScrapeMateApp(matecfg)
	if err != nil {
		return nil, err
	}

	return &ans, nil
}

func (d *dbrunner) Run(ctx context.Context) error {
	_ = runner.Telemetry().Send(ctx, tlmt.NewEvent("databaserunner.Run", nil))

	if d.produce {
		return d.produceSeedJobs(ctx)
	}

	return d.app.Start(ctx)
}

func (d *dbrunner) Close(context.Context) error {
	if d.app != nil {
		return d.app.Close()
	}

	if d.conn != nil {
		return d.conn.Close()
	}

	return nil
}

func (d *dbrunner) produceSeedJobs(ctx context.Context) error {
	var input io.Reader

	switch d.cfg.InputFile {
	case "stdin":
		input = os.Stdin
	default:
		f, err := os.Open(d.cfg.InputFile)
		if err != nil {
			return err
		}

		defer f.Close()

		input = f
	}

	jobs, err := runner.CreateSeedJobs(
		d.cfg.FastMode,
		d.cfg.LangCode,
		input,
		d.cfg.MaxDepth,
		d.cfg.Email,
		d.cfg.GeoCoordinates,
		d.cfg.Zoom,
		d.cfg.Radius,
		nil,
		nil,
		d.cfg.ExtraReviews,
	)
	if err != nil {
		return err
	}

	for i := range jobs {
		if err := d.provider.Push(ctx, jobs[i]); err != nil {
			return err
		}
	}

	_ = runner.Telemetry().Send(ctx, tlmt.NewEvent("databaserunner.produceSeedJobs", map[string]any{
		"job_count": len(jobs),
	}))

	return nil
}

func openPsqlConn(dsn string) (conn *sql.DB, err error) {
	// Sanitize DSN to handle special characters in password
	sanitizedDSN, err := sanitizeDSN(dsn)
	if err != nil {
		// If sanitization fails, try with original DSN
		sanitizedDSN = dsn
	}

	conn, err = sql.Open("pgx", sanitizedDSN)
	if err != nil {
		return
	}

	err = conn.Ping()
	if err != nil {
		return
	}

	conn.SetMaxOpenConns(10)

	return
}

// sanitizeDSN converts URL format DSN to key-value format to handle special characters in password
func sanitizeDSN(dsn string) (string, error) {
	// Check if it's a URL format (postgres:// or postgresql://)
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		// Assume it's already in key-value format, return as-is
		return dsn, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	// Convert to key-value format which handles special characters better
	var parts []string

	// Host and port
	host := u.Hostname()
	port := u.Port()
	if host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", host))
	}
	if port != "" {
		parts = append(parts, fmt.Sprintf("port=%s", port))
	}

	// Database name (from path, remove leading slash)
	dbname := strings.TrimPrefix(u.Path, "/")
	if dbname != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", dbname))
	}

	// User and password
	if u.User != nil {
		username := u.User.Username()
		if username != "" {
			parts = append(parts, fmt.Sprintf("user=%s", username))
		}
		password, hasPassword := u.User.Password()
		if hasPassword {
			// Escape single quotes in password by doubling them
			password = strings.ReplaceAll(password, "'", "''")
			parts = append(parts, fmt.Sprintf("password='%s'", password))
		}
	}

	// Query parameters (like sslmode)
	for key, values := range u.Query() {
		if len(values) > 0 {
			parts = append(parts, fmt.Sprintf("%s=%s", key, values[0]))
		}
	}

	return strings.Join(parts, " "), nil
}
