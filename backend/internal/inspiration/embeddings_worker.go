package inspiration

import (
	"context"
	"log/slog"
	"time"

	"github.com/sidnevart/proof-forge/backend/internal/ai"
)

type EmbeddableRepository interface {
	GoalsNeedingEmbedding(ctx context.Context, limit int) ([]GoalEmbedRow, error)
	SetGoalEmbedding(ctx context.Context, goalID int64, vec []float32) error
	CheckInsNeedingEmbedding(ctx context.Context, limit int) ([]CheckInEmbedRow, error)
	SetCheckInEmbedding(ctx context.Context, checkInID int64, vec []float32) error
}

type EmbeddingsWorker struct {
	repo     EmbeddableRepository
	embedder ai.EmbedProvider
	log      *slog.Logger
	interval time.Duration
}

func NewEmbeddingsWorker(repo EmbeddableRepository, embedder ai.EmbedProvider, log *slog.Logger) *EmbeddingsWorker {
	return &EmbeddingsWorker{
		repo:     repo,
		embedder: embedder,
		log:      log,
		interval: 30 * time.Second,
	}
}

func (w *EmbeddingsWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *EmbeddingsWorker) tick(ctx context.Context) {
	w.embedGoals(ctx)
	w.embedCheckIns(ctx)
}

func (w *EmbeddingsWorker) embedGoals(ctx context.Context) {
	rows, err := w.repo.GoalsNeedingEmbedding(ctx, 20)
	if err != nil {
		w.log.Error("embeddings: list goals", "err", err)
		return
	}
	for _, g := range rows {
		text := g.Title + " " + g.Smart
		vec, err := w.embedder.Embed(ctx, text)
		if err != nil {
			w.log.Error("embeddings: embed goal", "id", g.ID, "err", err)
			continue
		}
		if err := w.repo.SetGoalEmbedding(ctx, g.ID, vec); err != nil {
			w.log.Error("embeddings: save goal embedding", "id", g.ID, "err", err)
		}
	}
}

func (w *EmbeddingsWorker) embedCheckIns(ctx context.Context) {
	rows, err := w.repo.CheckInsNeedingEmbedding(ctx, 20)
	if err != nil {
		w.log.Error("embeddings: list checkins", "err", err)
		return
	}
	for _, ci := range rows {
		text := ci.GoalTitle + " " + ci.TextContent
		vec, err := w.embedder.Embed(ctx, text)
		if err != nil {
			w.log.Error("embeddings: embed checkin", "id", ci.ID, "err", err)
			continue
		}
		if err := w.repo.SetCheckInEmbedding(ctx, ci.ID, vec); err != nil {
			w.log.Error("embeddings: save checkin embedding", "id", ci.ID, "err", err)
		}
	}
}
