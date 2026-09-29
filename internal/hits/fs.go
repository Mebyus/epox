package hits

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// RenderFile writes file contents into response body and
// sets headers typically used when serving files in
// http context.
//
//	Content-Length
//	Content-Type
//	Last-Modified
//
// On success returns (true, nil).
// First return value indicates whether or not this call
// has already written response headers.
func (c *Context) RenderFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, ErrNotFile
	}
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	size := info.Size()
	modtime := info.ModTime()

	c.SetHeader("Content-Type", getTypeFromExt(filepath.Ext(path)))
	c.SetHeader("Content-Length", strconv.FormatInt(size, 10))
	if !modtime.IsZero() {
		c.SetHeader("Last-Modified", modtime.UTC().Format(headerTimeFormat))
	}

	c.writeHeaders(http.StatusOK)
	if size > 0 {
		_, err = io.CopyN(c.rw, file, size)
	}

	c.fin = true
	return true, err
}

func getTypeFromExt(ext string) string {
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json"
	default:
		return "text/plain"
	}

}

const headerTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// CopyFileBytes copies regular file contents into supplied writer.
//
// Returns number of bytes copied and last modtime of file
func CopyFileBytes(dst io.Writer, path string) (uint64, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	if !info.Mode().IsRegular() {
		return 0, time.Time{}, ErrNotFile
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer file.Close()

	size := info.Size()
	modtime := info.ModTime()

	var n int64
	if size > 0 {
		n, err = io.CopyN(dst, file, size)
	}

	return uint64(n), modtime, err
}
