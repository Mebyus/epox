package http

import (
	"path/filepath"
	"sync"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/hits"
	"github.com/mebyus/epox/internal/logic"
)

func New(lg *zap.Logger, logic *logic.Logic) (*hits.Engine, error) {
	lg = lg.Named("http")

	const static = "ui/web" // TODO: take this from config
	provider := provider{
		rpool: sync.Pool{New: func() any {
			return &hits.RenBuf{}
		}},
		logic:  logic,
		static: static,
		index:  filepath.Join(static, "index.html"),
	}
	root := hits.NewRoot()

	root.Add("GET", "ext", provider.Index)
	root.Add("GET", "ext/static/**", provider.Static)
	root.Add("POST", "ext/login", provider.Login)

	// external endpoints with required authorization
	ext := root.Group("ext", provider.Auth)

	ext.Add("GET", "tasks/active", provider.GetActiveTasks)
	ext.Add("GET", "tasks/history", provider.GetHistoryTasks)
	ext.Add("POST", "task", provider.AddActiveTask)
	ext.Add("POST", "task/state", provider.ChangeTaskState)

	ext.Add("POST", "tags", provider.SaveTags)

	router := hits.NewRouter(root)
	return hits.NewEngine(&hits.EngineConfig{
		RenderError: RenderError,
	}, lg, router), nil
}

// provider is a collection of http endpoints
//
// each endpoint provides its own handler
type provider struct {
	rpool sync.Pool

	// system path to directory with static server content
	static string

	// system path to index page
	index string

	logic *logic.Logic
}

func (p *provider) getRenBuf() *hits.RenBuf {
	return p.rpool.Get().(*hits.RenBuf)
}

func (p *provider) putRenBuf(g *hits.RenBuf) {
	g.Reset()
	p.rpool.Put(g)
}
