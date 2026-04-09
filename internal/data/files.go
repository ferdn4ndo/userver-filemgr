package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MimeType struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	MimeType    string         `db:"mime_type" json:"mime_type"`
	GenericType sql.NullString `db:"generic_type" json:"generic_type"`
	Description sql.NullString `db:"description" json:"description"`
	Extensions  sql.NullString `db:"extensions" json:"extensions"`
}

type StorageFile struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      *time.Time      `db:"updated_at" json:"updated_at,omitempty"`
	SignatureKey   string          `db:"signature_key" json:"signature_key"`
	StorageID      uuid.UUID       `db:"storage_id" json:"-"`
	OwnerID        uuid.NullUUID   `db:"owner_id" json:"-"`
	Name           sql.NullString  `db:"name" json:"name"`
	Status         string          `db:"status" json:"status"`
	Visibility     string          `db:"visibility" json:"visibility"`
	Size           int64           `db:"size" json:"size"`
	Hash           sql.NullString  `db:"hash" json:"hash"`
	TypeID         uuid.NullUUID   `db:"type_id" json:"-"`
	Extension      sql.NullString  `db:"extension" json:"extension"`
	ExifMetadata   json.RawMessage `db:"exif_metadata" json:"exif_metadata"`
	CustomMetadata json.RawMessage `db:"custom_metadata" json:"custom_metadata"`
	Origin         string          `db:"origin" json:"origin"`
	OriginalPath   sql.NullString  `db:"original_path" json:"original_path"`
	RealPath       sql.NullString  `db:"real_path" json:"real_path"`
	VirtualPath    sql.NullString  `db:"virtual_path" json:"virtual_path"`
	Available      bool            `db:"available" json:"available"`
	Excluded       bool            `db:"excluded" json:"excluded"`
	CreatedByID    uuid.NullUUID   `db:"created_by_id" json:"-"`
	UpdatedByID    uuid.NullUUID   `db:"updated_by_id" json:"-"`
	Owner          *UserPublic     `json:"owner,omitempty"`
	Type           *MimeType       `json:"type,omitempty"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedBy      *uuid.UUID      `json:"updated_by,omitempty"`
}

func (d *DB) FindMimeByExtension(ctx context.Context, ext string) (*MimeType, error) {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
	if ext == "" {
		return nil, sql.ErrNoRows
	}
	pat := "%" + ext + "%"
	var m MimeType
	err := d.db.GetContext(ctx, &m, `
		SELECT id, mime_type, generic_type, description, extensions FROM core_storagefilemimetype
		WHERE extensions IS NOT NULL AND lower(extensions) LIKE $1 LIMIT 1`, pat)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *DB) ListFiles(ctx context.Context, storageID uuid.UUID, excluded bool, admin bool, userID uuid.UUID, limit, offset int) ([]StorageFile, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	mayRead, _ := d.StorageUserMayRead(ctx, storageID, userID)

	base := `
		FROM core_storagefile f
		LEFT JOIN core_customuser o ON o.id = f.owner_id
		WHERE f.storage_id = $1 AND f.excluded = $2`
	args := []any{storageID, excluded}
	argPos := 3
	var visClause string
	if !admin {
		if !mayRead {
			visClause = ` AND false `
		} else {
			visClause = fmt.Sprintf(` AND (
				f.visibility = 'PUBLIC' OR
				(f.visibility = 'SYSTEM' AND o.system_name IS DISTINCT FROM (SELECT system_name FROM core_customuser WHERE id = $%d)) OR
				(f.visibility = 'USER' AND f.owner_id = $%d)
			)`, argPos, argPos)
			args = append(args, userID)
		}
	} else {
		visClause = ""
	}

	var total int
	qCount := `SELECT COUNT(*) ` + base + visClause
	if err := d.db.GetContext(ctx, &total, qCount, args...); err != nil {
		return nil, 0, err
	}

	rowsSQL := `SELECT f.id, f.created_at, f.updated_at, f.signature_key, f.storage_id, f.owner_id, f.name, f.status, f.visibility,
		f.size, f.hash, f.type_id, f.extension, f.exif_metadata, f.custom_metadata, f.origin, f.original_path, f.real_path, f.virtual_path,
		f.available, f.excluded, f.created_by_id, f.updated_by_id ` + base + visClause + ` ORDER BY f.created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args2 := append(append([]any{}, args...), limit, offset)
	var raw []StorageFile
	if err := d.db.SelectContext(ctx, &raw, rowsSQL, args2...); err != nil {
		return nil, 0, err
	}
	out := make([]StorageFile, 0, len(raw))
	for _, f := range raw {
		ef, err := d.enrichFile(ctx, f)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *ef)
	}
	return out, total, nil
}

