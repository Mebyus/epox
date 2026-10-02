package logic

import (
	"context"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/gateways/db"
)

type Logic struct {
	auth scache
	tags TagCache

	db *db.Client
}

func New(db *db.Client) *Logic {
	return &Logic{db: db}
}

func (g *Logic) Init(ctx context.Context, lg *zap.Logger) error {
	lg = lg.Named("init")

	err := g.tags.init(ctx, lg, g.db)
	if err != nil {
		return err
	}

	sessions, err := g.db.GetActiveSessions(ctx, lg, scap)
	if err != nil {
		return err
	}
	g.auth.init(sessions)
	if len(sessions) != 0 {
		lg.Debug("loaded active sessions into cache", zap.Int("count", len(sessions)))
	}

	return nil
}
