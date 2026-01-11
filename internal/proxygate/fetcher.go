package proxygate

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type Fetcher struct {
	sources []string
	pool    *Pool
	client  *http.Client
}

func NewFetcher(sources []string, pool *Pool) *Fetcher {
	return &Fetcher{
		sources: sources,
		pool:    pool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (f *Fetcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Fetch immediately on startup
	if err := f.fetchAll(ctx); err != nil {
		log.Printf("[ProxyGate] Initial fetch failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := f.fetchAll(ctx); err != nil {
				log.Printf("[ProxyGate] Fetch failed: %v", err)
			}
		}
	}
}

func (f *Fetcher) fetchAll(ctx context.Context) error {
	for _, url := range f.sources {
		if err := f.fetchOne(ctx, url); err != nil {
			log.Printf("[ProxyGate] Fetch from %s failed: %v", url, err)
			continue
		}
	}
	return nil
}

func (f *Fetcher) fetchOne(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		select {
		case f.pool.raw <- line:
		case <-ctx.Done():
			return nil
		}
	}

	return scanner.Err()
}
