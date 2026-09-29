package db

import (
	"context"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

func (c *Client) GetTags(ctx context.Context, lg *zap.Logger) ([]base.Tag, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, `
	SELECT
		  id
		, name
	FROM
		public.tags
	;`)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var tags []base.Tag
	for rows.Next() {
		var tag base.Tag

		err := rows.Scan(
			&tag.ID,
			&tag.Name,
		)
		if err != nil {
			lg.Error("scan tag", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}

		tags = append(tags, tag)
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next tag", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return tags, nil
}

func (c *Client) SaveTags(ctx context.Context, lg *zap.Logger, list []string) ([]base.Tag, error) {
	if len(list) == 0 {
		return nil, nil
	}
	lg = lg.Named("db")

	query, args := buildInsertTags(list)
	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	tags := make([]base.Tag, 0, len(list))
	for rows.Next() {
		var tag base.Tag

		err := rows.Scan(
			&tag.ID,
			&tag.Name,
		)
		if err != nil {
			lg.Error("scan tag", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}

		tags = append(tags, tag)
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next tag", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	lg.Info("saved tags", zap.Strings("tags", list))
	return tags, nil
}

func buildInsertTags(list []string) (string, []any) {
	args := make([]any, 0, len(list))
	for _, name := range list {
		args = append(args, name)
	}

	var g strings.Builder
	g.Grow(128)
	g.WriteString(`
	INSERT INTO public.tags AS t (name)
	VALUES `)

	g.WriteString("($1)")
	for i := range len(list) - 1 {
		g.WriteString(", ($")
		g.WriteString(strconv.FormatInt(int64(i)+2, 10))
		g.WriteByte(')')
	}

	g.WriteString(`
	ON CONFLICT (name) DO UPDATE SET
		id = t.id
	RETURNING
		  id
		, name
	;`)

	return g.String(), args
}

func buildAttachTaskTags(taskID base.TaskID, tags []base.Tag) (string, []any) {
	var g builder
	g.init(2*len(tags), 128+len(tags)*16)

	g.puts(`
	INSERT INTO public.task_tags (task_id, tag_id)
	VALUES
	`)

	addTaskTag(&g, taskID, tags[0].ID)
	for _, t := range tags[1:] {
		g.puts(",\n")
		addTaskTag(&g, taskID, t.ID)
	}

	g.puts(`
	ON CONFLICT (task_id, tag_id) DO NOTHING
	;`)

	return g.take()
}

func addTaskTag(g *builder, taskID base.TaskID, tagID base.TagID) {
	g.putb('(')
	g.add(int64(taskID))
	g.puts(", ")
	g.add(int64(tagID))
	g.putb(')')
}

// save attached relationships between task and multiple (one or more) tags.
func (c *Client) attachTaskTags(ctx context.Context, lg *zap.Logger, taskID base.TaskID, tags []base.Tag) error {
	query, args := buildAttachTaskTags(taskID, tags)
	_, err := c.db.ExecContext(ctx, query, args...)
	return err
}
