package http_api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ferdn4ndo/userver-filemgr/internal/auth"
	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/urlfetch"
)

func (r *Router) streamDownloadParseIDs(c *gin.Context) (did, sid, fid uuid.UUID, ok bool) {
	var err error
	did, err = uuid.Parse(c.Param("downloadID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid download id"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	sid, err = uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	fid, err = uuid.Parse(c.Param("fileID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid file"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return did, sid, fid, true
}

func (r *Router) streamDownloadLoadContext(c *gin.Context, u *auth.User, did, sid, fid uuid.UUID) (*data.FileDownload, *data.StorageFile, *data.Storage, bool) {
	dl, err := r.db.GetValidDownload(c.Request.Context(), did, u.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return nil, nil, nil, false
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	if err != nil || f.ID != dl.StorageFileID {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return nil, nil, nil, false
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return nil, nil, nil, false
	}
	return dl, f, st, true
}

func redirectIfExternalDownloadURL(c *gin.Context, dl *data.FileDownload) bool {
	if !dl.DownloadURL.Valid {
		return false
	}
	u := dl.DownloadURL.String
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return false
	}
	c.Redirect(http.StatusFound, u)
	return true
}

func streamDownloadAttachmentName(f *data.StorageFile) string {
	if f.Name.Valid && f.Name.String != "" {
		return f.Name.String
	}
	return "file"
}

func extensionFromSourceURL(sourceURL string) string {
	ext := "bin"
	if i := strings.LastIndex(sourceURL, "."); i >= 0 {
		ext = strings.ToLower(strings.TrimSpace(sourceURL[i+1:]))
		if len(ext) > 16 {
			ext = ext[:16]
		}
	}
	return ext
}

func (r *Router) newUploadingFileFromURL(c *gin.Context, u *auth.User, sid uuid.UUID, sourceURL string, ext string) (*data.StorageFile, bool) {
	id := uuid.New()
	sig := randomSig()
	vpath := "/" + id.String() + "/file"
	real := strings.TrimPrefix(vpath, "/")
	frow := &data.StorageFile{
		ID: id, SignatureKey: sig, StorageID: sid,
		OwnerID: uuid.NullUUID{UUID: u.ID, Valid: true},
		Status:  "UPLOADING", Visibility: "SYSTEM", Size: 0,
		Origin: "WEB", OriginalPath: sql.NullString{String: sourceURL, Valid: true},
		VirtualPath: sql.NullString{String: vpath, Valid: true},
		RealPath:    sql.NullString{String: real, Valid: true},
		CreatedByID: uuid.NullUUID{UUID: u.ID, Valid: true},
	}
	if mt, err := r.db.FindMimeByExtension(c.Request.Context(), ext); err == nil {
		frow.TypeID = uuid.NullUUID{UUID: mt.ID, Valid: true}
	}
	frow.Extension = sql.NullString{String: ext, Valid: true}
	if err := r.db.InsertFile(c.Request.Context(), frow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return nil, false
	}
	return frow, true
}

// putStreamURLUpload streams resp.Body into object storage (no full in-memory buffer) and publishes the file row.
func (r *Router) putStreamURLUpload(c *gin.Context, st *data.Storage, frow *data.StorageFile, resp *http.Response, sid uuid.UUID, u *auth.User) bool {
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch url"})
		return false
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	be, err := r.objects.ForStorage(st.Type.String, st.Credentials)
	if err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return false
	}
	real := frow.RealPath.String
	vpath := frow.VirtualPath.String
	cr := &urlfetch.CappedReader{R: resp.Body, Max: r.env.URLFetchMaxBytes}
	if err := be.Put(c.Request.Context(), real, cr, -1, ct); err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return false
	}
	if err := r.db.UpdateFilePathsAndStatus(c.Request.Context(), frow.ID, "PUBLISHED", cr.N, real, vpath, "", frow.TypeID); err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return false
	}
	out, _ := r.db.GetFile(c.Request.Context(), sid, frow.ID, false, true, u.ID)
	r.maybeEnqueueMedia(c.Request.Context(), sid, out)
	c.JSON(http.StatusCreated, out)
	return true
}
