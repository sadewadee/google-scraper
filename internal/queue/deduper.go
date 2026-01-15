package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Deduper provides distributed deduplication using Redis
type Deduper struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// DedupeConfig holds deduper configuration
type DedupeConfig struct {
	RedisURL  string
	RedisAddr string
	Password  string
	DB        int
	Prefix    string
	TTL       time.Duration
}

// NewDeduper creates a new Redis-based deduplicator
func NewDeduper(cfg *DedupeConfig) (*Deduper, error) {
	var client *redis.Client

	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
		// Add connection pool settings
		opt.PoolSize = 10
		opt.MinIdleConns = 2
		opt.DialTimeout = 5 * time.Second
		opt.ReadTimeout = 3 * time.Second
		opt.WriteTimeout = 3 * time.Second
		opt.PoolTimeout = 4 * time.Second
		client = redis.NewClient(opt)
	} else if cfg.RedisAddr != "" {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.RedisAddr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     10,
			MinIdleConns: 2,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolTimeout:  4 * time.Second,
		})
	} else {
		return nil, fmt.Errorf("redis URL or address is required")
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "dedup"
	}

	ttl := cfg.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &Deduper{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}, nil
}

// IsDuplicate checks if a place has already been scraped
// Returns true if duplicate (already seen), false if new
func (d *Deduper) IsDuplicate(ctx context.Context, placeID string) (bool, error) {
	key := fmt.Sprintf("%s:place:%s", d.prefix, placeID)

	// SetNX returns true if the key was set (i.e., it didn't exist)
	wasSet, err := d.client.SetNX(ctx, key, 1, d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate: %w", err)
	}

	// If wasSet is true, this is NOT a duplicate (first time seeing it)
	// If wasSet is false, this IS a duplicate (already existed)
	return !wasSet, nil
}

// IsDuplicateURL checks if a URL has already been processed
func (d *Deduper) IsDuplicateURL(ctx context.Context, url string) (bool, error) {
	key := fmt.Sprintf("%s:url:%s", d.prefix, url)

	wasSet, err := d.client.SetNX(ctx, key, 1, d.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate URL: %w", err)
	}

	return !wasSet, nil
}

// MarkAsSeen marks a place as seen without checking
func (d *Deduper) MarkAsSeen(ctx context.Context, placeID string) error {
	key := fmt.Sprintf("%s:place:%s", d.prefix, placeID)
	return d.client.Set(ctx, key, 1, d.ttl).Err()
}

// Clear removes all dedup keys (use with caution)
func (d *Deduper) Clear(ctx context.Context) error {
	pattern := fmt.Sprintf("%s:*", d.prefix)

	iter := d.client.Scan(ctx, 0, pattern, 1000).Iterator()
	for iter.Next(ctx) {
		if err := d.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}

	return iter.Err()
}

// Stats returns deduplication statistics
func (d *Deduper) Stats(ctx context.Context) (int64, error) {
	pattern := fmt.Sprintf("%s:place:*", d.prefix)

	var count int64
	iter := d.client.Scan(ctx, 0, pattern, 1000).Iterator()
	for iter.Next(ctx) {
		count++
	}

	return count, iter.Err()
}

// Close closes the Redis connection
func (d *Deduper) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// Seen implements the deduper.Deduper interface for compatibility
func (d *Deduper) Seen(id string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	isDup, err := d.IsDuplicate(ctx, id)
	if err != nil {
		// On error, assume not duplicate to avoid data loss
		return false
	}

	return isDup
}

// AddIfNotExists implements the deduper.Deduper interface
// Returns true if the key was added (not a duplicate), false if already existed
func (d *Deduper) AddIfNotExists(ctx context.Context, key string) bool {
	isDup, err := d.IsDuplicateURL(ctx, key)
	if err != nil {
		// On error, return true to allow processing (avoid data loss)
		return true
	}

	// IsDuplicateURL returns true if key already exists
	// AddIfNotExists should return true if key was NEW (not duplicate)
	return !isDup
}
