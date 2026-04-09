package media_processor

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/object_store"
	"github.com/ferdn4ndo/userver-filemgr/lib"
)

var thumbSpecs = []struct {
	Tag string
	W   int
	H   int
}{
	{"SIZE_THUMB_SMALL", 240, 180},
	{"SIZE_THUMB_MEDIUM", 480, 360},
	{"SIZE_THUMB_LARGE", 960, 720},
}

// Processor generates storagemedia rows, thumbnails, and optional video metadata.
type Processor struct {
	env     lib.Env
	db      *data.DB
	objects *object_store.Factory
}

// NewProcessor builds a processor (worker-safe).
func NewProcessor(env lib.Env, db *data.DB, ob *object_store.Factory, _ lib.Logger) *Processor {
	return &Processor{env: env, db: db, objects: ob}
}

// MediaCategory returns IMAGE, VIDEO, DOCUMENT, or empty.
func MediaCategory(f *data.StorageFile) string {
	if f == nil || f.Type == nil {
		return ""
	}
	g := strings.ToUpper(strings.TrimSpace(f.Type.GenericType.String))
	if g == "IMAGE" || g == "VIDEO" {
		return g
	}
	mt := strings.ToLower(f.Type.MimeType)
	if strings.HasPrefix(mt, "image/") {
		return "IMAGE"
	}
	if strings.HasPrefix(mt, "video/") {
		return "VIDEO"
	}
	if mt == "application/pdf" || strings.Contains(mt, "pdf") {
		return "DOCUMENT"
	}
	if g == "APPLICATION" && strings.Contains(mt, "pdf") {
		return "DOCUMENT"
	}
	return ""
}

// ShouldEnqueue returns true when this file may produce media metadata.
func ShouldEnqueue(f *data.StorageFile) bool {
	return MediaCategory(f) != ""
}

