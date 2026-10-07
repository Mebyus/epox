package logic

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// AddRepTask adds a repeatable task for the user.
// Sets id upon success.
func (g *Logic) AddRepTask(ctx context.Context, lg *zap.Logger, task *base.RepTask) error {
	now := time.Now()
	if task.NextTrigger == 0 {
		task.NextTrigger = base.FromTime(task.Schedule.Next(now))
	}

	nowts := base.FromTime(now)
	if nowts >= task.NextTrigger {
		return errors.New("next trigger already passed")
	}

	err := g.tags.valid(task.Tags)
	if err != nil {
		return err
	}

	task.CreateTime = nowts

	return g.db.AddRepTask(ctx, lg, task)
}
