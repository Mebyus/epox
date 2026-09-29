/*
	package froll

Implements rotating log file suitable for logging.
*/
package froll

import (
	"io"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	Path    string
	MaxSize uint64
	MaxNum  uint32
}

// Roller implements io.Writer, io.Closer and io.Syncer interfaces.
// It acts like a sink for writes by rotating underlying files.
type Roller struct {
	// access to reusable byte buffers
	pool sync.Pool

	// for sending data-to-be-written to background worker
	sink chan *buffer

	// for sending sync signal
	ss chan struct{}

	// for sending close signal
	cs chan struct{}
}

var _ io.WriteCloser = &Roller{}

func NewRoller(cfg *Config) (*Roller, error) {
	if cfg.Path == "" {
		panic("empty path")
	}

	sink := make(chan *buffer, 32)
	ss := make(chan struct{})
	cs := make(chan struct{})
	swap := make(chan swap, 1)

	dir := filepath.Dir(cfg.Path)
	if dir == "." {
		dir = ""
	}
	ext := filepath.Ext(cfg.Path)

	r := &Roller{
		sink: sink,
		ss:   ss,
		cs:   cs,
		pool: sync.Pool{New: func() any {
			return &buffer{}
		}},
	}
	w := worker{
		maxsize: cfg.MaxSize,
		maxnum:  cfg.MaxNum,

		dir:    dir,
		ext:    ext,
		csuf:   getCompressedName(ext),
		prefix: strings.TrimSuffix(cfg.Path, ext),

		tap:  sink,
		swap: swap,
		ss:   ss,
		cs:   cs,
		pool: &r.pool,
	}
	slots, err := w.init()
	if err != nil {
		return nil, err
	}

	gc := gc{
		dir:     dir,
		symlink: cfg.Path,
		tap:     swap,
		cbuf:    cycle{slots: slots},
	}
	go gc.watch()
	go w.watch()

	return r, nil
}

func (r *Roller) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	buf := r.getBuffer()
	buf.put(data)
	r.sink <- buf

	return len(data), nil
}

func (r *Roller) Sync() error {
	r.ss <- struct{}{}
	return nil
}

func (r *Roller) Close() error {
	r.cs <- struct{}{}
	return nil
}
