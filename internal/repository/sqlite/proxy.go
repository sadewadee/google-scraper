package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"github.com/sadewadee/google-scraper/internal/domain"
)

type ProxyRepository struct {
	db *sql.DB
}

func NewProxyRepository(db *sql.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// scanTime helper to scan nullable time or string into time.Time
// SQLite driver might return string for datetime columns
func scanTime(dest *time.Time, val interface{}) error {
	switch v := val.(type) {
	case time.Time:
		*dest = v
		return nil
	case string:
		// Try parsing common SQLite datetime formats
		// The migration uses datetime('now') which is 'YYYY-MM-DD HH:MM:SS'
		t, err := time.Parse("2006-01-02 15:04:05", v)
		if err == nil {
			*dest = t
			return nil
		}
		// Fallback to RFC3339 if needed
		t, err = time.Parse(time.RFC3339, v)
		if err == nil {
			*dest = t
			return nil
		}
		return fmt.Errorf("failed to parse time %q: %w", v, err)
	case []byte:
		return scanTime(dest, string(v))
	case nil:
		return nil
	default:
		return fmt.Errorf("unsupported type for time scanning: %T", val)
	}
}

func (r *ProxyRepository) Create(ctx context.Context, url string) (*domain.ProxySource, error) {
	query := `
		INSERT INTO proxy_sources (url, created_at, updated_at)
		VALUES (?, datetime('now'), datetime('now'))
		RETURNING id, created_at, updated_at
	`

	source := &domain.ProxySource{
		URL: url,
	}

	var createdAt, updatedAt string
	// Scan into strings first, then parse
	err := r.db.QueryRowContext(ctx, query, url).Scan(
		&source.ID, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy source: %w", err)
	}
	
	// Parse times
	if err := scanTime(&source.CreatedAt, createdAt); err != nil {
		return nil, err
	}
	if err := scanTime(&source.UpdatedAt, updatedAt); err != nil {
		return nil, err
	}

	return source, nil
}

func (r *ProxyRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM proxy_sources WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *ProxyRepository) List(ctx context.Context) ([]*domain.ProxySource, error) {
	query := `SELECT id, url, created_at, updated_at FROM proxy_sources ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*domain.ProxySource
	for rows.Next() {
		s := &domain.ProxySource{}
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.URL, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if err := scanTime(&s.CreatedAt, createdAt); err != nil {
			return nil, err
		}
		if err := scanTime(&s.UpdatedAt, updatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *ProxyRepository) GetByID(ctx context.Context, id int64) (*domain.ProxySource, error) {
	query := `SELECT id, url, created_at, updated_at FROM proxy_sources WHERE id = ?`
	s := &domain.ProxySource{}
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.URL, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proxy source not found: %w", err)
		}
		return nil, err
	}
	if err := scanTime(&s.CreatedAt, createdAt); err != nil {
		return nil, err
	}
	if err := scanTime(&s.UpdatedAt, updatedAt); err != nil {
		return nil, err
	}
	return s, nil
}
