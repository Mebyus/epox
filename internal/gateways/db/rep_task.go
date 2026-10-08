package db

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// GetRepTasksQueueEntries loads active repeatable tasks for all users.
// Used for initializing repeatable tasks queue on server start.
func (c *Client) GetRepTasksQueueEntries(ctx context.Context, lg *zap.Logger) ([]base.RepTaskQueueEntry, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, `
	SELECT
		  id
		, next_trigger_ts
	FROM
		public.rep_tasks AS t
	WHERE
		state = 0
		AND next_trigger_ts IS NOT NULL
	ORDER BY
		next_trigger_ts ASC
	;
	`)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var tasks []base.RepTaskQueueEntry
	for rows.Next() {
		var ent base.RepTaskQueueEntry

		err := rows.Scan(
			&ent.ID,
			&ent.NextTrigger,
		)
		if err != nil {
			lg.Error("scan reptask", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}

		tasks = append(tasks, ent)
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next reptask", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return tasks, nil
}

func (c *Client) AddRepTask(ctx context.Context, lg *zap.Logger, task *base.RepTask) error {
	lg = lg.Named("db")

	data, next, err := task.Schedule.Marshal()
	if err != nil {
		return err
	}

	tzstr := task.Timezone.String()
	if tzstr == "" || tzstr == "Local" {
		name, offset := next.Zone()
		if name != "" {
			tzstr = name
		} else {
			minutes := offset / 60
			h := minutes / 60
			m := minutes - 60*h
			// TODO: handle negative offset
			var sign string
			if offset >= 0 {
				sign = "+"
			} else {
				sign = "-"
			}

			// outputs offset in +HH:MM (or -HH:MM) format
			tzstr = fmt.Sprintf("%s%02d:%02d", sign, h, m)
		}
	}

	var desc sql.NullString
	if task.Description != "" {
		desc = sql.NullString{
			Valid:  true,
			String: task.Description,
		}
	}

	var maxProgress sql.NullInt32
	if task.MaxProgress != 0 {
		maxProgress = sql.NullInt32{
			Valid: true,
			Int32: int32(task.MaxProgress),
		}
	}

	typ := uint32(task.Schedule.Type())
	nextts := task.NextTrigger.Raw()

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()
	row := c.db.QueryRowContext(ctx, `
	INSERT INTO public.rep_tasks (
		  user_id
		, state
		, title
		, description
		, importance
		, next_trigger_ts
		, timezone
		, schedule
		, create_ts
		, max_progress
		, type
		, time_limit
	)
	VALUES
		($1, 0, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING
		id
	;
	`,
		task.UserID,                   // $1  user_id
		task.Title,                    // $2  title
		desc,                          // $3  description
		task.Importance,               // $4  importance
		nextts,                        // $5  next_trigger_ts
		tzstr,                         // $6  timezone
		data,                          // $7  schedule
		task.CreateTime.Raw(),         // $8  create_ts
		maxProgress,                   // $9  max_progress
		typ,                           // $10 type
		task.TimeLimit.Microseconds(), // $11 time_limit
	)

	var id int64
	err = row.Scan(&id)
	if err != nil {
		lg.Error("insert", zap.Error(err))
		return base.ErrDatabaseQuery
	}
	task.ID = base.RepTaskID(id)

	// TODO: attach tags

	return nil
}
