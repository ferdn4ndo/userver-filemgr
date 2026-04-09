package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (s *Service) loadOrCreateUserFromMe(ctx context.Context, tx *sqlx.Tx, me *MeResponse) (User, error) {
	var user User
	err := tx.GetContext(ctx, &user, `SELECT id, username, system_name, is_admin, is_active, registered_at, last_activity_at
		FROM core_customuser WHERE username = $1 LIMIT 1`, me.Username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		uid := uuid.New()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO core_customuser (
				id, password, username, system_name, is_admin, is_active,
				registered_at, last_activity_at, updated_at, is_superuser, last_login
			) VALUES ($1, '!', $2, $3, $4, true, NOW(), NOW(), NOW(), false, NULL)`,
			uid, me.Username, me.SystemName, me.IsAdmin)
		if err != nil {
			return User{}, err
		}
		return User{
			ID: uid, Username: me.Username, SystemName: me.SystemName, IsAdmin: me.IsAdmin, IsActive: true,
			RegisteredAt: time.Now(), LastActivityAt: time.Now(),
		}, nil
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE core_customuser SET system_name = $1, is_admin = $2, is_active = true, last_activity_at = NOW(), updated_at = NOW()
		WHERE id = $3`,
		me.SystemName, me.IsAdmin, user.ID)
	if err != nil {
		return User{}, err
	}
	user.SystemName = me.SystemName
	user.IsAdmin = me.IsAdmin
	user.IsActive = true
	return user, nil
}

func (s *Service) cacheBearerToken(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, rawToken string, tt *tokenTimes) error {
	if tt == nil {
		tt = &tokenTimes{}
	}
	issued := parseAuthTime(tt.IssuedAt)
	expires := parseAuthTime(tt.ExpiresAt)
	if issued.IsZero() {
		issued = time.Now().UTC()
	}
	if expires.IsZero() {
		expires = issued.Add(24 * time.Hour)
	}
	tokID := uuid.New()
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_usertoken (uuid, token, issued_at, expires_at, created_at, user_id)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (token) DO UPDATE SET expires_at = EXCLUDED.expires_at, user_id = EXCLUDED.user_id`,
		tokID, rawToken, issued, expires, userID)
	return err
}
