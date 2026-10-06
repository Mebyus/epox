package db

import (
	"context"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// GetActiveRepTasks loads active repeatable tasks for all users.
func (c *Client) GetActiveRepTasks(ctx context.Context, lg *zap.Logger) ([]base.RepTask, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, `
	SELECT
		  t.id
		, t.user_id
		, t."type"
		, t.time_limit
		, t.timezone
		, t.schedule
		, t.title
		, t.description
		, t.importance
		, t.max_progress
		, t.create_ts
	FROM
		public.tasks AS t
	WHERE
		AND t.state = 0
	GROUP BY
		t.id
	ORDER BY
		t.next_trigger_ts ASC
	;
	`,
	)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var tasks []base.RepTask
	for rows.Next() {
		var t taskrow

		err := rows.Scan(
			&t.id,
			&t.title,
			&t.desc,
			&t.importance,
			&t.deadline,
			&t.create,
			&t.progress,
			&t.maxprog,
			&t.tagslist,
		)
		if err != nil {
			lg.Error("scan task", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}
		task, err := t.convert()
		if err != nil {
			lg.Error("convert task row", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}

		_ = task
		tasks = append(tasks, base.RepTask{})
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next task", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return tasks, nil
}
