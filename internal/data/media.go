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
	"github.com/jmoiron/sqlx"
)

// StorageMedia is a row in core_storagemedia.
type StorageMedia struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     sql.NullTime    `db:"updated_at" json:"updated_at,omitempty"`
	Title         sql.NullString  `db:"title" json:"title"`
	Type          string          `db:"type" json:"type"`
	Description   sql.NullString  `db:"description" json:"description"`
	StorageFileID uuid.UUID       `db:"storage_file_id" json:"storage_file"`
	CreatedByID   uuid.NullUUID   `db:"created_by_id" json:"-"`
	UpdatedByID   uuid.NullUUID   `db:"updated_by_id" json:"-"`
	File          *StorageFile    `json:"storage_file_detail,omitempty"`
	Image         *MediaImage     `json:"image,omitempty"`
	Video         *MediaVideo     `json:"video,omitempty"`
	Document      *MediaDocument  `json:"document,omitempty"`
	Thumbnails    []MediaThumbRow `json:"thumbnails,omitempty"`
}

// MediaImageSized is a derived JPEG linked from core_storagemediaimagesized (legacy Django image resizer).
type MediaImageSized struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at" json:"updated_at,omitempty"`
	SizeTag       sql.NullString `db:"size_tag" json:"size_tag"`
	Height        sql.NullInt64  `db:"height" json:"height"`
	Width         sql.NullInt64  `db:"width" json:"width"`
	Megapixels    sql.NullString `db:"megapixels" json:"megapixels"`
	MediaImageID  uuid.UUID      `db:"media_image_id" json:"media_image"`
	StorageFileID uuid.UUID      `db:"storage_file_id" json:"storage_file"`
	File          *StorageFile   `json:"storage_file_detail,omitempty"`
}

type MediaImage struct {
	ID                 uuid.UUID            `db:"id" json:"id"`
	CreatedAt          time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt          sql.NullTime         `db:"updated_at" json:"updated_at,omitempty"`
	FocalLength        sql.NullFloat64      `db:"focal_length" json:"focal_length"`
	Aperture           sql.NullString       `db:"aperture" json:"aperture"`
	FlashFired         sql.NullBool         `db:"flash_fired" json:"flash_fired"`
	ISO                sql.NullInt64        `db:"iso" json:"iso"`
	OrientationAngle   sql.NullInt64        `db:"orientation_angle" json:"orientation_angle"`
	IsFlipped          sql.NullBool         `db:"is_flipped" json:"is_flipped"`
	Exposition         sql.NullString       `db:"exposition" json:"exposition"`
	DatetimeTaken      sql.NullString       `db:"datetime_taken" json:"datetime_taken"`
	CameraManufacturer sql.NullString       `db:"camera_manufacturer" json:"camera_manufacturer"`
	CameraModel        sql.NullString       `db:"camera_model" json:"camera_model"`
	ExifImageHeight    sql.NullInt64        `db:"exif_image_height" json:"exif_image_height"`
	ExifImageWidth     sql.NullInt64        `db:"exif_image_width" json:"exif_image_width"`
	SizeTag            sql.NullString       `db:"size_tag" json:"size_tag"`
	Height             sql.NullInt64        `db:"height" json:"height"`
	Width              sql.NullInt64        `db:"width" json:"width"`
	Megapixels         sql.NullString       `db:"megapixels" json:"megapixels"`
	MediaID            uuid.UUID            `db:"media_id" json:"media"`
	SizedImages        []MediaImageSized `json:"sized_images,omitempty" db:"-"`
}

type MediaVideo struct {
	ID         uuid.UUID      `db:"id" json:"id"`
	CreatedAt  time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt  sql.NullTime   `db:"updated_at" json:"updated_at,omitempty"`
	FPS        sql.NullInt64  `db:"fps" json:"fps"`
	Duration   sql.NullString `db:"duration" json:"duration"`
	SizeTag    sql.NullString `db:"size_tag" json:"size_tag"`
	Height     sql.NullInt64  `db:"height" json:"height"`
	Width      sql.NullInt64  `db:"width" json:"width"`
	Megapixels sql.NullString `db:"megapixels" json:"megapixels"`
	MediaID    uuid.UUID      `db:"media_id" json:"media"`
}

type MediaDocument struct {
	ID            uuid.UUID     `db:"id" json:"id"`
	CreatedAt     time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt     sql.NullTime  `db:"updated_at" json:"updated_at,omitempty"`
	Pages         sql.NullInt64 `db:"pages" json:"pages"`
	BlackAndWhite bool          `db:"black_and_white" json:"black_and_white"`
	MediaID       uuid.UUID     `db:"media_id" json:"media"`
}

