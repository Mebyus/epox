package logic

import (
	"context"

	"github.com/mebyus/epox/internal/gateways/db"
	"go.uber.org/zap"
)

type Logic struct {
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

	return nil
}
