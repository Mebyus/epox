package db

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

func (c *Client) AddNotesTopic(ctx context.Context, lg *zap.Logger, topic *base.NotesTopic) error {
	lg = lg.Named("db")

	var desc sql.NullString
	if topic.Description != "" {
		desc = sql.NullString{
			Valid:  true,
			String: topic.Description,
		}
	}

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()
	row := c.db.QueryRowContext(ctx, `
	INSERT INTO notes.topics (
		  user_id
		, "path"
		, title
		, description
		, create_ts
		, "offset"
	)
	VALUES
		($1, $2, $3, $4, $5, 0)
	RETURNING
		id
	;
	`,
		int64(topic.UserID),          // $1 user_id
		topic.Path,                   // $2 path
		topic.Title,                  // $3 title
		desc,                         // $4 description
		topic.CreateTime.UnixMicro(), // $5 create_ts
	)

	var id int64
	err := row.Scan(&id)
	if err != nil {
		lg.Error("insert", zap.Error(err))
		return base.ErrDatabaseQuery
	}
	topic.ID = base.TopicID(id)

	return nil
}

// LookupNotesTopicByPath check existence of topic in user notes by specified path.
//
// Returns its id and current offset if such topic is found.
func (c *Client) LookupNotesTopicByPath(ctx context.Context, lg *zap.Logger,
	userID base.UserID, path string) (base.TopicID, uint64, error) {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()
	row := c.db.QueryRowContext(ctx, `
	SELECT
		id,
		"offset"
	FROM
		notes.topics
	WHERE
		user_id = $1
		AND "path" = $2
	;
	`,
		int64(userID), // $1 user_id
		path,          // $2 path
	)

	var id int64
	var offset int64
	err := row.Scan(
		&id,
		&offset,
	)
	if err != nil {
		lg.Error("select", zap.Error(err))
		return 0, 0, base.ErrDatabaseQuery
	}

	return base.TopicID(id), uint64(offset), nil
}

func (c *Client) GetNotesMessages(ctx context.Context, lg *zap.Logger, req *base.RequestNotes) ([]base.NotesMessage, error) {
	lg = lg.Named("db")

	query, args := buildGetNotesMessages(req)

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		lg.Error("select", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}
	defer rows.Close()

	var messages []base.NotesMessage
	for rows.Next() {
		var id int64
		var offset int64
		var text string
		var createts int64

		err := rows.Scan(
			&id,
			&offset,
			&text,
			&createts,
		)
		if err != nil {
			lg.Error("scan message", zap.Error(err))
			return nil, base.ErrDatabaseQuery
		}

		messages = append(messages, base.NotesMessage{
			CreateTime: time.UnixMicro(createts),
			Text:       text,
			Offset:     uint64(offset),
			ID:         base.MessageID(id),
		})
	}
	err = rows.Err()
	if err != nil {
		lg.Error("prepare next task", zap.Error(err))
		return nil, base.ErrDatabaseQuery
	}

	return messages, nil
}

func buildGetNotesMessages(req *base.RequestNotes) (string, []any) {
	var g builder
	g.init(4, 256)

	g.puts(`
	SELECT
		  id
		, "offset"
		, "text"
		, create_ts
	FROM
		notes.messages
	WHERE
		user_id = `)
	g.add(int64(req.UserID))

	g.add("\n\tAND topic_id = ")
	g.add(int64(req.TopicID))

	if req.Offset != 0 {
		g.puts("\n\tAND \"offset\" < ")
		g.add(req.Offset)
	}

	g.puts(`
	ORDER BY
		"offset" DESC
	LIMIT `)
	g.add(req.Limit)
	g.puts("\n;")

	return g.take()
}
