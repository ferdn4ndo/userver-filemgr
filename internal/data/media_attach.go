package data

import (
	"context"

	"github.com/google/uuid"
)

func (d *DB) attachImageMediaDetail(ctx context.Context, storageID uuid.UUID, m *StorageMedia) {
	var img MediaImage
	if err := d.db.GetContext(ctx, &img, `
		SELECT id, created_at, updated_at, focal_length, aperture, flash_fired, iso, orientation_angle, is_flipped,
			exposition, datetime_taken, camera_manufacturer, camera_model, exif_image_height, exif_image_width,
			size_tag, height, width, megapixels, media_id
		FROM core_storagemediaimage WHERE media_id = $1`, m.ID); err != nil {
		return
	}
	m.Image = &img
	var sized []MediaImageSized
	if err := d.db.SelectContext(ctx, &sized, `
		SELECT id, created_at, updated_at, size_tag, height, width, megapixels, media_image_id, storage_file_id
		FROM core_storagemediaimagesized WHERE media_image_id = $1 ORDER BY width DESC NULLS LAST, height DESC NULLS LAST`, img.ID); err != nil {
		return
	}
	for i := range sized {
		tf, err := d.loadFileRow(ctx, storageID, sized[i].StorageFileID, false)
		if err == nil {
			ef2, _ := d.enrichFile(ctx, *tf)
			sized[i].File = ef2
		}
	}
	m.Image.SizedImages = sized
}

func (d *DB) attachVideoMediaDetail(ctx context.Context, m *StorageMedia) {
	var v MediaVideo
	if err := d.db.GetContext(ctx, &v, `
		SELECT id, created_at, updated_at, fps, duration::text, size_tag, height, width, megapixels, media_id
		FROM core_storagemediavideo WHERE media_id = $1`, m.ID); err == nil {
		m.Video = &v
	}
}

func (d *DB) attachDocumentMediaDetail(ctx context.Context, m *StorageMedia) {
	var doc MediaDocument
	if err := d.db.GetContext(ctx, &doc, `
		SELECT id, created_at, updated_at, pages, black_and_white, media_id
		FROM core_storagemediadocument WHERE media_id = $1`, m.ID); err == nil {
		m.Document = &doc
	}
}

func (d *DB) attachThumbnailsForMedia(ctx context.Context, storageID uuid.UUID, m *StorageMedia) {
	var thumbs []MediaThumbRow
	if err := d.db.SelectContext(ctx, &thumbs, `
		SELECT id, created_at, updated_at, size_tag, height, width, megapixels, media_id, storage_file_id
		FROM core_storagemediathumbnail WHERE media_id = $1 ORDER BY size_tag`, m.ID); err != nil {
		return
	}
	for i := range thumbs {
		tf, err := d.loadFileRow(ctx, storageID, thumbs[i].StorageFileID, false)
		if err == nil {
			ef2, _ := d.enrichFile(ctx, *tf)
			thumbs[i].File = ef2
		}
	}
	m.Thumbnails = thumbs
}
