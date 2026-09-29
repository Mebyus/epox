package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/endpoints/http"
	"github.com/mebyus/epox/internal/gateways/db"
	"github.com/mebyus/epox/internal/hits"
	"github.com/mebyus/epox/internal/logic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("config file not specified")
		os.Exit(1)
	}
	configPath := os.Args[1]

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = setupLoggerAndRun(cfg, newProcessContext())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupLoggerAndRun(cfg *Config, ctx context.Context) error {
	lg, logCloser, err := newLogger(&cfg.Log)
	if err != nil {
		return fmt.Errorf("setup logger: %v", err)
	}
	defer lg.Sync()
	defer logCloser.Close()

	lg.Debug("start")

	err = run(cfg, ctx, lg)
	if err != nil {
		lg.Error("exit", zap.Error(err))
		return err
	}

	lg.Info("exit")
	return nil
}

func run(cfg *Config, ctx context.Context, lg *zap.Logger) error {
	db, err := db.ConnectAndPing(ctx, &db.Params{
		Host: cfg.Database.Host,
		Port: cfg.Database.Port,
		Name: cfg.Database.Name,

		User:     cfg.Database.User,
		Password: cfg.Database.Password,

		QueryTimeout: cfg.Database.QueryTimeout,
		PingTimeout:  cfg.Database.PingTimeout,

		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,

		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		return fmt.Errorf("setup db connection: %v", err)
	}

	logic := logic.New(db)
	err = logic.Init(ctx, lg)
	if err != nil {
		return fmt.Errorf("init service logic: %v", err)
	}

	engine, err := http.New(lg, logic)
	if err != nil {
		return fmt.Errorf("setup http server: %v", err)
	}

	serverConfig := hits.ServerConfig{
		CertKey:  cfg.Server.TLS.CertKey,
		CertFile: cfg.Server.TLS.CertFile,

		Address: cfg.Server.Address,

		ShutdownTimeout: cfg.Server.ShutdownTimeout,

		ServeGlobalTimeout:       cfg.Server.ServeGlobalTimeout,
		RequestReadHeaderTimeout: cfg.Server.RequestReadHeaderTimeout,
		RequestReadTimeout:       cfg.Server.RequestReadTimeout,
		ResponseWriteTimeout:     cfg.Server.ResponseWriteTimeout,
		KeepAliveIdleTimeout:     cfg.Server.KeepAliveIdleTimeout,

		MaxRequestHeaderSize: int(cfg.Server.MaxRequestHeaderSize),
		MaxRequestBodySize:   int(cfg.Server.MaxRequestBodySize),
	}
	server := hits.NewServer(&serverConfig, lg, engine)
	err = server.Run(ctx)
	if err != nil {
		return fmt.Errorf("http server exit: %v", err)
	}
	return nil
}
