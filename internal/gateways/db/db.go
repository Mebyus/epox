package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Params struct {
	Host string

	// Database name.
	Name string

	User     string
	Password string

	QueryTimeout time.Duration
	PingTimeout  time.Duration

	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	MaxOpenConns uint32
	MaxIdleConns uint32

	Port uint32
}

func applyParams(db *sql.DB, params *Params) {
	db.SetMaxOpenConns(int(params.MaxOpenConns))
	db.SetMaxIdleConns(int(params.MaxIdleConns))
	db.SetConnMaxLifetime(params.ConnMaxLifetime)
	db.SetConnMaxIdleTime(params.ConnMaxIdleTime)
}

func formatDSN(params *Params) string {
	if params.Host == "" {
		panic("empty host")
	}
	if params.User == "" {
		panic("empty user")
	}
	if params.Name == "" {
		panic("empty database name")
	}

	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		params.Host,
		params.User,
		params.Password,
		params.Name,
		params.Port,
	)
}

type Client struct {
	timeout time.Duration

	db *sql.DB
}

func ConnectAndPing(ctx context.Context, params *Params) (*Client, error) {
	if params.PingTimeout <= 0 {
		panic("invalid ping timeout")
	}
	if params.QueryTimeout <= 0 {
		panic("invalid query timeout")
	}

	dsn := formatDSN(params)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	applyParams(db, params)

	err = pingTimeout(ctx, db, params.PingTimeout)
	if err != nil {
		return nil, err
	}

	return &Client{
		db:      db,
		timeout: params.QueryTimeout,
	}, nil
}

func pingTimeout(ctx context.Context, db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	return nil
}

func (c *Client) newQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) Close() error {
	return c.db.Close()
}

const truncateTablesQuery = `
TRUNCATE
	  public.users
	, public.invites
	, public.sessions
	, public.groups
	, public.participants
	, public.messages
;`

// Purge truncate all data in database.
// Use only for consistent testing.
//
// Do not use in regular code (or do so with extreme caution).
func (c *Client) Purge(ctx context.Context, lg *zap.Logger) error {
	ctx, cancel := c.newQueryContext(ctx)
	defer cancel()

	_, err := c.db.ExecContext(ctx, truncateTablesQuery)
	return err
}
