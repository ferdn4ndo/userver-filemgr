package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type StorageUser struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at,omitempty"`
	MayWrite  bool       `db:"may_write" json:"may_write"`
	MayRead   bool       `db:"may_read" json:"may_read"`
	StorageID uuid.UUID  `db:"storage_id" json:"storage"`
	UserID    uuid.UUID  `db:"user_id" json:"user"`
}

func (d *DB) ListStorageUsers(ctx context.Context, storageID uuid.UUID) ([]StorageUser, error) {
	var rows []StorageUser
	err := d.db.SelectContext(ctx, &rows, `
		SELECT id, created_at, updated_at, may_write, may_read, storage_id, user_id
		FROM core_storageuser WHERE storage_id = $1 ORDER BY created_at`, storageID)
	return rows, err
}

func (d *DB) GetStorageUser(ctx context.Context, storageID, rowID uuid.UUID) (*StorageUser, error) {
	var r StorageUser
	err := d.db.GetContext(ctx, &r, `
		SELECT id, created_at, updated_at, may_write, may_read, storage_id, user_id
		FROM core_storageuser WHERE storage_id = $1 AND id = $2`, storageID, rowID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) InsertStorageUser(ctx context.Context, storageID, userID uuid.UUID, mayRead, mayWrite bool, actor uuid.UUID) (*StorageUser, error) {
	id := uuid.New()
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO core_storageuser (id, created_at, updated_at, may_write, may_read, storage_id, user_id, created_by_id, updated_by_id)
		VALUES ($1, NOW(), NOW(), $2, $3, $4, $5, $6, $6)`, id, mayWrite, mayRead, storageID, userID, actor)
	if err != nil {
		return nil, err
	}
	return d.findStorageUserByPair(ctx, storageID, userID)
}

func (d *DB) findStorageUserByPair(ctx context.Context, storageID, userID uuid.UUID) (*StorageUser, error) {
	var r StorageUser
	err := d.db.GetContext(ctx, &r, `
		SELECT id, created_at, updated_at, may_write, may_read, storage_id, user_id FROM core_storageuser WHERE storage_id = $1 AND user_id = $2`,
		storageID, userID)
	return &r, err
}

func (d *DB) UpdateStorageUser(ctx context.Context, storageID, rowID uuid.UUID, mayRead, mayWrite *bool, actor uuid.UUID) (*StorageUser, error) {
	su, err := d.GetStorageUser(ctx, storageID, rowID)
	if err != nil {
		return nil, err
	}
	if mayRead != nil {
		su.MayRead = *mayRead
	}
	if mayWrite != nil {
		su.MayWrite = *mayWrite
	}
	_, err = d.db.ExecContext(ctx, `
		UPDATE core_storageuser SET may_read = $3, may_write = $4, updated_at = NOW(), updated_by_id = $5
		WHERE storage_id = $1 AND id = $2`, storageID, rowID, su.MayRead, su.MayWrite, actor)
	if err != nil {
		return nil, err
	}
	return d.GetStorageUser(ctx, storageID, rowID)
}

func (d *DB) DeleteStorageUser(ctx context.Context, storageID, rowID uuid.UUID) error {
	res, err := d.db.ExecContext(ctx, `DELETE FROM core_storageuser WHERE storage_id = $1 AND id = $2`, storageID, rowID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *DB) StorageUserMayWrite(ctx context.Context, storageID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := d.db.GetContext(ctx, &ok, `
		SELECT may_write FROM core_storageuser WHERE storage_id = $1 AND user_id = $2`, storageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (d *DB) StorageUserMayRead(ctx context.Context, storageID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := d.db.GetContext(ctx, &ok, `
		SELECT may_read FROM core_storageuser WHERE storage_id = $1 AND user_id = $2`, storageID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}
