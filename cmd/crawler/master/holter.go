package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/purrice/prawler/internal/heartbeat"
	"github.com/purrice/prawler/internal/repository"
)

type HolterHandler struct {
	ctx  context.Context
	repo repository.CrawlerRepository
}

func NewHolterHandler(ctx context.Context, db *pgxpool.Pool) HolterHandler {
	repo := repository.NewPostgresCrawlerRepository(db)

	return HolterHandler{
		ctx:  ctx,
		repo: repo,
	}
}

func (h HolterHandler) handleOnBeat(node heartbeat.Node) {
	h.repo.UpdateCrawlerStatus(h.ctx, node.UUID, string(node.Status), node.LastSeen)
}

func (h HolterHandler) handleOnTimeout(node heartbeat.Node) {
	h.repo.RemoveCrawler(h.ctx, node.UUID)
}