type MediaThumbRow struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     sql.NullTime   `db:"updated_at" json:"updated_at,omitempty"`
	SizeTag       string         `db:"size_tag" json:"size_tag"`
	Height        sql.NullInt64  `db:"height" json:"height"`
	Width         sql.NullInt64  `db:"width" json:"width"`
	Megapixels    sql.NullString `db:"megapixels" json:"megapixels"`
	MediaID       uuid.UUID      `db:"media_id" json:"media"`
	StorageFileID uuid.UUID      `db:"storage_file_id" json:"storage_file"`
	File          *StorageFile   `json:"storage_file_detail,omitempty"`
}

// ListStorageMedia lists media rows for a storage with the same visibility rules as files.
func (d *DB) ListStorageMedia(ctx context.Context, storageID uuid.UUID, admin bool, userID uuid.UUID, limit, offset int) ([]StorageMedia, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	mayRead, err := d.StorageUserMayRead(ctx, storageID, userID)
	if err != nil {
		return nil, 0, err
	}

	base := `
		FROM core_storagemedia m
		JOIN core_storagefile f ON f.id = m.storage_file_id
		LEFT JOIN core_customuser o ON o.id = f.owner_id
		WHERE f.storage_id = $1 AND f.excluded = false`
	args := []any{storageID}
	argPos := 2
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
	}

	var total int
	qCount := `SELECT COUNT(*) ` + base + visClause
	if err := d.db.GetContext(ctx, &total, qCount, args...); err != nil {
		return nil, 0, err
	}

	rowsSQL := `SELECT m.id, m.created_at, m.updated_at, m.title, m.type, m.description, m.storage_file_id, m.created_by_id, m.updated_by_id ` +
		base + visClause + ` ORDER BY m.created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args2 := append(append([]any{}, args...), limit, offset)

	var raw []StorageMedia
	if err := d.db.SelectContext(ctx, &raw, rowsSQL, args2...); err != nil {
		return nil, 0, err
	}
	out := make([]StorageMedia, 0, len(raw))
	for _, m := range raw {
		f, err := d.loadFileRow(ctx, storageID, m.StorageFileID, false)
		if err != nil {
			continue
		}
		ef, err := d.enrichFile(ctx, *f)
		if err != nil {
			continue
		}
		m.File = ef
		out = append(out, m)
	}
	return out, total, nil
}

// GetStorageMedia returns one media row if visible to the user.
func (d *DB) GetStorageMedia(ctx context.Context, storageID, mediaID uuid.UUID, admin bool, userID uuid.UUID) (*StorageMedia, error) {
	var m StorageMedia
	err := d.db.GetContext(ctx, &m, `
		SELECT m.id, m.created_at, m.updated_at, m.title, m.type, m.description, m.storage_file_id, m.created_by_id, m.updated_by_id
		FROM core_storagemedia m
		JOIN core_storagefile f ON f.id = m.storage_file_id
		WHERE m.id = $1 AND f.storage_id = $2 AND f.excluded = false`, mediaID, storageID)
	if err != nil {
		return nil, err
	}
	f, err := d.loadFileRow(ctx, storageID, m.StorageFileID, false)
	if err != nil {
		return nil, err
	}
	ef, err := d.enrichFile(ctx, *f)
	if err != nil {
		return nil, err
	}
	if !d.fileVisible(ctx, ef, admin, userID) {
		return nil, sql.ErrNoRows
	}
	m.File = ef

	switch strings.ToUpper(m.Type) {
	case "IMAGE":
		d.attachImageMediaDetail(ctx, storageID, &m)
	case "VIDEO":
		d.attachVideoMediaDetail(ctx, &m)
	case "DOCUMENT":
		d.attachDocumentMediaDetail(ctx, &m)
	}
	d.attachThumbnailsForMedia(ctx, storageID, &m)

	return &m, nil
}

// GetFileForWorker loads a file row for background processing (no ACL).
func (d *DB) GetFileForWorker(ctx context.Context, storageID, fileID uuid.UUID) (*StorageFile, error) {
	f, err := d.loadFileRow(ctx, storageID, fileID, false)
	if err != nil {
		return nil, err
	}
	return d.enrichFile(ctx, *f)
}

// StorageMediaExistsForFile returns true if core_storagemedia already exists for this file.
func (d *DB) StorageMediaExistsForFile(ctx context.Context, fileID uuid.UUID) (bool, error) {
	var n int
	err := d.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM core_storagemedia WHERE storage_file_id = $1`, fileID)
	return n > 0, err
}

// GetMimeByMIME looks up a mime row by exact mime_type string.
func (d *DB) GetMimeByMIME(ctx context.Context, mime string) (*MimeType, error) {
	var m MimeType
	err := d.db.GetContext(ctx, &m, `
		SELECT id, mime_type, generic_type, description, extensions FROM core_storagefilemimetype
		WHERE lower(mime_type) = lower($1) LIMIT 1`, mime)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// BeginTx starts a transaction for media writes.
func (d *DB) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return d.db.BeginTxx(ctx, nil)
}

