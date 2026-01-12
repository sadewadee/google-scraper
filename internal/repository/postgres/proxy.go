package postgres

import (
	"context"
	"database/sql"

	"github.com/sadewadee/google-scraper/internal/domain"
)

type ProxyRepository struct {
	db *sql.DB
}

func NewProxyRepository(db *sql.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

func (r *ProxyRepository) Create(ctx context.Context, url string) (*domain.ProxySource, error) {
	query := `
		INSERT INTO proxy_sources (url, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	source := &domain.ProxySource{
		URL: url,
	}

	err := r.db.QueryRowContext(ctx, query, url).Scan(
		&source.ID, &source.CreatedAt, &source.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return source, nil
}

func (r *ProxyRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM proxy_sources WHERE id = $1`
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
		if err := rows.Scan(&s.ID, &s.URL, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

func (r *ProxyRepository) GetByID(ctx context.Context, id int64) (*domain.ProxySource, error) {
	query := `SELECT id, url, created_at, updated_at FROM proxy_sources WHERE id = $1`
	s := &domain.ProxySource{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.URL, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
