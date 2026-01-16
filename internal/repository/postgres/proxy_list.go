package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sadewadee/google-scraper/internal/domain"
)

// ProxyListRepository implements domain.ProxyListRepository for PostgreSQL
type ProxyListRepository struct {
	db *sql.DB
}

// NewProxyListRepository creates a new ProxyListRepository
func NewProxyListRepository(db *sql.DB) *ProxyListRepository {
	return &ProxyListRepository{db: db}
}

// Upsert creates or updates a proxy (based on IP:port unique constraint)
func (r *ProxyListRepository) Upsert(ctx context.Context, proxy *domain.Proxy) error {
	query := `
		INSERT INTO proxies (ip, port, protocol, country, uptime, response_time, status, source_id, source_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (ip, port) DO UPDATE SET
			protocol = EXCLUDED.protocol,
			country = COALESCE(EXCLUDED.country, proxies.country),
			uptime = COALESCE(EXCLUDED.uptime, proxies.uptime),
			response_time = COALESCE(EXCLUDED.response_time, proxies.response_time),
			source_url = COALESCE(EXCLUDED.source_url, proxies.source_url),
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		proxy.IP,
		proxy.Port,
		proxy.Protocol,
		nullString(proxy.Country),
		nullFloat64(proxy.Uptime),
		nullFloat64(proxy.ResponseTime),
		proxy.Status,
		nullInt64(proxy.SourceID),
		nullString(proxy.SourceURL),
	).Scan(&proxy.ID, &proxy.CreatedAt, &proxy.UpdatedAt)

	return err
}

// UpsertBatch creates or updates multiple proxies
func (r *ProxyListRepository) UpsertBatch(ctx context.Context, proxies []*domain.Proxy) error {
	if len(proxies) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO proxies (ip, port, protocol, country, uptime, response_time, status, source_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (ip, port) DO UPDATE SET
			country = COALESCE(EXCLUDED.country, proxies.country),
			uptime = COALESCE(EXCLUDED.uptime, proxies.uptime),
			response_time = COALESCE(EXCLUDED.response_time, proxies.response_time),
			source_url = COALESCE(EXCLUDED.source_url, proxies.source_url),
			updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, proxy := range proxies {
		_, err := stmt.ExecContext(ctx,
			proxy.IP,
			proxy.Port,
			proxy.Protocol,
			nullString(proxy.Country),
			nullFloat64(proxy.Uptime),
			nullFloat64(proxy.ResponseTime),
			proxy.Status,
			nullString(proxy.SourceURL),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByAddress retrieves a proxy by IP:port
func (r *ProxyListRepository) GetByAddress(ctx context.Context, ip string, port int) (*domain.Proxy, error) {
	query := `
		SELECT id, ip, port, protocol, country, uptime, response_time, status,
		       last_checked, last_used, fail_count, success_count, source_id, source_url,
		       created_at, updated_at
		FROM proxies
		WHERE ip = $1 AND port = $2
	`

	proxy := &domain.Proxy{}
	var country, sourceURL sql.NullString
	var uptime, responseTime sql.NullFloat64
	var lastChecked, lastUsed sql.NullTime
	var sourceID sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, ip, port).Scan(
		&proxy.ID, &proxy.IP, &proxy.Port, &proxy.Protocol,
		&country, &uptime, &responseTime, &proxy.Status,
		&lastChecked, &lastUsed, &proxy.FailCount, &proxy.SuccessCount,
		&sourceID, &sourceURL, &proxy.CreatedAt, &proxy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if country.Valid {
		proxy.Country = country.String
	}
	if uptime.Valid {
		proxy.Uptime = uptime.Float64
	}
	if responseTime.Valid {
		proxy.ResponseTime = responseTime.Float64
	}
	if lastChecked.Valid {
		proxy.LastChecked = &lastChecked.Time
	}
	if lastUsed.Valid {
		proxy.LastUsed = &lastUsed.Time
	}
	if sourceID.Valid {
		proxy.SourceID = &sourceID.Int64
	}
	if sourceURL.Valid {
		proxy.SourceURL = sourceURL.String
	}

	return proxy, nil
}

// List retrieves proxies with optional filtering
func (r *ProxyListRepository) List(ctx context.Context, params domain.ProxyListParams) ([]*domain.Proxy, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, params.Status)
		argNum++
	}
	if params.Protocol != "" {
		conditions = append(conditions, fmt.Sprintf("protocol = $%d", argNum))
		args = append(args, params.Protocol)
		argNum++
	}
	if params.Country != "" {
		conditions = append(conditions, fmt.Sprintf("country = $%d", argNum))
		args = append(args, params.Country)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM proxies %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List query
	query := fmt.Sprintf(`
		SELECT id, ip, port, protocol, country, uptime, response_time, status,
		       last_checked, last_used, fail_count, success_count, source_id, source_url,
		       created_at, updated_at
		FROM proxies
		%s
		ORDER BY uptime DESC NULLS LAST, response_time ASC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	proxies, err := scanProxies(rows)
	if err != nil {
		return nil, 0, err
	}

	return proxies, total, nil
}

