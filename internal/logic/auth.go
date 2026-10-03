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
		ExpireTime: now.Add(sttl),
		Token:      uuid.NewString(),
		UserID:     auth.UserID,
	}

	err = g.db.AddSession(ctx, lg, s)
	if err != nil {
		return nil, err
	}
	g.auth.put(s.Token, sent{
		user:  s.UserID,
		expts: uint64(s.ExpireTime.UnixMicro()),
	})

	lg.Info("new session", zap.Uint64("user.id", uint64(s.UserID)))
	return s, nil
}

func (g *Logic) LoadSession(ctx context.Context, lg *zap.Logger, session *base.Session) error {
	ent, code := g.auth.get(session.Token, uint64(time.Now().UnixMicro()))
	switch code {
	case sinv:
		// continue execution, try to load from database
	case sval:
		session.UserID = ent.user
		session.ExpireTime = time.UnixMicro(int64(ent.expts))
		return nil
	case sexp:
		_ = g.db.RemoveExpiredSession(ctx, lg, session.Token)
		return base.ErrTokenNotFound // TODO: return separate expired token error?
	}

	lg = lg.Named("auth")
	err := g.db.LoadSession(ctx, lg, session)
	if err != nil {
		return err
	}

	g.auth.put(session.Token, sent{
		user:  session.UserID,
		expts: uint64(session.ExpireTime.UnixMicro()),
	})
	return nil
}
