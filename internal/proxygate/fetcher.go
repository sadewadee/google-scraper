package proxygate

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Fetcher struct {
	sources     []string
	pool        *Pool
	client      *http.Client
	mu          sync.RWMutex
	lastUpdated time.Time
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
	f.mu.RLock()
	sources := make([]string, len(f.sources))
	copy(sources, f.sources)
	f.mu.RUnlock()

	for _, url := range sources {
		if err := f.fetchOne(ctx, url); err != nil {
			log.Printf("[ProxyGate] Fetch from %s failed: %v", url, err)
			continue
		}
	}

	f.mu.Lock()
	f.lastUpdated = time.Now()
	f.mu.Unlock()

	return nil
}

func (f *Fetcher) ForceRefresh(ctx context.Context) error {
	return f.fetchAll(ctx)
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

func (f *Fetcher) LastUpdated() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastUpdated
}

func (f *Fetcher) AddSource(url string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sources {
		if s == url {
			return
		}
	}
	f.sources = append(f.sources, url)
}

func (f *Fetcher) RemoveSource(url string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, s := range f.sources {
		if s == url {
			f.sources = append(f.sources[:i], f.sources[i+1:]...)
			return
		}
	}
}

func (f *Fetcher) GetSources() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	sources := make([]string, len(f.sources))
	copy(sources, f.sources)
	return sources
}
