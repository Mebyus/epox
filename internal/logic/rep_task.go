package logic

import (
	"context"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// AddRepTask adds a repeatable task for the user.
// Sets id upon success.
func (g *Logic) AddRepTask(ctx context.Context, lg *zap.Logger, task *base.RepTask) error {
	return nil
}
