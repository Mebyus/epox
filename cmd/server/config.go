package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Database DatabaseParams
	Server   ServerConfig

	Log LogConfig
}

func loadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("empty config path")
	}
	path = filepath.Clean(path)
	if path == "" || path == "." || strings.Contains(path, "..") {
		return nil, errors.New("bad config path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_ = data

	// TODO: check path and load config from file
	return testConfig, nil
}

var testConfig = &Config{
	Database: DatabaseParams{
		Host: "localhost",
		Port: 5432,
		Name: "epox",

		User:     "postgres",
		Password: "123",

		QueryTimeout: 4 * time.Second,
		PingTimeout:  2 * time.Second,

		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 5 * time.Minute,

		MaxOpenConns: 32,
		MaxIdleConns: 8,
	},
	Server: ServerConfig{
		TLS: ServerTLSConfig{
			// CertKey:  "tls/key.pem",
			// CertFile: "tls/cert.pem",
		},

		Address: "localhost:8733",

		ShutdownTimeout: 5 * time.Second,

		ServeGlobalTimeout:       15 * time.Second,
		RequestReadHeaderTimeout: 2 * time.Second,
		RequestReadTimeout:       5 * time.Second,
		ResponseWriteTimeout:     5 * time.Second,
		KeepAliveIdleTimeout:     60 * time.Second,

		MaxRequestHeaderSize: 1 << 16,
		MaxRequestBodySize:   1 << 22,
	},
	Log: LogConfig{
		File: "data/log/epox.log",

		MaxSize: 1 << 24,
		MaxNum:  8,
	},
}

type ServerConfig struct {
	TLS ServerTLSConfig

	Address string

	ServeGlobalTimeout       time.Duration
	RequestReadTimeout       time.Duration
	RequestReadHeaderTimeout time.Duration
	ResponseWriteTimeout     time.Duration
	KeepAliveIdleTimeout     time.Duration

	ShutdownTimeout time.Duration

	MaxRequestHeaderSize uint32
	MaxRequestBodySize   uint32
}

type ServerTLSConfig struct {
	CertKey  string
	CertFile string
}

type DatabaseParams struct {
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

type LogConfig struct {
	// Path to log file.
	File string

	MaxSize uint64
	MaxNum  uint32
}
