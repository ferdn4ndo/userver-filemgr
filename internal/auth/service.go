package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

var ErrUnauthorized = errors.New("unauthorized")

// User is a row from core_customuser.
type User struct {
	ID             uuid.UUID `db:"id"`
	Username       string    `db:"username"`
	SystemName     string    `db:"system_name"`
	IsAdmin        bool      `db:"is_admin"`
	IsActive       bool      `db:"is_active"`
	RegisteredAt   time.Time `db:"registered_at"`
	LastActivityAt time.Time `db:"last_activity_at"`
}

// Service resolves Token / Bearer credentials to a local user (legacy DB rows).
type Service struct {
	db     lib.Database
	client *Client
}

func NewService(db lib.Database, env lib.Env) *Service {
	return &Service{db: db, client: NewClient(env.AuthHost, env.AuthHTTPTimeout)}
}

func (s *Service) userByToken(ctx context.Context, token string) (*User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, `
		SELECT u.id, u.username, u.system_name, u.is_admin, u.is_active, u.registered_at, u.last_activity_at
		FROM core_customuser u
		JOIN core_usertoken t ON t.user_id = u.id
		WHERE t.token = $1 AND t.expires_at > NOW()`, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func parseAuthTime(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05-07:00", v)
	}
	return t
}

// Authenticate resolves a raw access token using cached core_usertoken or GET uServer-Auth /auth/me (Bearer).
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*User, error) {
	if rawToken == "" {
		return nil, ErrUnauthorized
	}
	u, err := s.userByToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	if u != nil {
		if !u.IsActive {
			return nil, ErrUnauthorized
		}
		return u, nil
	}
	me, tt, err := s.client.ValidateBearer(ctx, rawToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	user, err := s.loadOrCreateUserFromMe(ctx, tx, me)
	if err != nil {
		return nil, err
	}
	if err := s.cacheBearerToken(ctx, tx, user.ID, rawToken, tt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &user, nil
}

// WithTx runs fn inside a transaction (helper for tests).
func (s *Service) WithTx(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
