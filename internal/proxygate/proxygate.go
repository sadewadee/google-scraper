package proxygate

import (
	"context"
	"log"
	"time"

	"github.com/sadewadee/google-scraper/internal/domain"
	"golang.org/x/sync/errgroup"
)

// DefaultPoolRefreshInterval is the default interval for refreshing pool from database
const DefaultPoolRefreshInterval = 2 * time.Minute

type ProxyGate struct {
	cfg       *Config
	pool      *Pool
	fetcher   *Fetcher
	validator *Validator
	server    *Server
}

func New(cfg *Config) *ProxyGate {
	pool := NewPool()
	fetcher := NewFetcher(cfg.SourceURLs, pool)
	validator := NewValidator(cfg.ValidatorConcurrency, pool)
	server := NewServer(cfg.ListenAddr, pool)

	return &ProxyGate{
		cfg:       cfg,
		pool:      pool,
		fetcher:   fetcher,
		validator: validator,
		server:    server,
	}
}

func (pg *ProxyGate) Run(ctx context.Context) error {
	egroup, ctx := errgroup.WithContext(ctx)

	egroup.Go(func() error { return pg.fetcher.Run(ctx) })
	egroup.Go(func() error { return pg.validator.Run(ctx) })
	egroup.Go(func() error { return pg.server.Run(ctx) })
	egroup.Go(func() error { return pg.runPoolRefresher(ctx) })

	return egroup.Wait()
}

// runPoolRefresher periodically reloads proxies from database
// This ensures any proxies added directly to DB are picked up
func (pg *ProxyGate) runPoolRefresher(ctx context.Context) error {
	ticker := time.NewTicker(DefaultPoolRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !pg.pool.HasRepo() {
				continue // No database configured
			}
			if err := pg.pool.LoadFromDatabase(ctx); err != nil {
				log.Printf("[ProxyGate] Pool refresh failed: %v", err)
			}
		}
	}
}

func (pg *ProxyGate) Refresh(ctx context.Context) error {
	return pg.fetcher.ForceRefresh(ctx)
}

func (pg *ProxyGate) GetStats() (int, int, time.Time) {
	// total, healthy
	// For now assume all in pool are healthy
	return pg.pool.Size(), pg.pool.Size(), pg.fetcher.LastUpdated()
}

func (pg *ProxyGate) GetSources() []string {
	return pg.fetcher.GetSources()
}

func (pg *ProxyGate) AddSource(url string) {
	pg.fetcher.AddSource(url)
}

func (pg *ProxyGate) RemoveSource(url string) {
	pg.fetcher.RemoveSource(url)
}

// SetPoolRepo sets the database repository for proxy persistence
// This can be called after construction when database becomes available
func (pg *ProxyGate) SetPoolRepo(repo domain.ProxyListRepository) {
	pg.pool.SetRepo(repo)
}

// LoadFromDatabase loads healthy proxies from database into the in-memory pool
// This should be called after SetPoolRepo to initialize the pool with existing proxies
func (pg *ProxyGate) LoadFromDatabase(ctx context.Context) error {
	return pg.pool.LoadFromDatabase(ctx)
}

// AddProxyToPool adds a proxy directly to the in-memory pool
// Use this when adding proxies via API to immediately make them available
func (pg *ProxyGate) AddProxyToPool(proxy *domain.Proxy) {
	pg.pool.AddValidatedProxy(proxy)
}

// ReloadFromDatabase reloads all healthy proxies from database
// This can be called periodically or after bulk updates
func (pg *ProxyGate) ReloadFromDatabase(ctx context.Context) error {
	return pg.pool.LoadFromDatabase(ctx)
}

// PoolSize returns the current number of proxies in memory
func (pg *ProxyGate) PoolSize() int {
	return pg.pool.Size()
}
