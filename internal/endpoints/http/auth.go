package http

import (
	"net/http"
	"strings"

	"github.com/mebyus/epox/internal/base"
	"github.com/mebyus/epox/internal/hits"
)

const authTokenKey = "auth"

type LoginData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (p *provider) Login(c *hits.Context) error {
	var body LoginData
	err := c.ParseBodyJSON(&body)
	if err != nil {
		return err
	}

	var data base.LoginData
	err = validateLoginData(&body, &data)
	if err != nil {
		return err
	}

	session, err := p.logic.Login(c.Req.Context(), c.Log, &data)
	if err != nil {
		return err
	}

	setSessionCookie(c, session)
	c.RenderStatus(http.StatusOK)
	return nil
}

// Auth handles authorization logic for each incoming request.
func (p *provider) Auth(c *hits.Context) error {
	token := c.Cookie(authTokenKey)
	err := checkAuthToken(token)
	if err != nil {
		return err
	}

	session := base.Session{Token: token}
	err = p.logic.LoadSession(c.Req.Context(), c.Log, &session)
	if err != nil {
		return err
	}

	c.Auth = &session
	return nil
}

// SessionBody object for marshalling response body
// with session properties.
type SessionBody struct {
	Token      string `json:"token"`
	UserID     string `json:"user_id"`
	ExpireTime string `json:"expire_time"`
}

func (p *provider) GetSession(c *hits.Context) error {
	s, err := p.tryLoadSession(c)
	if err != nil {
		return err
	}
	if s == nil {
		return base.ErrTokenNotFound
	}

	body := SessionBody{
		Token:      s.Token,
		UserID:     s.UserID.String(),
		ExpireTime: formatOptZeroTime(s.ExpireTime),
	}
	c.RenderJSON(http.StatusOK, &body)
	return nil
}

// try to load session possibly stored (via token) in request context.
//
// Returns error only when its server error, not logic error.
//
// Should be used for optional checking if user is authorized
// upon rendering index page.
func (p *provider) tryLoadSession(c *hits.Context) (*base.Session, error) {
	token := c.Cookie(authTokenKey)
	err := checkAuthToken(token)
	if err != nil {
		return nil, nil
	}

	session := base.Session{Token: token}
	err = p.logic.LoadSession(c.Req.Context(), c.Log, &session)
	if err != nil {
		if err == base.ErrTokenNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func getUserID(c *hits.Context) base.UserID {
	return c.Auth.(*base.Session).UserID
}

func validateLoginData(body *LoginData, data *base.LoginData) error {
	login := strings.TrimSpace(body.Login)
	if login == "" {
		return ErrEmptyLogin
	}
	password := strings.TrimSpace(body.Password)
	if password == "" {
		return ErrEmptyPassword
	}

	data.Login = login
	data.Password = password
	return nil
}

func checkAuthToken(token string) error {
	if token == "" {
		return base.ErrEmptyAuth
	}
	return nil
}

func setSessionCookie(c *hits.Context, s *base.Session) {
	c.SetCookie(&http.Cookie{
		Name:     authTokenKey,
		Value:    s.Token,
		Expires:  s.ExpireTime,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(c *hits.Context) {
	c.SetCookie(&http.Cookie{
		Name:     authTokenKey,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}
