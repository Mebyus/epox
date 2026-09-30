package http

import (
	"net/http"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func RenderError(c *hits.Context, e error) {
	body := ErrorBody{Error: e.Error()}
	var status int

	switch e.(type) {
	case *BadReqError:
		status = http.StatusBadRequest
	case *base.AuthError:
		status = http.StatusUnauthorized
	case *base.ServerError:
		status = http.StatusInternalServerError
	default:
		if e == hits.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusBadRequest
		}
	}

	c.RenderJSON(status, &body)
}

type BadReqError struct {
	msg string
}

func (e *BadReqError) Error() string {
	return e.msg
}

func NewBadReqError(msg string) error {
	return &BadReqError{msg: msg}
}

var (
	ErrEmptyLogin    = NewBadReqError("empty login")
	ErrEmptyPassword = NewBadReqError("empty password")

	ErrEmptyTitle     = NewBadReqError("empty title")
	ErrEmptyTaskID    = NewBadReqError("empty task id")
	ErrEmptyTaskState = NewBadReqError("empty task state")
	ErrEmptyTag       = NewBadReqError("empty tag")
	ErrNoTags         = NewBadReqError("no tags")
	ErrTagSpaces      = NewBadReqError("tag contains spaces")

	ErrBadTaskID = NewBadReqError("bad task id")
	ErrBadTagID  = NewBadReqError("bad tag id")

	ErrEmptyPath        = NewBadReqError("empty path")
	ErrEmptyPathSegment = NewBadReqError("empty path segment")
)
