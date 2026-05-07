package analytics

import (
	"context"
	"time"
)

// RetentionWorker manages analytics partition lifecycle.
type RetentionWorker struct {
	recorder Recorder
	now      func() time.Time
}

// NewRetentionWorker builds the worker.
func NewRetentionWorker(recorder Recorder) *RetentionWorker {
	return &RetentionWorker{recorder: recorder, now: time.Now}
}

// Run executes the monthly partition creation and optional drop.
func (w *RetentionWorker) Run(ctx context.Context) error {
	_ = ctx
	// TODO: implement partition creation and dry-run/drop logic.
	// Phase 4 skeleton — the real implementation needs SQL execution
	// against information_schema.tables to identify old partitions.
	return nil
}

// NoopRecorder is already defined in recorder.go.
// Ensure the worker can operate with it during bootstrap.
var _ Recorder = NoopRecorder{}