// UpdateFileExifMetadata sets exif_metadata JSON on the parent file.
func (d *DB) UpdateFileExifMetadata(ctx context.Context, fileID uuid.UUID, meta json.RawMessage) error {
	_, err := d.db.ExecContext(ctx, `
		UPDATE core_storagefile SET exif_metadata = $2, updated_at = NOW() WHERE id = $1`, fileID, jsonOrEmpty(meta))
	return err
}

// InsertStorageMediaTx inserts core_storagemedia.
func InsertStorageMediaTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, mediaType string, fileID uuid.UUID, createdBy uuid.NullUUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemedia (id, created_at, type, storage_file_id, created_by_id)
		VALUES ($1, NOW(), $2, $3, $4)`, id, mediaType, fileID, uuidOrNull(createdBy))
	return err
}

// InsertStorageMediaImageTx inserts core_storagemediaimage.
func InsertStorageMediaImageTx(ctx context.Context, tx *sqlx.Tx, img *MediaImage, createdBy uuid.NullUUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemediaimage (
			id, created_at, focal_length, aperture, flash_fired, iso, orientation_angle, is_flipped, exposition,
			datetime_taken, camera_manufacturer, camera_model, exif_image_height, exif_image_width,
			size_tag, height, width, megapixels, media_id, created_by_id
		) VALUES (
			$1, NOW(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		img.ID, nullFloat64(img.FocalLength), nullStr(img.Aperture), nullBool(img.FlashFired), nullInt64(img.ISO),
		nullInt64(img.OrientationAngle), nullBool(img.IsFlipped), nullStr(img.Exposition), nullStr(img.DatetimeTaken),
		nullStr(img.CameraManufacturer), nullStr(img.CameraModel), nullInt64(img.ExifImageHeight), nullInt64(img.ExifImageWidth),
		nullStr(img.SizeTag), nullInt64(img.Height), nullInt64(img.Width), nullNumericString(img.Megapixels),
		img.MediaID, uuidOrNull(createdBy))
	return err
}

// InsertStorageMediaVideoTx inserts core_storagemediavideo.
func InsertStorageMediaVideoTx(ctx context.Context, tx *sqlx.Tx, v *MediaVideo, durationSec float64) error {
	var durArg any
	if durationSec > 0 {
		durArg = fmt.Sprintf("%g seconds", durationSec)
	} else {
		durArg = nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemediavideo (
			id, created_at, fps, duration, size_tag, height, width, megapixels, media_id, created_by_id
		) VALUES (
			$1, NOW(), $2, $3::interval, $4, $5, $6, $7, $8, NULL
		)`,
		v.ID, nullInt64(v.FPS), durArg, nullStr(v.SizeTag), nullInt64(v.Height), nullInt64(v.Width),
		nullNumericString(v.Megapixels), v.MediaID)
	return err
}

// InsertStorageMediaDocumentTx inserts core_storagemediadocument.
func InsertStorageMediaDocumentTx(ctx context.Context, tx *sqlx.Tx, doc *MediaDocument) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemediadocument (id, created_at, pages, black_and_white, media_id, created_by_id)
		VALUES ($1, NOW(), $2, $3, $4, NULL)`,
		doc.ID, nullInt64(doc.Pages), doc.BlackAndWhite, doc.MediaID)
	return err
}

// InsertStorageMediaImageSizedTx links a resized image file to core_storagemediaimage.
func InsertStorageMediaImageSizedTx(ctx context.Context, tx *sqlx.Tx, row *MediaImageSized, createdBy uuid.NullUUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemediaimagesized (
			id, created_at, size_tag, height, width, megapixels, media_image_id, storage_file_id, created_by_id
		) VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7, $8)`,
		row.ID, nullStr(row.SizeTag), nullInt64(row.Height), nullInt64(row.Width), nullNumericString(row.Megapixels),
		row.MediaImageID, row.StorageFileID, uuidOrNull(createdBy))
	return err
}

// InsertStorageMediaThumbnailTx links a thumbnail file to media.
func InsertStorageMediaThumbnailTx(ctx context.Context, tx *sqlx.Tx, t *MediaThumbRow, createdBy uuid.NullUUID) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO core_storagemediathumbnail (
			id, created_at, size_tag, height, width, megapixels, media_id, storage_file_id, created_by_id
		) VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.SizeTag, nullInt64(t.Height), nullInt64(t.Width), nullNumericString(t.Megapixels),
		t.MediaID, t.StorageFileID, uuidOrNull(createdBy))
	return err
}

// InsertFileTx inserts core_storagefile inside a transaction.
func (d *DB) InsertFileTx(ctx context.Context, tx *sqlx.Tx, f *StorageFile) error {
	_, err := tx.ExecContext(ctx, `
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

func nullFloat64(n sql.NullFloat64) any {
	if !n.Valid {
		return nil
	}
	return n.Float64
}

func nullBool(n sql.NullBool) any {
	if !n.Valid {
		return nil
	}
	return n.Bool
}

func nullInt64(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func nullNumericString(n sql.NullString) any {
	if !n.Valid {
		return nil
	}
	return n.String
}
