package base

import (
	"strconv"
	"time"
)

type UserID uint64

func (i UserID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

// LoginData contains data from login request.
type LoginData struct {
	// Always not empty.
	Login string

	// Always not empty.
	//
	// Plain text of password entered by user.
	Password string
}

// AuthData contains necessary data to verify login request
// and create new session.
type AuthData struct {
	// Always not empty.
	Login string

	// Always not empty.
	//
	// Password hash.
	Password string

	UserID UserID
}

type Session struct {
	CreateTime time.Time
	ExpireTime time.Time

	// Token uniquely identifies user login session.
	Token string

	UserID UserID
}

type SessionEntry struct {
	Token  string
	UserID UserID

	// Unix microseconds timestamp when session expires.
	Expire uint64
}

type AuthError struct {
	msg string
}

func (e *AuthError) Error() string {
	return e.msg
}

func NewAuthError(msg string) error {
	return &AuthError{msg: msg}
}

var (
	ErrEmptyAuth     = NewAuthError("empty auth token")
	ErrBadAuthToken  = NewAuthError("bad auth token")
	ErrTokenNotFound = NewAuthError("auth token not found")

	ErrCredentialsMismatch = NewAuthError("credentials mismatch")
)
