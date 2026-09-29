package db

import (
	"slices"
	"strconv"
	"strings"
)

// builder is a helper object for building complex
// dynamic queries
//
// Zero value is ready to use.
type builder struct {
	buf strings.Builder

	args []any
}

// init builder with starting size for internal buffers
func (g *builder) init(argc, size int) {
	if g.args == nil {
		g.args = make([]any, 0, argc)
	} else if cap(g.args) < argc {
		g.args = slices.Grow(g.args, argc-cap(g.args))
	}
	g.buf.Grow(size)
}

func (g *builder) puts(s string) {
	g.buf.WriteString(s)
}

func (g *builder) putb(b byte) {
	g.buf.WriteByte(b)
}

func (g *builder) puti(i int) {
	g.puts(strconv.FormatInt(int64(i), 10))
}

func (g *builder) add(arg any) {
	g.args = append(g.args, arg)
	g.putb('$')
	g.puti(len(g.args))
}

func (g *builder) take() (string, []any) {
	query := g.buf.String()
	args := g.args

	g.buf.Reset()
	g.args = g.args[:0]

	return query, args
}
