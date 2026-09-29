package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/mebyus/epox/internal/base"
)

func (g *Logic) Login(ctx context.Context, lg *zap.Logger, data *base.LoginData) (*base.Session, error) {
	lg = lg.Named("login")

	auth := base.AuthData{Login: data.Login}
	err := g.db.LoadAuthData(ctx, lg, &auth)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(auth.Password), []byte(data.Password))
	if err != nil {
		return nil, base.ErrCredentialsMismatch
	}

	now := time.Now()
	s := &base.Session{
		CreateTime: now,
		ExpireTime: now.Add(24 * time.Hour * 7),
		Token:      uuid.NewString(),
		UserID:     auth.UserID,
	}

	err = g.db.AddSession(ctx, lg, s)
	if err != nil {
		return nil, err
	}

	lg.Info("new session", zap.Uint64("user.id", uint64(s.UserID)))
	return s, nil
}

func (g *Logic) LoadSession(ctx context.Context, lg *zap.Logger, session *base.Session) error {
	lg = lg.Named("auth")
	return g.db.LoadSession(ctx, lg, session)
}
