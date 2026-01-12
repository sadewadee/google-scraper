package proxygate

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

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

	return egroup.Wait()
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
