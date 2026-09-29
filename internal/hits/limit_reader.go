package hits

import (
	"errors"
	"io"
)

var ErrBodyLimitReached = errors.New("body limit reached")

// LimitReader behaves much like io.LimitedReader, with few subtle differences:
//
//   - decorates io.ReadCloser instead of io.Reader
//
//   - when reader is exhausted by reaching the set limit it returns distinct
//     error ErrBodyLimitReached, not io.EOF
//
//   - if EOF is reached upon simultaneously reaching the set limit, then io.EOF is returned
type LimitReader struct {
	r  io.ReadCloser // underlying reader
	nr int64         // number of bytes remaining until limit is reached
}

// Explicit interface implementation check.
var _ io.ReadCloser = &LimitReader{}

func NewLimitReader(r io.ReadCloser, limit int64) *LimitReader {
	return &LimitReader{r: r, nr: limit}
}

func (l *LimitReader) Read(p []byte) (int, error) {
	if l.nr == 0 {
		return 0, io.EOF
	}
	if l.nr < 0 {
		return 0, ErrBodyLimitReached
	}

	lim := l.nr
	if int64(len(p)) > lim+1 {
		p = p[:lim+1]
	}
	n, err := l.r.Read(p)
	l.nr -= int64(n)

	if int64(n) > lim {
		n = int(lim)
		if err == nil || err == io.EOF {
			err = ErrBodyLimitReached
		}
	}
	return n, err
}

func (l *LimitReader) Close() error {
	return l.r.Close()
}
