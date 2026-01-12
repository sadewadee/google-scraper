package worker

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gosom/scrapemate"
)

// MemoryWriter is a ResultWriter that stores results in memory
type MemoryWriter struct {
	mu      sync.Mutex
	Results [][]byte
}

// Run implements scrapemate.ResultWriter
func (w *MemoryWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		w.mu.Lock()
		data, err := json.Marshal(result.Data)
		if err != nil {
			w.mu.Unlock()
			return err
		}
		w.Results = append(w.Results, data)
		w.mu.Unlock()
	}
	return nil
}

// GetResults returns a copy of the stored results
func (w *MemoryWriter) GetResults() [][]byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	result := make([][]byte, len(w.Results))
	copy(result, w.Results)
	return result
}
