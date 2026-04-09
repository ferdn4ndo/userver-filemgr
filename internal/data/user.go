package data

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserPublic is exposed in file owner JSON (legacy serializer shape).
type UserPublic struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	SystemName     string    `json:"system_name"`
	IsAdmin        bool      `json:"is_admin"`
	IsActive       bool      `json:"is_active"`
	RegisteredAt   time.Time `json:"registered_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	TotalDownloads int       `json:"total_downloads"`
	TotalUploads   int       `json:"total_uploads"`
}

func (d *DB) UserPublic(ctx context.Context, id uuid.UUID) (*UserPublic, error) {
	var u struct {
		ID             uuid.UUID `db:"id"`
		Username       string    `db:"username"`
		SystemName     string    `db:"system_name"`
		IsAdmin        bool      `db:"is_admin"`
		IsActive       bool      `db:"is_active"`
		RegisteredAt   time.Time `db:"registered_at"`
		LastActivityAt time.Time `db:"last_activity_at"`
	}
	err := d.db.GetContext(ctx, &u, `SELECT id, username, system_name, is_admin, is_active, registered_at, last_activity_at FROM core_customuser WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	var dl, up int
	_ = d.db.GetContext(ctx, &dl, `SELECT COUNT(*) FROM core_storagefiledownload WHERE owner_id = $1`, id)
	_ = d.db.GetContext(ctx, &up, `SELECT COUNT(*) FROM core_storagefile WHERE created_by_id = $1`, id)
	return &UserPublic{
		ID: u.ID, Username: u.Username, SystemName: u.SystemName, IsAdmin: u.IsAdmin, IsActive: u.IsActive,
		RegisteredAt: u.RegisteredAt, LastActivityAt: u.LastActivityAt, TotalDownloads: dl, TotalUploads: up,
	}, nil
}
