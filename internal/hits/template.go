package hits

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

type Content struct {
	// Fills Last-Modified header if not empty.
	ModTime time.Time

	// Fills Content-Type header if not empty.
	Type string

	// What to render in response body.
	//
	// Must not be nil.
	Data io.Reader

	// Fills Content-Length header.
	//
	// Also limits how many bytes are copied into response body
	// from provided reader.
	Size uint64
}

func (c *Context) RenderContent(con *Content) error {
	if con.Type != "" {
		c.SetHeader("Content-Type", con.Type)
	}
	c.SetHeader("Content-Length", strconv.FormatUint(con.Size, 10))
	if !con.ModTime.IsZero() {
		c.SetHeader("Last-Modified", con.ModTime.UTC().Format(headerTimeFormat))
	}

	c.writeHeaders(http.StatusOK)
	var err error
	if con.Size > 0 {
		_, err = io.CopyN(c.rw, con.Data, int64(con.Size))
	}

	c.fin = true
	return err
}

// RenBuf is a reusable render buffer.
//
// Satisfies io.Reader and io.Writer interfaces.
//
// Zero value is ready to use, empty buffer.
//
// Not safe for concurrent use.
type RenBuf struct {
	buf []byte

	// reader position (offset in bytes)
	pos uint64
}

var _ io.Reader = &RenBuf{}

func (g *RenBuf) Read(buf []byte) (int, error) {
	if g.pos >= uint64(len(g.buf)) {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}

	n := copy(buf, g.buf[g.pos:])
	g.pos += uint64(n)
	return n, nil
}

func (g *RenBuf) Write(data []byte) (int, error) {
	g.buf = append(g.buf, data...)
	return len(data), nil
}

func (g *RenBuf) Puts(s string) {
	g.buf = append(g.buf, s...)
}

// Reset buffer to empty state, but keep underlying memory
// for future use.
func (g *RenBuf) Reset() {
	g.buf = g.buf[:0]
	g.pos = 0
}

// Size returns how many bytes are stored in buffer.
func (g *RenBuf) Size() uint64 {
	return uint64(len(g.buf))
}
