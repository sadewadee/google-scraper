package domain

import "github.com/google/uuid"

// ResultBatch represents a batch of results for submission
type ResultBatch struct {
	JobID uuid.UUID `json:"job_id"`
	Data  [][]byte  `json:"data"`
}
