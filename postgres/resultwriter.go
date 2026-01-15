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

type bufferItem struct {
	Entry    *gmaps.Entry
	ParentID string
}

func (r *resultWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	const maxBatchSize = 50

	buff := make([]bufferItem, 0, maxBatchSize)
	lastSave := time.Now().UTC()

	for result := range in {
		entry, ok := result.Data.(*gmaps.Entry)

		if !ok {
			return errors.New("invalid data type")
		}

		buff = append(buff, bufferItem{
			Entry:    entry,
			ParentID: result.Job.GetParentID(),
		})

		if len(buff) >= maxBatchSize || time.Now().UTC().Sub(lastSave) >= time.Minute {
			err := r.batchSave(ctx, buff)
			if err != nil {
				return err
			}

			buff = buff[:0]
			lastSave = time.Now().UTC()
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

func (r *resultWriter) batchSave(ctx context.Context, items []bufferItem) error {
	if len(items) == 0 {
		return nil
	}

	// 1. Insert into results table
	// We use job_id if ParentID is a valid UUID, otherwise NULL
	q := `INSERT INTO results
		(data, job_id)
		VALUES
		`
	elements := make([]string, 0, len(items))
	args := make([]interface{}, 0, len(items)*2)

	// Map to track result counts per job for progress update
	resultsPerJob := make(map[string]int)

	for i, item := range items {
		data, err := json.Marshal(item.Entry)
		if err != nil {
			return err
		}

		// Handle job_id
		var jobID interface{} = nil
		if item.ParentID != "" {
			jobID = item.ParentID
			resultsPerJob[item.ParentID]++
		}

		// ($1, $2), ($3, $4), ...
		elements = append(elements, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, data, jobID)
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
		// If insert fails (e.g. invalid UUID for job_id), try inserting without job_id
		log.Printf("[ResultWriter] WARNING: Insert failed (likely invalid job_id), retrying without job_id link: %v", err)

		// Fallback: Insert without job_id
		qFallback := `INSERT INTO results (data) VALUES `
		elementsFallback := make([]string, 0, len(items))
		argsFallback := make([]interface{}, 0, len(items))

		for i, item := range items {
			data, _ := json.Marshal(item.Entry)
			elementsFallback = append(elementsFallback, fmt.Sprintf("($%d)", i+1))
			argsFallback = append(argsFallback, data)
		}

		qFallback += strings.Join(elementsFallback, ", ")
		qFallback += " ON CONFLICT DO NOTHING"

		if _, errFallback := tx.ExecContext(ctx, qFallback, argsFallback...); errFallback != nil {
			return fmt.Errorf("fallback insert failed: %w", errFallback)
		}
	}

	// 2. Update progress (scraped_places)
	if r.syncParentProgress {
		for jobID, count := range resultsPerJob {
			if count > 0 {
				_, err := tx.ExecContext(ctx, `
					UPDATE jobs_queue
					SET scraped_places = scraped_places + $2,
						updated_at = NOW()
					WHERE id = $1::uuid
				`, jobID, count)
				if err != nil {
					log.Printf("[ResultWriter] WARNING: failed to update scraped_places for job %s: %v", jobID, err)
					// Don't fail the transaction, just log
				}
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	// 3. Sync parent task status (async, outside transaction)
	if r.syncParentProgress {
		r.syncAllParentProgress(ctx)
	}

	return nil
}

// syncAllParentProgress updates status and completed_tasks for parent jobs.
// NOTE: It does NOT update scraped_places anymore, as that is handled incrementally in batchSave.
func (r *resultWriter) syncAllParentProgress(ctx context.Context) {
	q := `
	UPDATE jobs_queue
	SET
		completed_tasks = sub.completed,
		-- scraped_places update REMOVED to avoid double counting
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
