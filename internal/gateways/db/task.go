package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

type taskrow struct {
	desc       sql.NullString
	start      sql.NullInt64
	deadline   sql.NullInt64
	update     sql.NullInt64
	title      string
	tagslist   string
	importance sql.NullInt32
	progress   sql.NullInt32
	maxprog    sql.NullInt32
	id         uint64
	create     int64
}

// Parse list of tags from list of ids in text format
// separated by spaces. Examples (quotes are for Go strings):
//
//	""     - empty string means empty list
//	"5"    - list with only one element [5]
//	"5 10" - list with two elements [5, 10]
//
// Returns nil slice for empty string input. Only tag ids are
// filled in result, names are left empty.
func parseTagsList(s string) ([]base.Tag, error) {
	if s == "" {
		return nil, nil
	}

	n := 1 // number of elements counter (+1 to number of spaces)
	for i := range len(s) {
		c := s[i]
		if c == ' ' {
			n += 1
		}
	}

	tags := make([]base.Tag, 0, n)
	for {
		str, rest := cutspace(s)

		id, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse tag id from string \"%s\": %v", str, err)
		}
		tags = append(tags, base.Tag{ID: base.TagID(id)})

		if rest == "" {
			return tags, nil
		}
		s = rest
	}
}

func cutspace(s string) (string, string) {
	for i := range len(s) {
		c := s[i]
		if c == ' ' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func (t *taskrow) convert() (base.Task, error) {
	tags, err := parseTagsList(t.tagslist)
	if err != nil {
		return base.Task{}, err
	}

	var deadline base.MicroTime
	if t.deadline.Valid {
		deadline = base.MicroTime(t.deadline.Int64)
	}

	var updateTime time.Time
	if t.update.Valid {
		updateTime = time.UnixMicro(t.update.Int64)
	}

	var start base.MicroTime
	if t.start.Valid {
		start = base.MicroTime(t.start.Int64)
	}

	return base.Task{
		Title:       t.title,
		ID:          base.TaskID(t.id),
		Tags:        tags,
		Description: t.desc.String,
		CreateTime:  time.UnixMicro(t.create),
		UpdateTime:  updateTime,
		StartTime:   start,
		Deadline:    deadline,
		Importance:  uint32(t.importance.Int32),
		Progress:    uint32(t.progress.Int32),
		MaxProgress: uint32(t.maxprog.Int32),
	}, nil
}

func (c *Client) GetActiveTasks(ctx context.Context, lg *zap.Logger, userID base.UserID) ([]base.ActiveTask, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, `
	SELECT
		  t.id
		, t.title
		, t.description
		, t.importance
		, t.start_ts
		, t.deadline_ts
		, t.create_ts
		, t.progress
		, t.max_progress
		, coalesce(string_agg(tt.tag_id::text, ' '), '') AS tags_list
	FROM
		public.tasks AS t
	LEFT JOIN
		public.task_tags AS tt ON tt.task_id = t.id
	WHERE
		t.user_id = $1
		AND t.state = 0
		AND (t.deadline_ts IS NULL OR t.deadline_ts > $2)
	GROUP BY
		t.id
	ORDER BY
		t.create_ts
	;
	`,
		uint64(userID),         // $1 user_id
		time.Now().UnixMicro(), // $2 deadline_ts
	)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var tasks []base.ActiveTask
	for rows.Next() {
		var t taskrow

		err := rows.Scan(
			&t.id,
			&t.title,
			&t.desc,
			&t.importance,
			&t.start,
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
		task.UserID = userID

		tasks = append(tasks, base.ActiveTask{Task: task})
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next task", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return tasks, nil
}

// GetHistoryTasks get historical (non-active) user tasks
// created after specified cutoff.
func (c *Client) GetHistoryTasks(ctx context.Context, lg *zap.Logger, userID base.UserID, cutoff time.Time) ([]base.HistoryTask, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, `
	SELECT
		  id
		, title
		, description
		, importance
		, deadline_ts
		, create_ts
		, update_ts
		, progress
		, max_progress
		, state
	FROM
		public.tasks
	WHERE
		user_id = $1
		AND create_ts >= $2
		AND (
			state != 0
			OR deadline_ts <= $3
		)
	ORDER BY
		create_ts
	;
	`,
		uint64(userID),         // $1 user_id
		cutoff.UnixMicro(),     // $2 create_ts
		time.Now().UnixMicro(), // $3 deadline_ts
	)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var tasks []base.HistoryTask
	for rows.Next() {
		var state uint32
		var t taskrow

		err := rows.Scan(
			&t.id,
			&t.title,
			&t.desc,
			&t.importance,
			&t.deadline,
			&t.create,
			&t.update,
			&t.progress,
			&t.maxprog,
			&state,
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
		task.UserID = userID

		tasks = append(tasks, base.HistoryTask{
			Task:  task,
			State: base.TaskState(state),
		})
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next task", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return tasks, nil

}

// AddTask adds an active task for the user.
// Sets task id upon success.
func (c *Client) AddTask(ctx context.Context, lg *zap.Logger, task *base.ActiveTask) error {
	lg = lg.Named("db")

	var deadline sql.NullInt64
	if task.Deadline != 0 {
		deadline = sql.NullInt64{
			Valid: true,
			Int64: int64(task.Deadline.Raw()),
		}
	}

	var desc sql.NullString
	if task.Description != "" {
		desc = sql.NullString{
			Valid:  true,
			String: task.Description,
		}
	}

	var progress sql.NullInt32
	var maxProgress sql.NullInt32
	if task.MaxProgress != 0 {
		progress = sql.NullInt32{
			Valid: true,
			Int32: int32(task.Progress),
		}
		maxProgress = sql.NullInt32{
			Valid: true,
			Int32: int32(task.MaxProgress),
		}
	}

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()
	row := c.db.QueryRowContext(ctx, `
	INSERT INTO public.tasks (
		  user_id
		, state
		, title
		, description
		, importance
		, deadline_ts
		, create_ts
		, progress
		, max_progress
	)
	VALUES
		($1, 0, $2, $3, $4, $5, $6, $7, $8)
	RETURNING
		id
	;
	`,
		task.UserID,                 // $1 user_id
		task.Title,                  // $2 title
		desc,                        // $3 description
		task.Importance,             // $4 importance
		deadline,                    // $5 deadline_ts
		task.CreateTime.UnixMicro(), // $6 create_ts
		progress,                    // $7 progress
		maxProgress,                 // $8 max_progress
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		lg.Error("insert", zap.Error(err))
		return base.ErrDatabaseQuery
	}
	task.ID = base.TaskID(id)

	if len(task.Tags) != 0 {
		err = c.attachTaskTags(ctx, lg, task.ID, task.Tags)
		if err != nil {
			lg.Error("attach tags", zap.Error(err))
		}
	}

	return nil
}

func (c *Client) ChangeTaskState(ctx context.Context, lg *zap.Logger, req *base.RequestChangeTaskState) error {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	result, err := c.db.ExecContext(ctx, `
	UPDATE public.tasks SET
		state = $1,
		update_ts = $2
	WHERE
		id = $3
		AND user_id = $4
	;
	`,
		int32(req.State),
		time.Now().UnixMicro(),
		int64(req.ID),
		int64(req.UserID),
	)
	if err != nil {
		lg.Error("update", zap.Error(err))
		return base.ErrDatabaseQuery
	}

	count, err := result.RowsAffected()
	if err != nil {
		lg.Error("check result", zap.Error(err))
		return base.ErrDatabaseQuery
	}
	if count <= 0 {
		return base.ErrTaskNotFound
	}

	return nil
}

func (c *Client) MarkTasksExpired(ctx context.Context, lg *zap.Logger, tasks []base.ExpiredTaskChore) error {
	if len(tasks) == 0 {
		return nil
	}

	query, args := buildMarkTasksExpiredQuery(tasks)
	_, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		lg.Error("update", zap.Error(err))
		return base.ErrDatabaseQuery
	}

	return nil
}

func addExpiredTaskChore(g *builder, task base.ExpiredTaskChore) {
	g.putb('(')
	g.add(int64(task.ID))
	g.puts("::int8, ")
	g.add(task.Time.UnixMicro())
	g.puts("::int8)")
}

func buildMarkTasksExpiredQuery(tasks []base.ExpiredTaskChore) (string, []any) {
	var g builder
	g.init(2*len(tasks), 128+len(tasks)*16)

	g.puts(`
	UPDATE public.tasks AS t SET
		state = 3,
		update_ts = v.update_ts
	FROM (VALUES
	`)

	addExpiredTaskChore(&g, tasks[0])
	for _, t := range tasks[1:] {
		g.puts(",\n")
		addExpiredTaskChore(&g, t)
	}
	g.puts(`
	) AS v(id, update_ts)
	WHERE
		t.id = v.id
	;`)

	return g.take()
}
