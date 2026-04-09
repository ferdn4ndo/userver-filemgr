package media_processor

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/object_store"
)

func (p *Processor) insertSizedImageRows(ctx context.Context, tx *sqlx.Tx, be object_store.Backend, img image.Image, f *data.StorageFile, sizes []imageBox, imageRowID uuid.UUID, jpegType uuid.NullUUID, ow, oh int) error {
	for _, box := range sizes {
		rw, rh := computeNewImageDimensions(ow, oh, box.W, box.H)
		if ow < rw || oh < rh {
			continue
		}
		resized := imaging.Resize(img, rw, rh, imaging.Lanczos)
		var bout bytes.Buffer
		if err := imaging.Encode(&bout, resized, imaging.JPEG, imaging.JPEGQuality(jpegQualityResized)); err != nil {
			return err
		}
		tid := uuid.New()
		sizeTag := sizeTagFromDimensions(rw, rh)
		vpath := "/" + tid.String() + "/resized_" + strings.ToLower(sizeTag) + ".jpg"
		real := strings.TrimPrefix(vpath, "/")
		sfile := &data.StorageFile{
			ID:             tid,
			SignatureKey:   randomSig(),
			StorageID:      f.StorageID,
			OwnerID:        f.OwnerID,
			Name:           sql.NullString{String: sizeTag + "_resized.jpg", Valid: true},
			Status:         "PUBLISHED",
			Visibility:     f.Visibility,
			Size:           int64(bout.Len()),
			Extension:      sql.NullString{String: "jpg", Valid: true},
			Origin:         "DERIVED",
			VirtualPath:    sql.NullString{String: vpath, Valid: true},
			RealPath:       sql.NullString{String: real, Valid: true},
			Available:      true,
			Excluded:       false,
			CreatedByID:    f.CreatedByID,
			TypeID:         jpegType,
			ExifMetadata:   f.ExifMetadata,
			CustomMetadata: f.CustomMetadata,
		}
		if err := p.db.InsertFileTx(ctx, tx, sfile); err != nil {
			return err
		}
		if err := be.Put(ctx, real, bytes.NewReader(bout.Bytes()), int64(bout.Len()), "image/jpeg"); err != nil {
			return err
		}
		szRow := &data.MediaImageSized{
			ID:            uuid.New(),
			SizeTag:       sql.NullString{String: sizeTag, Valid: true},
			Height:        sql.NullInt64{Int64: int64(rh), Valid: true},
			Width:         sql.NullInt64{Int64: int64(rw), Valid: true},
			Megapixels:    sql.NullString{String: fmt.Sprintf("%.4f", float64(rw*rh)/1e6), Valid: true},
			MediaImageID:  imageRowID,
			StorageFileID: tid,
		}
		if err := data.InsertStorageMediaImageSizedTx(ctx, tx, szRow, f.CreatedByID); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) insertThumbnailRows(ctx context.Context, tx *sqlx.Tx, be object_store.Backend, img image.Image, f *data.StorageFile, mediaID uuid.UUID, jpegType uuid.NullUUID) error {
	for _, spec := range thumbSpecs {
		thumb := imaging.Fit(img, spec.W, spec.H, imaging.Lanczos)
		var out bytes.Buffer
		if err := imaging.Encode(&out, thumb, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
			return err
		}
		tid := uuid.New()
		vpath := "/" + tid.String() + "/" + spec.Tag + ".jpg"
		real := strings.TrimPrefix(vpath, "/")
		tfile := &data.StorageFile{
			ID:           tid,
			SignatureKey: randomSig(),
			StorageID:    f.StorageID,
			OwnerID:      f.OwnerID,
			Name:         sql.NullString{String: spec.Tag + ".jpg", Valid: true},
			Status:       "PUBLISHED",
			Visibility:   f.Visibility,
			Size:         int64(out.Len()),
			Extension:    sql.NullString{String: "jpg", Valid: true},
			Origin:       "DERIVED",
			VirtualPath:  sql.NullString{String: vpath, Valid: true},
			RealPath:     sql.NullString{String: real, Valid: true},
			Available:    true,
			Excluded:     false,
			CreatedByID:  f.CreatedByID,
			TypeID:       jpegType,
		}
		if err := p.db.InsertFileTx(ctx, tx, tfile); err != nil {
			return err
		}
		if err := be.Put(ctx, real, bytes.NewReader(out.Bytes()), int64(out.Len()), "image/jpeg"); err != nil {
			return err
		}
		tb := thumb.Bounds()
		tw, th2 := tb.Dx(), tb.Dy()
		tmp := float64(tw*th2) / 1e6
		thRow := &data.MediaThumbRow{
			ID:            uuid.New(),
			SizeTag:       spec.Tag,
			Height:        sql.NullInt64{Int64: int64(th2), Valid: true},
			Width:         sql.NullInt64{Int64: int64(tw), Valid: true},
			Megapixels:    sql.NullString{String: fmt.Sprintf("%.4f", tmp), Valid: true},
			MediaID:       mediaID,
			StorageFileID: tid,
		}
		if err := data.InsertStorageMediaThumbnailTx(ctx, tx, thRow, f.CreatedByID); err != nil {
			return err
		}
	}
	return nil
}
