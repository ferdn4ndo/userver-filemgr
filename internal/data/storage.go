package data

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

type Storage struct {
	ID                        uuid.UUID       `db:"id" json:"id"`
	Type                      sql.NullString  `db:"type" json:"type"`
	Credentials               json.RawMessage `db:"credentials" json:"-"`
	MediaConvertConfiguration json.RawMessage `db:"media_convert_configuration" json:"media_convert_configuration,omitempty"`
	TotalSize                 int64           `db:"total_size" json:"total_size"`
}

func (d *DB) ListStorages(ctx context.Context, limit, offset int) ([]Storage, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := d.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM core_storage`); err != nil {
		return nil, 0, err
	}
	var rows []Storage
	err := d.db.SelectContext(ctx, &rows, `
		SELECT s.id, s.type, s.credentials, s.media_convert_configuration,
			COALESCE(SUM(f.size) FILTER (WHERE f.excluded = false), 0)::bigint AS total_size
		FROM core_storage s
		LEFT JOIN core_storagefile f ON f.storage_id = s.id
		GROUP BY s.id, s.type, s.credentials, s.media_convert_configuration
		ORDER BY s.id
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (d *DB) GetStorage(ctx context.Context, id uuid.UUID) (*Storage, error) {
	var r Storage
	err := d.db.GetContext(ctx, &r, `
		SELECT s.id, s.type, s.credentials, s.media_convert_configuration,
			COALESCE((SELECT SUM(size) FROM core_storagefile f WHERE f.storage_id = s.id AND f.excluded = false), 0)::bigint AS total_size
		FROM core_storage s WHERE s.id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) InsertStorage(ctx context.Context, stType string, creds, media json.RawMessage) (*Storage, error) {
	id := uuid.New()
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO core_storage (id, type, credentials, media_convert_configuration) VALUES ($1, $2, $3, $4)`,
		id, nullString(stType), creds, media)
	if err != nil {
		return nil, err
	}
	return d.GetStorage(ctx, id)
}

func (d *DB) UpdateStorage(ctx context.Context, id uuid.UUID, stType string, creds, media json.RawMessage) (*Storage, error) {
	var mediaArg any = nil
	if len(media) > 0 && string(media) != "null" {
		mediaArg = media
	}
	_, err := d.db.ExecContext(ctx, `
		UPDATE core_storage SET type = $2, credentials = $3, media_convert_configuration = COALESCE($4, media_convert_configuration)
		WHERE id = $1`,
		id, nullString(stType), creds, mediaArg)
	if err != nil {
		return nil, err
	}
	return d.GetStorage(ctx, id)
}

func (d *DB) DeleteStorage(ctx context.Context, id uuid.UUID) error {
	res, err := d.db.ExecContext(ctx, `DELETE FROM core_storage WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
