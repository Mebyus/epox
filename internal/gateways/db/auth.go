package db

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

func (c *Client) LoadAuthData(ctx context.Context, lg *zap.Logger, data *base.AuthData) error {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	row := c.db.QueryRowContext(ctx, `
	SELECT
		id
		, password
	FROM
		public.users
	WHERE
		login = $1
	;
	`,
		data.Login,
	)

	var id int64
	var password string
	err := row.Scan(
		&id,
		&password,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return base.ErrCredentialsMismatch
		}

		lg.Error("query db", zap.Error(err))
		return base.ErrDatabaseQuery
	}

	data.UserID = base.UserID(id)
	data.Password = password
	return nil
}

// LoadSession loads session by its token.
func (c *Client) LoadSession(ctx context.Context, lg *zap.Logger, session *base.Session) error {
	lg = lg.Named("db")

	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	row := c.db.QueryRowContext(ctx, `
	SELECT
		  user_id
		, create_ts
		, expire_ts
	FROM
		public.sessions
	WHERE
		token = $1
		AND expire_ts > $2
	;
	`,
		session.Token,          // $1 token
		time.Now().UnixMicro(), // $2 expire_ts
	)

	var create int64
	var expire int64
	err := row.Scan(
		&session.UserID,
		&create,
		&expire,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return base.ErrTokenNotFound
		}

		lg.Error("query db", zap.Error(err))
		return base.ErrDatabaseQuery
	}

	session.CreateTime = time.UnixMicro(create)
	session.ExpireTime = time.UnixMicro(expire)

	return nil
}

func (c *Client) AddSession(ctx context.Context, lg *zap.Logger, session *base.Session) error {
	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	_, err := c.db.ExecContext(ctx, `
	INSERT INTO public.sessions (
		  token
		, user_id
		, create_ts
		, expire_ts
	)
	VALUES
		($1, $2, $3, $4)
	;
	`,
		session.Token,
		session.UserID,
		session.CreateTime.UnixMicro(),
		session.ExpireTime.UnixMicro(),
	)
	if err != nil {
		lg.Error("query db", zap.Error(err))
		return base.ErrDatabaseQuery
	}

	return nil
}
