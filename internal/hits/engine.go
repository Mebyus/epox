package hits

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EngineConfig struct {
	RenderError ErrorHandler
	RenderPanic PanicHandler

	RenderNotFound Handler
}

type Handler func(*Context) error

type Engine struct {
	router Router

	renderError ErrorHandler
	renderPanic PanicHandler

	renderNotFound Handler

	lg *zap.Logger
}

var _ http.Handler = &Engine{}

func NewEngine(cfg *EngineConfig, lg *zap.Logger, router Router) *Engine {
	if router == nil {
		panic("nit router")
	}

	var renderError ErrorHandler
	if cfg.RenderError != nil {
		renderError = cfg.RenderError
	} else {
		renderError = RenderError
	}

	var renderPanic PanicHandler
	if cfg.RenderPanic != nil {
		renderPanic = cfg.RenderPanic
	} else {
		renderPanic = RenderPanic
	}

	var renderNotFound Handler
	if cfg.RenderNotFound != nil {
		renderNotFound = cfg.RenderNotFound
	} else {
		renderNotFound = RenderNotFound
	}

	g := &Engine{
		lg:     lg,
		router: router,

		renderError: renderError,
		renderPanic: renderPanic,

		renderNotFound: renderNotFound,
	}
	return g
}

func (g *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trace := uuid.NewString()
	lg := g.lg.With(zap.String("trace", trace))
	c := Context{
		Start: time.Now(),
		Trace: trace,
		Req:   r,
		Log:   lg,
		rw:    w,
		rh:    w.Header(),
	}
	if r.URL.RawQuery != "" {
		c.qv = r.URL.Query()
	}
	c.rh.Add("Trace", trace)

	defer g.exitAndHandlePanic(&c)
	lg.Info("handle request", zap.String("method", r.Method), zap.String("url.path", r.URL.Path))

	chain, err := g.router.Dispatch(&c)
	if err != nil {
		g.renderError(&c, err)
		return
	}

	if len(chain) == 0 {
		err := g.renderNotFound(&c)
		if err != nil {
			g.renderError(&c, err)
		}
		return
	}

	for _, f := range chain {
		err := f(&c)
		if err != nil {
			if c.fin {
				// TODO: add post-write error handler
				lg.Warn("handler error after response finalized", zap.Error(err))
			} else {
				g.renderError(&c, err)
			}
			return
		}
	}
}
