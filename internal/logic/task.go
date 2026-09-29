package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/mebyus/epox/internal/base"
)

func (g *Logic) GetActiveTasks(ctx context.Context, lg *zap.Logger, userID base.UserID) ([]base.ActiveTask, error) {
	tasks, err := g.db.GetActiveTasks(ctx, lg, userID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}

	for _, t := range tasks {
		if len(t.Tags) != 0 {
			g.tags.fill(t.Tags)
		}
	}

	return tasks, nil
}

// GetHistoryTasks get historical (non-active) user tasks
// created after specified cutoff.
func (g *Logic) GetHistoryTasks(ctx context.Context, lg *zap.Logger, userID base.UserID, cutoff time.Time) ([]base.HistoryTask, error) {
	tasks, err := g.db.GetHistoryTasks(ctx, lg, userID, cutoff)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}

	var chores []base.ExpiredTaskChore
	for i := range len(tasks) {
		t := &tasks[i]
		if t.State != base.TaskActive {
			continue
		}

		if t.Deadline.IsZero() {
			panic(fmt.Sprintf("impossible condition on history task (id=%d)", t.ID))
		}

		t.UpdateTime = t.Deadline
		t.State = base.TaskExpired
		chores = append(chores, base.ExpiredTaskChore{
			ID:   t.ID,
			Time: t.Deadline,
		})
	}

	if len(chores) != 0 {
		if lg.Level() <= zapcore.DebugLevel {
			ids := make([]uint64, 0, len(chores))
			for _, t := range chores {
				ids = append(ids, uint64(t.ID))
			}
			lg.Debug("found unmarked expired tasks", zap.Int("count", len(chores)), zap.Uint64s("ids", ids))
		}
		err = g.db.MarkTasksExpired(ctx, lg, chores)
		if err == nil {
			// note the condition for error
			lg.Debug("marked expired tasks")
		}
	}

	return tasks, nil
}

// AddTask adds an active task for the user.
// Sets task id upon success.
func (g *Logic) AddTask(ctx context.Context, lg *zap.Logger, task *base.ActiveTask) error {
	if task.Progress > task.MaxProgress {
		return errors.New("progress exceeds max progress")
	}

	now := time.Now()
	if !task.Deadline.IsZero() && !task.Deadline.After(now) {
		return errors.New("deadline already passed")
	}

	err := g.tags.valid(task.Tags)
	if err != nil {
		return err
	}

	task.CreateTime = now

	return g.db.AddTask(ctx, lg, task)
}

func (g *Logic) ChangeTaskState(ctx context.Context, lg *zap.Logger, req *base.RequestChangeTaskState) error {
	return g.db.ChangeTaskState(ctx, lg, req)
}
