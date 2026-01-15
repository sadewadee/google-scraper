package cache

import (
	"context"
	"sync"
	"time"
)

// NoOpCache is a cache implementation that does nothing (for when cache is disabled)
type NoOpCache struct{}

// NewNoOpCache creates a new no-op cache
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

func (c *NoOpCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrCacheMiss
}

func (c *NoOpCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (c *NoOpCache) Delete(_ context.Context, _ string) error {
	return nil
}

func (c *NoOpCache) DeleteByPattern(_ context.Context, _ string) error {
	return nil
}

func (c *NoOpCache) Close() error {
	return nil
}

// MemoryCache is an in-memory cache implementation for testing/development
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]cacheItem
	stopChan chan struct{}
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		items:    make(map[string]cacheItem),
		stopChan: make(chan struct{}),
	}
	// Start cleanup goroutine
	go c.cleanup()
	return c
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	if time.Now().After(item.expiresAt) {
		return nil, ErrCacheMiss
	}

	return item.value, nil
}

func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

func (c *MemoryCache) DeleteByPattern(_ context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple pattern matching - just check prefix for now
	// For more complex patterns, use regexp
	for key := range c.items {
		if matchSimplePattern(pattern, key) {
			delete(c.items, key)
		}
	}
	return nil
}

func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Signal cleanup goroutine to stop immediately
	close(c.stopChan)
	c.items = nil
	return nil
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return // Cache closed, exit immediately
		case <-ticker.C:
			c.mu.Lock()
			if c.items == nil {
				c.mu.Unlock()
				return // Cache closed
			}

			now := time.Now()
			for key, item := range c.items {
				if now.After(item.expiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		}
	}
}

// matchSimplePattern matches Redis-style patterns with * wildcards
func matchSimplePattern(pattern, key string) bool {
	// Simple implementation: just check if pattern without * is prefix of key
	// For production, use more sophisticated pattern matching
	if len(pattern) == 0 {
		return len(key) == 0
	}

	// Handle trailing wildcard
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return pattern == key
}