// ListHealthy retrieves all healthy proxies (for Pool)
func (r *ProxyListRepository) ListHealthy(ctx context.Context) ([]*domain.Proxy, error) {
	query := `
		SELECT id, ip, port, protocol, country, uptime, response_time, status,
		       last_checked, last_used, fail_count, success_count, source_id, source_url,
		       created_at, updated_at
		FROM proxies
		WHERE status = 'healthy'
		ORDER BY uptime DESC NULLS LAST, response_time ASC NULLS LAST
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProxies(rows)
}

// UpdateStatus updates the status of a proxy
func (r *ProxyListRepository) UpdateStatus(ctx context.Context, id int64, status domain.ProxyStatus) error {
	query := `
		UPDATE proxies
		SET status = $1, last_checked = NOW(), updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// IncrementFailCount increments fail count and optionally marks as dead
func (r *ProxyListRepository) IncrementFailCount(ctx context.Context, id int64, maxFails int) error {
	query := `
		UPDATE proxies
		SET fail_count = fail_count + 1,
		    status = CASE WHEN fail_count + 1 >= $2 THEN 'dead' ELSE status END,
		    last_checked = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, maxFails)
	return err
}

// IncrementSuccessCount increments success count
func (r *ProxyListRepository) IncrementSuccessCount(ctx context.Context, id int64) error {
	query := `
		UPDATE proxies
		SET success_count = success_count + 1,
		    fail_count = 0,
		    status = 'healthy',
		    last_checked = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// MarkUsed updates the last_used timestamp
func (r *ProxyListRepository) MarkUsed(ctx context.Context, id int64) error {
	query := `UPDATE proxies SET last_used = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteDead removes all dead proxies
func (r *ProxyListRepository) DeleteDead(ctx context.Context) (int, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM proxies WHERE status = 'dead'`)
	if err != nil {
		return 0, err
	}
	count, _ := result.RowsAffected()
	return int(count), nil
}

// GetStats retrieves proxy statistics
func (r *ProxyListRepository) GetStats(ctx context.Context) (*domain.ProxyStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'healthy') as healthy,
			COUNT(*) FILTER (WHERE status = 'dead') as dead,
			COUNT(*) FILTER (WHERE status = 'banned') as banned,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COALESCE(AVG(uptime) FILTER (WHERE uptime IS NOT NULL), 0) as avg_uptime
		FROM proxies
	`

	stats := &domain.ProxyStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.Total, &stats.Healthy, &stats.Dead, &stats.Banned, &stats.Pending, &stats.AvgUptime,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Helper function to scan proxies from rows
func scanProxies(rows *sql.Rows) ([]*domain.Proxy, error) {
	var proxies []*domain.Proxy

	for rows.Next() {
		proxy := &domain.Proxy{}
		var country, sourceURL sql.NullString
		var uptime, responseTime sql.NullFloat64
		var lastChecked, lastUsed sql.NullTime
		var sourceID sql.NullInt64

		err := rows.Scan(
			&proxy.ID, &proxy.IP, &proxy.Port, &proxy.Protocol,
			&country, &uptime, &responseTime, &proxy.Status,
			&lastChecked, &lastUsed, &proxy.FailCount, &proxy.SuccessCount,
			&sourceID, &sourceURL, &proxy.CreatedAt, &proxy.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if country.Valid {
			proxy.Country = country.String
		}
		if uptime.Valid {
			proxy.Uptime = uptime.Float64
		}
		if responseTime.Valid {
			proxy.ResponseTime = responseTime.Float64
		}
		if lastChecked.Valid {
			proxy.LastChecked = &lastChecked.Time
		}
		if lastUsed.Valid {
			proxy.LastUsed = &lastUsed.Time
		}
		if sourceID.Valid {
			proxy.SourceID = &sourceID.Int64
		}
		if sourceURL.Valid {
			proxy.SourceURL = sourceURL.String
		}

		proxies = append(proxies, proxy)
	}

	return proxies, rows.Err()
}

// Helper functions for nullable types
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullFloat64(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

func nullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
