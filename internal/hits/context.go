package hits

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

type Context struct {
	// Server time when request handling started.
	Start time.Time

	// Can be set by middleware.
	Auth any

	// Use for logging and request spanning.
	//
	// Read-only.
	Trace string

	rw http.ResponseWriter

	// response headers
	rh http.Header

	// parsed url query parameters
	qv url.Values

	// Recorded response status.
	// Does not affect handling logic.
	//
	// Use it for logging, analytics, dumps, etc.
	status int

	// Request being served.
	Req *http.Request

	Log *zap.Logger

	// Holds serving params (path variables for example).
	//
	// May be nil or empty if no params were detected.
	Params map[string]string

	// True if response (or at least response header) was
	// already written.
	fin bool
}

func (c *Context) ParseBodyJSON(v any) error {
	dec := json.NewDecoder(c.Req.Body)
	return dec.Decode(v)
}

// Query returns value of query parameter for the given key.
func (c *Context) Query(key string) string {
	return c.qv.Get(key)
}

// Header shortcut for getting value of request header.
func (c *Context) Header(key string) string {
	return c.Req.Header.Get(key)
}

// Cookie shortcut for getting value of request cookie.
func (c *Context) Cookie(key string) string {
	cookie, err := c.Req.Cookie(key)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.rw, cookie)
}

func (c *Context) SetHeader(key, value string) {
	c.rh.Add(key, value)
}

func (c *Context) RenderJSON(status int, v any) {
	c.writeHeadersJSON(status)

	enc := json.NewEncoder(c.rw)
	err := enc.Encode(v)
	if err != nil {
		c.Log.Error("encode response body", zap.Error(err))
	}
	c.fin = true
}

func (c *Context) RenderText(status int, text string) {
	c.Status(status)
	c.rw.WriteHeader(status)

	_, err := io.WriteString(c.rw, text)
	if err != nil {
		c.Log.Error("write response body", zap.Error(err))
	}
	c.fin = true
}

// RenderStatus write response with status code and nothing else.
// Response body will be empty.
func (c *Context) RenderStatus(status int) {
	c.Status(status)
	c.rw.WriteHeader(status)
	c.fin = true
}

// RenderPanicStatus sets InternalServerError status for response.
// Avoids limit of setting response status only once.
func (c *Context) RenderPanicStatus() {
	c.status = http.StatusInternalServerError
	c.rw.WriteHeader(http.StatusInternalServerError)
	c.fin = true
}

// Status records response status for future rendering.
// Panics if called more than once.
func (c *Context) Status(status int) {
	if c.status != 0 {
		panic("status was already set")
	}
	c.status = status
}

func (c *Context) writeHeadersJSON(status int) {
	c.Status(status)

	c.rh.Add("Content-Type", "application/json")
	c.rw.WriteHeader(status)
}

func (c *Context) writeHeaders(status int) {
	c.Status(status)
	c.rw.WriteHeader(status)
}
