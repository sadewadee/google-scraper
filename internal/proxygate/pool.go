package proxygate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// Pool manages a pool of proxies with optional database persistence
type Pool struct {
	mu      sync.RWMutex
	proxies []*domain.Proxy // In-memory cache of healthy proxies
	index   int
	raw     chan string // Channel for raw fetched proxies
	valid   chan string // Channel for validated proxies

	// Database persistence (optional)
	repo domain.ProxyListRepository
}

// NewPool creates a new proxy pool (in-memory only)
func NewPool() *Pool {
	return &Pool{
		proxies: make([]*domain.Proxy, 0),
		raw:     make(chan string, 10000),
		valid:   make(chan string, 1000),
	}
}

// NewPoolWithRepo creates a new proxy pool with database persistence
func NewPoolWithRepo(repo domain.ProxyListRepository) *Pool {
	return &Pool{
		proxies: make([]*domain.Proxy, 0),
		raw:     make(chan string, 10000),
		valid:   make(chan string, 1000),
		repo:    repo,
	}
}

// LoadFromDatabase loads healthy proxies from database into memory
func (p *Pool) LoadFromDatabase(ctx context.Context) error {
	if p.repo == nil {
		return nil // No database configured
	}

	proxies, err := p.repo.ListHealthy(ctx)
	if err != nil {
		return fmt.Errorf("load healthy proxies: %w", err)
	}

	p.mu.Lock()
	p.proxies = proxies
	p.index = 0
	p.mu.Unlock()

	log.Printf("[ProxyGate] Loaded %d healthy proxies from database", len(proxies))
	return nil
}

// GetNext returns the next proxy in round-robin fashion
func (p *Pool) GetNext() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return "", errors.New("no healthy proxies available")
	}

	proxy := p.proxies[p.index]
	p.index = (p.index + 1) % len(p.proxies)

	// Mark as used (async, don't block)
	if p.repo != nil {
		go func(id int64) {
			ctx := context.Background()
			if err := p.repo.MarkUsed(ctx, id); err != nil {
				log.Printf("[ProxyGate] Failed to mark proxy %d as used: %v", id, err)
			}
		}(proxy.ID)
	}

	return fmt.Sprintf("%s:%d", proxy.IP, proxy.Port), nil
}

// GetNextWithID returns the next proxy with its database ID
func (p *Pool) GetNextWithID() (*domain.Proxy, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return nil, errors.New("no healthy proxies available")
	}

	proxy := p.proxies[p.index]
	p.index = (p.index + 1) % len(p.proxies)

	return proxy, nil
}

// AddValidated adds a validated proxy to the pool
func (p *Pool) AddValidated(proxyAddr string) {
	ip, port, err := parseProxyAddress(proxyAddr)
	if err != nil {
		log.Printf("[ProxyGate] Invalid proxy address %s: %v", proxyAddr, err)
		return
	}

	proxy := &domain.Proxy{
		IP:       ip,
		Port:     port,
		Protocol: "socks5",
		Status:   domain.ProxyStatusHealthy,
	}

	p.AddValidatedProxy(proxy)
}

// AddValidatedProxy adds a validated proxy object to the pool
func (p *Pool) AddValidatedProxy(proxy *domain.Proxy) {
	// Save to database first
	if p.repo != nil {
		ctx := context.Background()
		proxy.Status = domain.ProxyStatusHealthy
		if err := p.repo.Upsert(ctx, proxy); err != nil {
			log.Printf("[ProxyGate] Failed to save proxy %s:%d to database: %v", proxy.IP, proxy.Port, err)
		}
	}

	// Add to memory cache
	p.mu.Lock()
	defer p.mu.Unlock()

	// Deduplicate
	for _, existing := range p.proxies {
		if existing.IP == proxy.IP && existing.Port == proxy.Port {
			return
		}
	}

	p.proxies = append(p.proxies, proxy)
}

// AddRawProxy adds a raw (unvalidated) proxy to the database with pending status
func (p *Pool) AddRawProxy(ctx context.Context, proxy *domain.Proxy) error {
	if p.repo == nil {
		return nil
	}

	proxy.Status = domain.ProxyStatusPending
	return p.repo.Upsert(ctx, proxy)
}

// AddRawProxies adds multiple raw proxies to the database
func (p *Pool) AddRawProxies(ctx context.Context, proxies []*domain.Proxy) error {
	if p.repo == nil {
		return nil
	}

	for _, proxy := range proxies {
		proxy.Status = domain.ProxyStatusPending
	}
	return p.repo.UpsertBatch(ctx, proxies)
}

// Remove removes a proxy from the pool
func (p *Pool) Remove(proxyAddr string) {
	ip, port, err := parseProxyAddress(proxyAddr)
	if err != nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for i, existing := range p.proxies {
		if existing.IP == ip && existing.Port == port {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			break
		}
	}
}

// RemoveByID removes a proxy by database ID
func (p *Pool) RemoveByID(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, existing := range p.proxies {
		if existing.ID == id {
			p.proxies = append(p.proxies[:i], p.proxies[i+1:]...)
			break
		}
	}
}

// MarkFailed marks a proxy as failed
func (p *Pool) MarkFailed(proxyAddr string) {
	ip, port, err := parseProxyAddress(proxyAddr)
	if err != nil {
		return
	}

	// Update database
	if p.repo != nil {
		ctx := context.Background()
		proxy, err := p.repo.GetByAddress(ctx, ip, port)
		if err == nil {
			// Increment fail count, mark as dead if > 3 failures
			if err := p.repo.IncrementFailCount(ctx, proxy.ID, 3); err != nil {
				log.Printf("[ProxyGate] Failed to increment fail count: %v", err)
			}

			// If too many failures, remove from memory cache
			if proxy.FailCount >= 2 { // Will be 3 after increment
				p.RemoveByID(proxy.ID)
			}
		}
	}
}

// MarkSuccess marks a proxy as successful
func (p *Pool) MarkSuccess(proxyAddr string) {
	if p.repo == nil {
		return
	}

	ip, port, err := parseProxyAddress(proxyAddr)
	if err != nil {
		return
	}

	ctx := context.Background()
	proxy, err := p.repo.GetByAddress(ctx, ip, port)
	if err == nil {
		if err := p.repo.IncrementSuccessCount(ctx, proxy.ID); err != nil {
			log.Printf("[ProxyGate] Failed to increment success count: %v", err)
		}
	}
}

// Size returns the number of proxies in the memory pool
func (p *Pool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.proxies)
}

// GetStats returns proxy statistics
func (p *Pool) GetStats(ctx context.Context) (*domain.ProxyStats, error) {
	if p.repo == nil {
		// Return memory-only stats
		p.mu.RLock()
		defer p.mu.RUnlock()
		return &domain.ProxyStats{
			Total:   len(p.proxies),
			Healthy: len(p.proxies),
		}, nil
	}

	return p.repo.GetStats(ctx)
}

// CleanupDead removes dead proxies from database
func (p *Pool) CleanupDead(ctx context.Context) (int, error) {
	if p.repo == nil {
		return 0, nil
	}
	return p.repo.DeleteDead(ctx)
}

// parseProxyAddress parses "IP:port" string
func parseProxyAddress(addr string) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid proxy address: %s", addr)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}

	return parts[0], port, nil
}
