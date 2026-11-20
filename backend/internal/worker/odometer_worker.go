package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/motospec/backend/internal/repository"
)

// OdometerWorker runs scheduled background jobs to auto-increment daily odometer readings.
type OdometerWorker struct {
	store  repository.Store
	cron   *cron.Cron
	entryID cron.EntryID
}

// NewOdometerWorker creates a new OdometerWorker.
func NewOdometerWorker(store repository.Store) *OdometerWorker {
	return &OdometerWorker{
		store: store,
		cron:  cron.New(cron.WithLocation(time.Local)),
	}
}

// Start registers the daily cron job (0 0 * * *) and starts the scheduler.
func (w *OdometerWorker) Start() error {
	entryID, err := w.cron.AddFunc("0 0 * * *", w.RunDailyIncrement)
	if err != nil {
		return err
	}
	w.entryID = entryID
	w.cron.Start()
	slog.Info("odometer auto-increment cron worker started", "schedule", "0 0 * * *")
	return nil
}

// RunDailyIncrement executes the idempotent set-based odometer auto-increment query.
func (w *OdometerWorker) RunDailyIncrement() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rowsAffected, err := w.store.BatchAutoIncrementOdometers(ctx)
	if err != nil {
		slog.Error("failed to execute batch odometer auto-increment", "error", err)
		return
	}

	slog.Info("odometer auto-increment completed", "updated_bikes", rowsAffected)
}

// Stop gracefully halts the cron worker.
func (w *OdometerWorker) Stop() {
	if w.cron != nil {
		w.cron.Stop()
		slog.Info("odometer auto-increment cron worker stopped")
	}
}
