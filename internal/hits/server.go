package hits

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type ServerConfig struct {
	Address string

	CertKey  string
	CertFile string

	ServeGlobalTimeout       time.Duration
	RequestReadTimeout       time.Duration
	RequestReadHeaderTimeout time.Duration
	ResponseWriteTimeout     time.Duration
	KeepAliveIdleTimeout     time.Duration

	ShutdownTimeout time.Duration

	MaxRequestHeaderSize int
	MaxRequestBodySize   int
}

func NewServer(cfg *ServerConfig, lg *zap.Logger, handler http.Handler) *Server {
	if handler == nil {
		panic("nil handler")
	}
	if cfg.Address == "" {
		panic("empty address")
	}
	if cfg.MaxRequestBodySize < 0 {
		panic("bad value")
	}
	if cfg.MaxRequestHeaderSize < 0 {
		panic("bad value")
	}
	if cfg.ServeGlobalTimeout < 0 {
		panic("bad value")
	}
	if cfg.ShutdownTimeout <= 0 {
		panic("bad value")
	}

	s := &Server{
		cfg: *cfg,
		lg:  lg.Named("http"),

		hs: http.Server{
			Addr:    cfg.Address,
			Handler: modifyHandlerWithLimits(cfg, handler),

			ReadTimeout:       cfg.RequestReadTimeout,
			ReadHeaderTimeout: cfg.RequestReadHeaderTimeout,
			WriteTimeout:      cfg.ResponseWriteTimeout,
			IdleTimeout:       cfg.KeepAliveIdleTimeout,
			MaxHeaderBytes:    int(cfg.MaxRequestHeaderSize),
		},
	}
	s.hs.BaseContext = s.getBaseContext
	return s
}

type Server struct {
	hs  http.Server
	cfg ServerConfig

	lg *zap.Logger

	// Global context for shutting down server and served requests.
	// Must be set once, before starting server, and never altered
	// after that.
	ctx context.Context
}

func (s *Server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	s.ctx = ctx

	g.Go(s.waitShutdown)
	g.Go(s.listenAndServe)

	return g.Wait()
}

func (s *Server) waitShutdown() error {
	lg := s.lg

	<-s.ctx.Done()
	lg.Info("shutdown server")

	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()

	return s.hs.Shutdown(ctx)
}

func (s *Server) listenAndServe() error {
	lg := s.lg

	lg.Info("start server", zap.String("addr", s.cfg.Address))
	defer lg.Info("exit server")

	var err error
	if s.cfg.CertFile != "" {
		err = s.hs.ListenAndServeTLS(s.cfg.CertFile, s.cfg.CertKey)
	} else {
		err = s.hs.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) getBaseContext(net.Listener) context.Context {
	return s.ctx
}

type limitHandler struct {
	base http.Handler

	timeout   time.Duration
	bodyLimit int64
}

func newLimitHandler(handler http.Handler, timeout time.Duration, bodyLimit int64) limitHandler {
	return limitHandler{
		base:      handler,
		timeout:   timeout,
		bodyLimit: bodyLimit,
	}
}

func (h limitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.bodyLimit != 0 {
		r.Body = NewLimitReader(r.Body, h.bodyLimit)
	}

	if h.timeout != 0 {
		ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
		defer cancel()

		r = r.WithContext(ctx)
	}

	h.base.ServeHTTP(w, r)
}

func modifyHandlerWithLimits(cfg *ServerConfig, handler http.Handler) http.Handler {
	if cfg.MaxRequestBodySize == 0 && cfg.ServeGlobalTimeout == 0 {
		// no modifications needed
		return handler
	}
	return newLimitHandler(handler, cfg.ServeGlobalTimeout, int64(cfg.MaxRequestBodySize))
}