// Process runs the pipeline for one job (idempotent if media already exists).
func (p *Processor) Process(ctx context.Context, job *data.MediaJob) error {
	exists, err := p.db.StorageMediaExistsForFile(ctx, job.StorageFileID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	f, err := p.db.GetFileForWorker(ctx, job.StorageID, job.StorageFileID)
	if err != nil {
		return err
	}
	if f.Status != "PUBLISHED" || !f.RealPath.Valid {
		return nil
	}
	st, err := p.db.GetStorage(ctx, job.StorageID)
	if err != nil {
		return err
	}
	stType := st.Type.String
	if !st.Type.Valid {
		stType = ""
	}
	switch MediaCategory(f) {
	case "IMAGE":
		return p.processImage(ctx, stType, st.Credentials, f)
	case "VIDEO":
		return p.processVideo(ctx, stType, st.Credentials, f)
	case "DOCUMENT":
		return p.processDocument(ctx, f)
	default:
		return nil
	}
}

func (p *Processor) processImage(ctx context.Context, stType string, creds []byte, f *data.StorageFile) error {
	be, err := p.objects.ForStorage(stType, creds)
	if err != nil {
		return err
	}
	rc, err := be.Open(ctx, f.RealPath.String)
	if err != nil {
		return err
	}
	defer rc.Close()
	buf, err := io.ReadAll(io.LimitReader(rc, p.env.MediaMaxImageBytes))
	if err != nil {
		return err
	}
	if int64(len(buf)) >= p.env.MediaMaxImageBytes {
		return fmt.Errorf("image exceeds MEDIA_MAX_IMAGE_BYTES")
	}
	var exifJSON json.RawMessage
	x, xerr := exif.Decode(bytes.NewReader(buf))
	if xerr == nil {
		exifJSON, _ = exifToJSON(x)
	} else {
		x = nil
	}
	if len(exifJSON) == 0 {
		exifJSON = json.RawMessage(`{}`)
	}
	img, err := imaging.Decode(bytes.NewReader(buf), imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	mp := float64(w*h) / 1e6

	jpegMime, _ := p.db.GetMimeByMIME(ctx, "image/jpeg")
	var jpegType uuid.NullUUID
	if jpegMime != nil {
		jpegType = uuid.NullUUID{UUID: jpegMime.ID, Valid: true}
	}

	mediaID := uuid.New()
	imageRowID := uuid.New()
	imgRow := &data.MediaImage{
		ID:         imageRowID,
		MediaID:    mediaID,
		SizeTag:    sql.NullString{String: "SIZE_ORIGINAL", Valid: true},
		Height:     sql.NullInt64{Int64: int64(h), Valid: true},
		Width:      sql.NullInt64{Int64: int64(w), Valid: true},
		Megapixels: sql.NullString{String: fmt.Sprintf("%.4f", mp), Valid: true},
	}
	fillEXIF(imgRow, x, w, h)

	tx, err := p.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := data.InsertStorageMediaTx(ctx, tx, mediaID, "IMAGE", f.ID, f.CreatedByID); err != nil {
		return err
	}
	if err := data.InsertStorageMediaImageTx(ctx, tx, imgRow, f.CreatedByID); err != nil {
		return err
	}

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

	if err := tx.Commit(); err != nil {
		return err
	}
	_ = p.db.UpdateFileExifMetadata(ctx, f.ID, exifJSON)
	return nil
}

func (p *Processor) processVideo(ctx context.Context, stType string, creds []byte, f *data.StorageFile) error {
	mediaID := uuid.New()
	videoID := uuid.New()
	var durSec float64
	var vw, vh int
	var fps sql.NullInt64

	if p.env.FFprobePath != "" && f.Size > 0 && f.Size <= p.env.MediaMaxVideoProbeBytes {
		be, err := p.objects.ForStorage(stType, creds)
		if err != nil {
			return err
		}
		rc, err := be.Open(ctx, f.RealPath.String)
		if err != nil {
			return err
		}
		tmp, err := os.CreateTemp("", "ffprobe-*")
		if err != nil {
			rc.Close()
			return err
		}
		tmpPath := tmp.Name()
		_, copyErr := io.Copy(tmp, io.LimitReader(rc, p.env.MediaMaxVideoProbeBytes))
		rc.Close()
		tmp.Close()
		if copyErr != nil {
			_ = os.Remove(tmpPath)
			return copyErr
		}
		durSec, vw, vh, fps = ffprobeMeta(ctx, p.env.FFprobePath, tmpPath)
		_ = os.Remove(tmpPath)
	}

	tx, err := p.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := data.InsertStorageMediaTx(ctx, tx, mediaID, "VIDEO", f.ID, f.CreatedByID); err != nil {
		return err
	}
	vrow := &data.MediaVideo{
		ID:      videoID,
		MediaID: mediaID,
		SizeTag: sql.NullString{String: "SIZE_ORIGINAL", Valid: true},
		Height:  sql.NullInt64{Int64: int64(vh), Valid: vh > 0},
		Width:   sql.NullInt64{Int64: int64(vw), Valid: vw > 0},
	}
	if vw > 0 && vh > 0 {
		vrow.Megapixels = sql.NullString{String: fmt.Sprintf("%.4f", float64(vw*vh)/1e6), Valid: true}
	}
	if fps.Valid {
		vrow.FPS = fps
	}
	if err := data.InsertStorageMediaVideoTx(ctx, tx, vrow, durSec); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Processor) processDocument(ctx context.Context, f *data.StorageFile) error {
	mediaID := uuid.New()
	docID := uuid.New()
	tx, err := p.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := data.InsertStorageMediaTx(ctx, tx, mediaID, "DOCUMENT", f.ID, f.CreatedByID); err != nil {
		return err
	}
	doc := &data.MediaDocument{
		ID:            docID,
		MediaID:       mediaID,
		BlackAndWhite: false,
	}
	if err := data.InsertStorageMediaDocumentTx(ctx, tx, doc); err != nil {
		return err
	}
	return tx.Commit()
}

type exifWalkFunc func(name exif.FieldName, tag *tiff.Tag) error

func (f exifWalkFunc) Walk(name exif.FieldName, tag *tiff.Tag) error { return f(name, tag) }

func exifToJSON(x *exif.Exif) (json.RawMessage, error) {
	m := map[string]any{}
	_ = x.Walk(exifWalkFunc(func(name exif.FieldName, tag *tiff.Tag) error {
		s, err := tag.StringVal()
		if err != nil {
			return nil
		}
		m[string(name)] = s
		return nil
	}))
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func fillEXIF(img *data.MediaImage, x *exif.Exif, w, h int) {
	img.ExifImageWidth = sql.NullInt64{Int64: int64(w), Valid: true}
	img.ExifImageHeight = sql.NullInt64{Int64: int64(h), Valid: true}
	if x == nil {
		return
	}
	if t, err := x.Get(exif.PixelXDimension); err == nil {
		if v, err := t.Int(0); err == nil {
			img.ExifImageWidth = sql.NullInt64{Int64: int64(v), Valid: true}
		}
	}
	if t, err := x.Get(exif.PixelYDimension); err == nil {
		if v, err := t.Int(0); err == nil {
			img.ExifImageHeight = sql.NullInt64{Int64: int64(v), Valid: true}
		}
	}
	img.CameraManufacturer = exifString(x, exif.Make)
	img.CameraModel = exifString(x, exif.Model)
	img.DatetimeTaken = firstNonEmpty(exifString(x, exif.DateTimeOriginal), exifString(x, exif.DateTime))
	img.Exposition = exifString(x, exif.ExposureTime)
	img.Aperture = exifString(x, exif.FNumber)
	if iso := exifInt(x, exif.ISOSpeedRatings); iso.Valid {
		img.ISO = iso
	}
	if o := exifInt(x, exif.Orientation); o.Valid {
		img.OrientationAngle = o
	}
	if fl := exifFloat(x, exif.FocalLength); fl.Valid {
		img.FocalLength = fl
	}
	if flash := exifInt(x, exif.Flash); flash.Valid {
		img.FlashFired = sql.NullBool{Bool: (flash.Int64 & 1) != 0, Valid: true}
	}
}

func firstNonEmpty(a, b sql.NullString) sql.NullString {
	if a.Valid && a.String != "" {
		return a
	}
	return b
}

func exifString(x *exif.Exif, n exif.FieldName) sql.NullString {
	t, err := x.Get(n)
	if err != nil {
		return sql.NullString{}
	}
	s, err := t.StringVal()
	if err != nil || s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func exifInt(x *exif.Exif, n exif.FieldName) sql.NullInt64 {
	t, err := x.Get(n)
	if err != nil {
		return sql.NullInt64{}
	}
	v, err := t.Int(0)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(v), Valid: true}
}

func exifFloat(x *exif.Exif, n exif.FieldName) sql.NullFloat64 {
	t, err := x.Get(n)
	if err != nil {
		return sql.NullFloat64{}
	}
	num, den, err := t.Rat2(0)
	if err != nil || den == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: float64(num) / float64(den), Valid: true}
}

func ffprobeMeta(ctx context.Context, bin, filePath string) (dur float64, w, h int, fps sql.NullInt64) {
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	// Larger probe windows help when the file on disk is a truncated copy (first N bytes only).
	cmd := exec.CommandContext(cctx, bin,
		"-v", "quiet",
		"-probesize", "50M",
		"-analyzeduration", "100M",
		"-print_format", "json", "-show_format", "-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, sql.NullInt64{}
	}
	var payload struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType    string `json:"codec_type"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			AvgFrameRate string `json:"avg_frame_rate"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return 0, 0, 0, sql.NullInt64{}
	}
	if payload.Format.Duration != "" {
		dur, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	}
	for _, s := range payload.Streams {
		if s.CodecType != "video" {
			continue
		}
		if s.Width > 0 {
			w = s.Width
		}
		if s.Height > 0 {
			h = s.Height
		}
		if s.AvgFrameRate != "" && s.AvgFrameRate != "0/0" {
			parts := strings.Split(s.AvgFrameRate, "/")
			if len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den != 0 {
					fps = sql.NullInt64{Int64: int64(num/den + 0.5), Valid: true}
				}
			}
		}
		break
	}
	return dur, w, h, fps
}

func randomSig() string {
	b := make([]byte, 48)
	_, _ = rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 64 {
		return s[:64]
	}
	return s
}