func (d *DB) loadFileRow(ctx context.Context, storageID, fileID uuid.UUID, excluded bool) (*StorageFile, error) {
	var f StorageFile
	err := d.db.GetContext(ctx, &f, `
		SELECT id, created_at, updated_at, signature_key, storage_id, owner_id, name, status, visibility,
			size, hash, type_id, extension, exif_metadata, custom_metadata, origin, original_path, real_path, virtual_path,
			available, excluded, created_by_id, updated_by_id
		FROM core_storagefile WHERE storage_id = $1 AND id = $2 AND excluded = $3`, storageID, fileID, excluded)
	if err != nil {
		return nil, err
	}
	return d.enrichFile(ctx, f)
}

func (d *DB) enrichFile(ctx context.Context, f StorageFile) (*StorageFile, error) {
	if f.OwnerID.Valid {
		if up, err := d.UserPublic(ctx, f.OwnerID.UUID); err == nil {
			f.Owner = up
		}
	}
	if f.TypeID.Valid {
		var mt MimeType
		if err := d.db.GetContext(ctx, &mt, `SELECT id, mime_type, generic_type, description, extensions FROM core_storagefilemimetype WHERE id = $1`, f.TypeID.UUID); err == nil {
			f.Type = &mt
		}
	}
	if f.CreatedByID.Valid {
		u := f.CreatedByID.UUID
		f.CreatedBy = &u
	}
	if f.UpdatedByID.Valid {
		u := f.UpdatedByID.UUID
		f.UpdatedBy = &u
	}
	return &f, nil
}

func (d *DB) fileVisible(ctx context.Context, f *StorageFile, admin bool, userID uuid.UUID) bool {
	if admin {
		return true
	}
	ok, _ := d.StorageUserMayRead(ctx, f.StorageID, userID)
	if !ok {
		return false
	}
	switch f.Visibility {
	case "PUBLIC":
		return true
	case "USER":
		return f.OwnerID.Valid && f.OwnerID.UUID == userID
	case "SYSTEM":
		// Matches historical is_visible_by_user: SYSTEM visible when owner system_name != viewer system_name.
		var usys, osys string
		_ = d.db.GetContext(ctx, &usys, `SELECT system_name FROM core_customuser WHERE id = $1`, userID)
		if !f.OwnerID.Valid {
			return false
		}
		_ = d.db.GetContext(ctx, &osys, `SELECT system_name FROM core_customuser WHERE id = $1`, f.OwnerID.UUID)
		return usys != osys
	default:
		return false
	}
}

func (d *DB) GetFile(ctx context.Context, storageID, fileID uuid.UUID, excluded bool, admin bool, userID uuid.UUID) (*StorageFile, error) {
	f, err := d.loadFileRow(ctx, storageID, fileID, excluded)
	if err != nil {
		return nil, err
	}
	if !d.fileVisible(ctx, f, admin, userID) {
		return nil, sql.ErrNoRows
	}
	return f, nil
}

