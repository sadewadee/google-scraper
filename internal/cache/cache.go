package cache

import (
	"context"
	"time"
)

// Cache interface for caching operations
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// DeleteByPattern removes all values matching a pattern (e.g., "cache:dashboard:*")
	DeleteByPattern(ctx context.Context, pattern string) error

	// Close closes the cache connection
	Close() error
}

// Key prefixes for dashboard caching
const (
	// KeyPrefixDashboardStats is the prefix for dashboard statistics
	KeyPrefixDashboardStats = "cache:dashboard:stats"

	// KeyPrefixDashboardJobs is the prefix for job listings
	KeyPrefixDashboardJobs = "cache:dashboard:jobs"

	// KeyPrefixDashboardResults is the prefix for result listings
	KeyPrefixDashboardResults = "cache:dashboard:results"

	// KeyPrefixDashboardSearch is the prefix for search results
	KeyPrefixDashboardSearch = "cache:dashboard:search"
)

// TTL configurations for different cache types
const (
	// TTLStats is the TTL for dashboard statistics (30 seconds)
	TTLStats = 30 * time.Second

	// TTLJobsList is the TTL for job listings (60 seconds)
	TTLJobsList = 60 * time.Second

	// TTLJobDetail is the TTL for job details (120 seconds)
	TTLJobDetail = 120 * time.Second

	// TTLResults is the TTL for result listings (60 seconds)
	TTLResults = 60 * time.Second

	// TTLSearch is the TTL for search results (30 seconds)
	TTLSearch = 30 * time.Second
)
