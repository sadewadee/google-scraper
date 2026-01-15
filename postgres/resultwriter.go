package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gosom/scrapemate"

	"github.com/sadewadee/google-scraper/gmaps"
)

// ResultWriterConfig configures the result writer
type ResultWriterConfig struct {
	DB                 *sql.DB
	SyncParentProgress bool // Enable syncing progress back to jobs_queue
}

func NewResultWriter(db *sql.DB) scrapemate.ResultWriter {
	return &resultWriter{db: db}
}

// NewResultWriterWithSync creates a result writer that syncs progress back to parent jobs
func NewResultWriterWithSync(cfg ResultWriterConfig) scrapemate.ResultWriter {
	return &resultWriter{
		db:                 cfg.DB,
		syncParentProgress: cfg.SyncParentProgress,
	}
}

type resultWriter struct {
	db                 *sql.DB
	syncParentProgress bool
}

func (r *resultWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	const maxBatchSize = 50

	buff := make([]*gmaps.Entry, 0, 50)
	lastSave := time.Now().UTC()

	for result := range in {
		entry, ok := result.Data.(*gmaps.Entry)

		if !ok {
			return errors.New("invalid data type")
		}

		buff = append(buff, entry)

		if len(buff) >= maxBatchSize || time.Now().UTC().Sub(lastSave) >= time.Minute {
			err := r.batchSave(ctx, buff)
			if err != nil {
				return err
			}

			buff = buff[:0]
		}
	}

	if len(buff) > 0 {
		err := r.batchSave(ctx, buff)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *resultWriter) batchSave(ctx context.Context, entries []*gmaps.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	q := `INSERT INTO results
		(data)
		VALUES
		`
	elements := make([]string, 0, len(entries))
	args := make([]interface{}, 0, len(entries))

	for i, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		elements = append(elements, fmt.Sprintf("($%d)", i+1))
		args = append(args, data)
	}

	q += strings.Join(elements, ", ")
	q += " ON CONFLICT DO NOTHING"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	// Sync parent job progress if enabled
	if r.syncParentProgress {
		r.syncAllParentProgress(ctx)
	}

	return nil
}

// syncAllParentProgress updates progress for all parent jobs that have pending gmaps_jobs
func (r *resultWriter) syncAllParentProgress(ctx context.Context) {
	q := `
	UPDATE jobs_queue
	SET
		completed_tasks = sub.completed,
		scraped_places = scraped_places + sub.completed,
		status = CASE
			WHEN sub.completed >= total_tasks THEN 'completed'
			ELSE status
		END,
		completed_at = CASE
			WHEN sub.completed >= total_tasks THEN NOW()
			ELSE completed_at
		END
	FROM (
		SELECT
			parent_job_id,
			COUNT(*) FILTER (WHERE status = 'queued') as completed
		FROM gmaps_jobs
		WHERE parent_job_id IS NOT NULL
		GROUP BY parent_job_id
	) sub
	WHERE jobs_queue.id = sub.parent_job_id::uuid
	AND jobs_queue.status NOT IN ('completed', 'failed', 'cancelled')
	`

	_, err := r.db.ExecContext(ctx, q)
	if err != nil {
		log.Printf("[ResultWriter] WARNING: failed to sync parent progress: %v", err)
	}
}

// UpdateParentJobProgress updates the progress of a parent job based on gmaps_jobs completion.
// This can be called directly to sync progress for a specific parent job.
func UpdateParentJobProgress(ctx context.Context, db *sql.DB, parentJobID string) error {
	if parentJobID == "" {
		return nil
	}

	q := `
	UPDATE jobs_queue
	SET
		completed_tasks = (
			SELECT COUNT(*) FROM gmaps_jobs
			WHERE parent_job_id = $1 AND status = 'queued'
		),
		status = CASE
			WHEN (SELECT COUNT(*) FROM gmaps_jobs WHERE parent_job_id = $1 AND status != 'queued') = 0
			THEN 'completed'
			ELSE status
		END,
		completed_at = CASE
			WHEN (SELECT COUNT(*) FROM gmaps_jobs WHERE parent_job_id = $1 AND status != 'queued') = 0
			THEN NOW()
			ELSE completed_at
		END
	WHERE id = $1::uuid
	`

	_, err := db.ExecContext(ctx, q, parentJobID)
	return err
}