func (d *DB) InsertFile(ctx context.Context, f *StorageFile) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO core_storagefile (
			id, created_at, updated_at, signature_key, storage_id, owner_id, name, status, visibility, size, hash, type_id, extension,
			exif_metadata, custom_metadata, origin, original_path, real_path, virtual_path, available, excluded,
			created_by_id, updated_by_id
		) VALUES (
			$1, NOW(), NOW(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)`,
		f.ID, f.SignatureKey, f.StorageID, uuidOrNull(f.OwnerID), nullStr(f.Name), f.Status, f.Visibility, f.Size, nullStr(f.Hash), uuidOrNull(f.TypeID), nullStr(f.Extension),
		jsonOrEmpty(f.ExifMetadata), jsonOrEmpty(f.CustomMetadata), f.Origin, nullStr(f.OriginalPath), nullStr(f.RealPath), nullStr(f.VirtualPath),
		f.Available, f.Excluded, uuidOrNull(f.CreatedByID), uuidOrNull(f.UpdatedByID))
	return err
}

func (d *DB) UpdateFileMeta(ctx context.Context, storageID, fileID uuid.UUID, name, visibility *string, customMeta json.RawMessage, virtualPath *string, actor uuid.UUID) error {
	f, err := d.loadFileRow(ctx, storageID, fileID, false)
	if err != nil {
		return err
	}
	if name != nil {
		f.Name = sql.NullString{String: *name, Valid: true}
	}
	if visibility != nil {
		f.Visibility = *visibility
	}
	if len(customMeta) > 0 {
		f.CustomMetadata = customMeta
	}
	if virtualPath != nil {
		f.VirtualPath = sql.NullString{String: *virtualPath, Valid: true}
	}
	f.UpdatedByID = uuid.NullUUID{UUID: actor, Valid: true}
	_, err = d.db.ExecContext(ctx, `
		UPDATE core_storagefile SET name = $3, visibility = $4, custom_metadata = $5, virtual_path = $6, updated_at = NOW(), updated_by_id = $7
		WHERE storage_id = $1 AND id = $2 AND excluded = false`,
		storageID, fileID, nullStr(f.Name), f.Visibility, jsonOrEmpty(f.CustomMetadata), nullStr(f.VirtualPath), actor)
	return err
}

func (d *DB) SetFileExcluded(ctx context.Context, storageID, fileID uuid.UUID, excluded bool, actor uuid.UUID) error {
	res, err := d.db.ExecContext(ctx, `
		UPDATE core_storagefile SET excluded = $4, updated_at = NOW(), updated_by_id = $5 WHERE storage_id = $1 AND id = $2 AND excluded = $3`,
		storageID, fileID, !excluded, excluded, actor)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *DB) UpdateFilePathsAndStatus(ctx context.Context, fileID uuid.UUID, status string, size int64, realPath, virtualPath, hash string, mimeID uuid.NullUUID) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE core_storagefile SET status = $2, size = $3, real_path = $4, virtual_path = $5, hash = NULLIF($6,''), type_id = $7, updated_at = NOW()
		WHERE id = $1`,
		fileID, status, size, nullStringPtr(realPath), nullStringPtr(virtualPath), hash, uuidOrNull(mimeID))
	return err
}

func nullStr(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}

func nullStringPtr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func uuidOrNull(u uuid.NullUUID) any {
	if !u.Valid {
		return nil
	}
	return u.UUID
}

func jsonOrEmpty(j json.RawMessage) any {
	if len(j) == 0 {
		return []byte("{}")
	}
	return j
}

type FileDownload struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	DownloadURL   sql.NullString `db:"download_url" json:"download_url"`
	ForceDownload bool           `db:"force_download" json:"force_download"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	ExpiresAt     time.Time      `db:"expires_at" json:"expires_at"`
	OwnerID       uuid.NullUUID  `db:"owner_id" json:"-"`
	StorageFileID uuid.UUID      `db:"storage_file_id" json:"storage_file"`
}

func (d *DB) InsertDownload(ctx context.Context, row *FileDownload) error {
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO core_storagefiledownload (id, download_url, force_download, created_at, expires_at, owner_id, storage_file_id)
		VALUES ($1, $2, $3, NOW(), $4, $5, $6)`,
		row.ID, nullStr(row.DownloadURL), row.ForceDownload, row.ExpiresAt, uuidOrNull(row.OwnerID), row.StorageFileID)
	return err
}

func (d *DB) UpdateDownloadURL(ctx context.Context, id uuid.UUID, url string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE core_storagefiledownload SET download_url = $2 WHERE id = $1`, id, url)
	return err
}

func (d *DB) GetValidDownload(ctx context.Context, id uuid.UUID, owner uuid.UUID) (*FileDownload, error) {
	var r FileDownload
	err := d.db.GetContext(ctx, &r, `
		SELECT id, download_url, force_download, created_at, expires_at, owner_id, storage_file_id
		FROM core_storagefiledownload WHERE id = $1 AND owner_id = $2 AND expires_at > NOW()`, id, owner)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) GetDownloadByID(ctx context.Context, id uuid.UUID) (*FileDownload, error) {
	var r FileDownload
	err := d.db.GetContext(ctx, &r, `
		SELECT id, download_url, force_download, created_at, expires_at, owner_id, storage_file_id
		FROM core_storagefiledownload WHERE id = $1 AND expires_at > NOW()`, id)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *DB) PermaDeleteFile(ctx context.Context, storageID, fileID uuid.UUID) error {
	res, err := d.db.ExecContext(ctx, `DELETE FROM core_storagefile WHERE storage_id = $1 AND id = $2 AND excluded = true`, storageID, fileID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
