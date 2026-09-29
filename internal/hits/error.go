package hits

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"go.uber.org/zap"
)

var (
	ErrNotFile  = errors.New("not a file")
	ErrNotFound = errors.New("not found")
)

type ErrorHandler func(*Context, error)

type PanicHandler func(c *Context, cause any, stack []byte)

// RenderError default ErrorHandler.
func RenderError(c *Context, e error) {
	c.RenderText(http.StatusBadRequest, e.Error())
}

func RenderNotFound(c *Context) error {
	c.RenderStatus(http.StatusNotFound)
	return nil
}

// RenderPanic default PanicHandler.
func RenderPanic(c *Context, cause any, stack []byte) {
	dumpPanicWithStack(os.Stderr, c.Trace, cause, stack)
	c.RenderPanicStatus()
}

func (g *Engine) exitAndHandlePanic(c *Context) {
	p := recover()
	if p == nil {
		status := c.status
		if status == 0 {
			status = http.StatusOK
		}
		if !c.fin {
			c.Log.Warn("no explicit response")
			c.rw.WriteHeader(status)
		}
		c.Log.Debug("handler exit", zap.Int("status", status))
		return
	}

	c.Log.Warn("handler panic")

	var buf [1 << 16]byte
	n := runtime.Stack(buf[:], false)

	g.renderPanic(c, p, buf[:n])
	c.Log.Debug("handler exit", zap.Int("status", c.status))
}

func dumpPanicWithStack(out io.Writer, trace string, p any, stack []byte) {
	fmt.Fprintf(out, "===== %s =====\n", trace)
	fmt.Fprintf(out, "panic: %v\n\n", p)
	out.Write(stack)
	fmt.Fprintln(out)
}
