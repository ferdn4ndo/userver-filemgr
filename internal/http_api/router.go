package http_api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	cors "github.com/rs/cors"
	corsgin "github.com/rs/cors/wrapper/gin"

	"github.com/ferdn4ndo/userver-filemgr/internal/auth"
	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/download_exp"
	"github.com/ferdn4ndo/userver-filemgr/internal/media_processor"
	"github.com/ferdn4ndo/userver-filemgr/internal/object_store"
	"github.com/ferdn4ndo/userver-filemgr/internal/urlfetch"
	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// Router wires HTTP routes (legacy /storages/* paths).
type Router struct {
	env      lib.Env
	gin      *gin.Engine
	db       *data.DB
	authz    *auth.Service
	objects  *object_store.Factory
	urlFetch *urlfetch.Client
	log      lib.Logger
	globalRL *ipRateLimiter
	urlRL    *ipRateLimiter
}

func NewRouter(handler lib.RequestHandler, env lib.Env, logger lib.Logger, db *data.DB, az *auth.Service, ob *object_store.Factory, fetch *urlfetch.Client) *Router {
	return &Router{
		gin:      handler.Gin,
		env:      env,
		db:       db,
		authz:    az,
		objects:  ob,
		urlFetch: fetch,
		log:      logger,
		globalRL: newIPRateLimiter(env.RateLimitEnabled, env.RateLimitRPS, env.RateLimitBurst),
		urlRL:    newIPRateLimiter(env.RateLimitEnabled, env.RateLimitUploadURLRPS, env.RateLimitUploadURLBurst),
	}
}

func (r *Router) Register() {
	g := r.gin
	origins, allowCreds := corsOptionsFromEnv(r.env)
	g.Use(corsgin.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: allowCreds,
		Debug:            r.env.CorsDebug,
	}))
	g.Use(securityHeadersMiddleware(r.env))

	g.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	needAuth := g.Group("")
	needAuth.Use(rateLimitMiddleware(r.globalRL))
	needAuth.Use(r.authenticateMiddleware)

	needAuth.GET("/storages", r.listStorages)
	needAuth.POST("/storages", r.createStorage)
	needAuth.GET("/storages/:storageID", r.getStorage)
	needAuth.PUT("/storages/:storageID", r.updateStorage)
	needAuth.PATCH("/storages/:storageID", r.patchStorage)
	needAuth.DELETE("/storages/:storageID", r.deleteStorage)

	needAuth.GET("/storages/:storageID/files", r.listFiles)
	needAuth.GET("/storages/:storageID/files/:fileID", r.getFile)
	needAuth.PATCH("/storages/:storageID/files/:fileID", r.patchFile)
	needAuth.DELETE("/storages/:storageID/files/:fileID", r.deleteFile)
	needAuth.POST("/storages/:storageID/files/:fileID/download", r.createDownload)

	needAuth.GET("/storages/:storageID/trash", r.listTrash)
	needAuth.GET("/storages/:storageID/trash/:fileID", r.getTrashFile)
	needAuth.DELETE("/storages/:storageID/trash/:fileID", r.permaDelete)

	needAuth.GET("/storages/:storageID/users", r.listStorageUsers)
	needAuth.POST("/storages/:storageID/users", r.createStorageUser)
	needAuth.GET("/storages/:storageID/users/:userRowID", r.getStorageUser)
	needAuth.PATCH("/storages/:storageID/users/:userRowID", r.patchStorageUser)
	needAuth.DELETE("/storages/:storageID/users/:userRowID", r.deleteStorageUser)

	needAuth.POST("/storages/:storageID/upload-from-file", r.uploadMultipart)
	needAuth.POST("/storages/:storageID/upload-from-url", rateLimitMiddleware(r.urlRL), r.uploadFromURL)

	needAuth.GET("/storages/:storageID/files/:fileID/raw/:downloadID", r.streamDownload)

	// Media CRUD was backed by optional async processing in the old stack; expose explicit stubs until a worker exists.
	needAuth.GET("/storages/:storageID/media", r.listMedia)
	needAuth.GET("/storages/:storageID/media/:mediaID", r.getMedia)
}

func (r *Router) authenticateMiddleware(c *gin.Context) {
	h := c.GetHeader("Authorization")
	if h == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Authentication credentials were not provided."})
		return
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token header."})
		return
	}
	scheme := strings.TrimSpace(parts[0])
	if !strings.EqualFold(scheme, "Token") && !strings.EqualFold(scheme, "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token header."})
		return
	}
	u, err := r.authz.Authenticate(c.Request.Context(), strings.TrimSpace(parts[1]))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token."})
		return
	}
	c.Set("user", u)
	c.Next()
}

func ctxUser(c *gin.Context) *auth.User {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	u := v.(*auth.User)
	return u
}

func (r *Router) listStorages(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	limit, offset := paginate(c)
	items, total, err := r.db.ListStorages(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": total, "_links": pageLinks(c, total, limit, offset)})
}

func (r *Router) createStorage(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	var body struct {
		Type                      string          `json:"type"`
		Credentials               json.RawMessage `json:"credentials"`
		MediaConvertConfiguration json.RawMessage `json:"media_convert_configuration"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if len(body.Credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "credentials required"})
		return
	}
	st, err := r.db.InsertStorage(c.Request.Context(), body.Type, body.Credentials, body.MediaConvertConfiguration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	st.Credentials = nil
	r.audit(c, "storage.create", "storage_id", st.ID.String())
	c.JSON(http.StatusOK, st)
}

func (r *Router) getStorage(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	st.Credentials = nil
	c.JSON(http.StatusOK, st)
}

func (r *Router) updateStorage(c *gin.Context) { r.writeStorage(c, true) }
func (r *Router) patchStorage(c *gin.Context)  { r.writeStorage(c, false) }

func (r *Router) writeStorage(c *gin.Context, requireCreds bool) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	var body struct {
		Type                      string          `json:"type"`
		Credentials               json.RawMessage `json:"credentials"`
		MediaConvertConfiguration json.RawMessage `json:"media_convert_configuration"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if requireCreds && len(body.Credentials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "credentials required"})
		return
	}
	existing, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	creds := body.Credentials
	if len(creds) == 0 {
		creds = existing.Credentials
	}
	media := body.MediaConvertConfiguration
	if len(media) == 0 {
		media = existing.MediaConvertConfiguration
	}
	stType := body.Type
	if stType == "" && existing.Type.Valid {
		stType = existing.Type.String
	}
	st, err := r.db.UpdateStorage(c.Request.Context(), sid, stType, creds, media)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	st.Credentials = nil
	if requireCreds {
		r.audit(c, "storage.update", "storage_id", sid.String())
	} else {
		r.audit(c, "storage.patch", "storage_id", sid.String())
	}
	c.JSON(http.StatusOK, st)
}

func (r *Router) deleteStorage(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	if err := r.db.DeleteStorage(c.Request.Context(), sid); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	r.audit(c, "storage.delete", "storage_id", sid.String())
	c.Status(http.StatusNoContent)
}

func (r *Router) listFiles(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	limit, offset := paginate(c)
	items, total, err := r.db.ListFiles(c.Request.Context(), sid, false, u.IsAdmin, u.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": total, "_links": pageLinks(c, total, limit, offset)})
}

func (r *Router) listMedia(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	if ok, _ := r.db.StorageUserMayRead(c.Request.Context(), sid, u.ID); !ok && !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	limit, offset := paginate(c)
	items, total, err := r.db.ListStorageMedia(c.Request.Context(), sid, u.IsAdmin, u.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": total, "_links": pageLinks(c, total, limit, offset)})
}

func (r *Router) getMedia(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	mid, err := uuid.Parse(c.Param("mediaID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid media id"})
		return
	}
	m, err := r.db.GetStorageMedia(c.Request.Context(), sid, mid, u.IsAdmin, u.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

func (r *Router) getFile(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, f)
}

func (r *Router) patchFile(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	if !u.IsAdmin && (!f.OwnerID.Valid || f.OwnerID.UUID != u.ID) {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	var body struct {
		Name           *string         `json:"name"`
		Visibility     *string         `json:"visibility"`
		CustomMetadata json.RawMessage `json:"custom_metadata"`
		VirtualPath    *string         `json:"virtual_path"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if err := r.db.UpdateFileMeta(c.Request.Context(), sid, fid, body.Name, body.Visibility, body.CustomMetadata, body.VirtualPath, u.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	nf, _ := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	r.maybeEnqueueMedia(c.Request.Context(), sid, nf)
	c.JSON(http.StatusOK, nf)
}

func (r *Router) deleteFile(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	if !u.IsAdmin && (!f.OwnerID.Valid || f.OwnerID.UUID != u.ID) {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	if err := r.db.SetFileExcluded(c.Request.Context(), sid, fid, true, u.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	r.audit(c, "file.soft_delete", "storage_id", sid.String(), "file_id", fid.String())
	c.Status(http.StatusNoContent)
}

func (r *Router) listTrash(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	limit, offset := paginate(c)
	items, total, err := r.db.ListFiles(c.Request.Context(), sid, true, u.IsAdmin, u.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": total, "_links": pageLinks(c, total, limit, offset)})
}

func (r *Router) getTrashFile(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, true, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, f)
}

func (r *Router) permaDelete(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, true, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	if !u.IsAdmin && (!f.OwnerID.Valid || f.OwnerID.UUID != u.ID) {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	be, err := r.objects.ForStorage(st.Type.String, st.Credentials)
	if err == nil && f.RealPath.Valid {
		_ = be.Delete(c.Request.Context(), f.RealPath.String)
	}
	if err := r.db.PermaDeleteFile(c.Request.Context(), sid, fid); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	r.audit(c, "file.perma_delete", "storage_id", sid.String(), "file_id", fid.String())
	c.Status(http.StatusNoContent)
}

func (r *Router) createDownload(c *gin.Context) {
	u := ctxUser(c)
	sid, fid, ok := r.parseSF(c)
	if !ok {
		return
	}
	f, err := r.db.GetFile(c.Request.Context(), sid, fid, false, u.IsAdmin, u.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	var body struct {
		ForceDownload *bool `json:"force_download"`
	}
	_ = c.ShouldBindJSON(&body)
	force := true
	if body.ForceDownload != nil {
		force = *body.ForceDownload
	}
	sec := download_exp.SecondsFromEnv(f.Size, r.env.DownloadRatio)
	exp := time.Now().UTC().Add(time.Duration(sec) * time.Second)
	row := &data.FileDownload{
		ForceDownload: force,
		ExpiresAt:     exp,
		OwnerID:       uuid.NullUUID{UUID: u.ID, Valid: true},
		StorageFileID: f.ID,
	}
	if err := r.db.InsertDownload(c.Request.Context(), row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	be, err := r.objects.ForStorage(st.Type.String, st.Credentials)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	ct := ""
	if f.Type != nil {
		ct = f.Type.MimeType
	}
	vpath := ""
	if f.VirtualPath.Valid {
		vpath = f.VirtualPath.String
	}
	rp := ""
	if f.RealPath.Valid {
		rp = f.RealPath.String
	}
	var url string
	if st.Type.String == "AMAZON_S3" {
		url, err = be.DownloadURL(c.Request.Context(), rp, vpath, ct, time.Duration(sec)*time.Second, force)
	} else {
		base := r.env.PublicBaseURL
		if base == "" {
			base = ""
		}
		url = strings.TrimRight(base, "/") + "/storages/" + sid.String() + "/files/" + fid.String() + "/raw/" + row.ID.String()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	_ = r.db.UpdateDownloadURL(c.Request.Context(), row.ID, url)
	row.DownloadURL = sql.NullString{String: url, Valid: true}
	c.JSON(http.StatusCreated, row)
}

// streamDownload serves local files for a valid download row (S3 uses presigned URLs).
func (r *Router) streamDownload(c *gin.Context) {
	u := ctxUser(c)
	did, sid, fid, ok := r.streamDownloadParseIDs(c)
	if !ok {
		return
	}
	dl, f, st, ok := r.streamDownloadLoadContext(c, u, did, sid, fid)
	if !ok {
		return
	}
	if redirectIfExternalDownloadURL(c, dl) {
		return
	}
	if !f.RealPath.Valid {
		c.JSON(http.StatusNotFound, gin.H{"detail": "no path"})
		return
	}
	be, err := r.objects.ForStorage(st.Type.String, st.Credentials)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	p, err := be.DownloadURL(c.Request.Context(), f.RealPath.String, "", "", 0, dl.ForceDownload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.FileAttachment(p, streamDownloadAttachmentName(f))
}

func (r *Router) listStorageUsers(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	rows, err := r.db.ListStorageUsers(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "count": len(rows)})
}

func (r *Router) createStorageUser(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	var body struct {
		UserID   uuid.UUID `json:"user"`
		MayRead  bool      `json:"may_read"`
		MayWrite bool      `json:"may_write"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	row, err := r.db.InsertStorageUser(c.Request.Context(), sid, body.UserID, body.MayRead, body.MayWrite, u.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	r.audit(c, "storage_user.create", "storage_id", sid.String(), "subject_user_id", body.UserID.String())
	c.JSON(http.StatusCreated, row)
}

func (r *Router) getStorageUser(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	rid, err := uuid.Parse(c.Param("userRowID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	row, err := r.db.GetStorageUser(c.Request.Context(), sid, rid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (r *Router) patchStorageUser(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	rid, err := uuid.Parse(c.Param("userRowID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	var body struct {
		MayRead  *bool `json:"may_read"`
		MayWrite *bool `json:"may_write"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	row, err := r.db.UpdateStorageUser(c.Request.Context(), sid, rid, body.MayRead, body.MayWrite, u.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (r *Router) deleteStorageUser(c *gin.Context) {
	u := ctxUser(c)
	if !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	rid, err := uuid.Parse(c.Param("userRowID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid id"})
		return
	}
	if err := r.db.DeleteStorageUser(c.Request.Context(), sid, rid); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
		return
	}
	r.audit(c, "storage_user.delete", "storage_id", sid.String(), "row_id", rid.String())
	c.Status(http.StatusNoContent)
}

func (r *Router) uploadMultipart(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	if ok, _ := r.db.StorageUserMayWrite(c.Request.Context(), sid, u.ID); !ok && !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "storage not found"})
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "file required"})
		return
	}
	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	defer src.Close()

	id := uuid.New()
	sig := randomSig()
	ext := ""
	if i := strings.LastIndex(fh.Filename, "."); i >= 0 {
		ext = strings.ToLower(fh.Filename[i+1:])
	}
	mimeID := uuid.NullUUID{}
	if mt, err := r.db.FindMimeByExtension(c.Request.Context(), ext); err == nil {
		mimeID = uuid.NullUUID{UUID: mt.ID, Valid: true}
	}
	vpath := "/" + id.String() + "/" + fh.Filename
	real := strings.TrimPrefix(vpath, "/")
	frow := &data.StorageFile{
		ID: id, SignatureKey: sig, StorageID: sid,
		OwnerID: uuid.NullUUID{UUID: u.ID, Valid: true},
		Name:    sql.NullString{String: fh.Filename, Valid: true},
		Status:  "UPLOADING", Visibility: "SYSTEM", Size: 0,
		Extension: sql.NullString{String: ext, Valid: ext != ""},
		Origin:    "LOCAL", VirtualPath: sql.NullString{String: vpath, Valid: true},
		RealPath:    sql.NullString{String: real, Valid: true},
		CreatedByID: uuid.NullUUID{UUID: u.ID, Valid: true},
		TypeID:      mimeID,
	}
	if err := r.db.InsertFile(c.Request.Context(), frow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	be, err := r.objects.ForStorage(st.Type.String, st.Credentials)
	if err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	ct := fh.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	if err := be.Put(c.Request.Context(), real, src, fh.Size, ct); err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	if err := r.db.UpdateFilePathsAndStatus(c.Request.Context(), id, "PUBLISHED", fh.Size, real, vpath, "", mimeID); err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	out, _ := r.db.GetFile(c.Request.Context(), sid, id, false, true, u.ID)
	r.maybeEnqueueMedia(c.Request.Context(), sid, out)
	c.JSON(http.StatusCreated, out)
}

func (r *Router) uploadFromURL(c *gin.Context) {
	u := ctxUser(c)
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return
	}
	if ok, _ := r.db.StorageUserMayWrite(c.Request.Context(), sid, u.ID); !ok && !u.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"detail": "forbidden"})
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "url required"})
		return
	}
	st, err := r.db.GetStorage(c.Request.Context(), sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "storage not found"})
		return
	}
	ext := extensionFromSourceURL(body.URL)
	frow, ok := r.newUploadingFileFromURL(c, u, sid, body.URL, ext)
	if !ok {
		return
	}
	resp, err := r.urlFetch.Get(c.Request.Context(), body.URL)
	if err != nil {
		r.rollbackUploadingFile(c.Request.Context(), st, frow)
		if errors.Is(err, urlfetch.ErrURLNotAllowed) {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "url not allowed"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"detail": "failed to fetch url"})
		return
	}
	r.putStreamURLUpload(c, st, frow, resp, sid, u)
}

func (r *Router) parseSF(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	sid, err := uuid.Parse(c.Param("storageID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid storage"})
		return uuid.Nil, uuid.Nil, false
	}
	fid, err := uuid.Parse(c.Param("fileID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "invalid file"})
		return uuid.Nil, uuid.Nil, false
	}
	return sid, fid, true
}

func paginate(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit"))
	offset, _ = strconv.Atoi(c.Query("offset"))
	return
}

func pageLinks(c *gin.Context, total, limit, offset int) gin.H {
	if limit <= 0 {
		limit = 100
	}
	self := c.Request.URL.Path + "?" + c.Request.URL.RawQuery
	next := ""
	if offset+limit < total {
		next = c.Request.URL.Path + "?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset+limit)
	}
	prev := ""
	if offset > 0 {
		po := offset - limit
		if po < 0 {
			po = 0
		}
		prev = c.Request.URL.Path + "?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(po)
	}
	return gin.H{"self": gin.H{"href": self}, "next": gin.H{"href": next}, "previous": gin.H{"href": prev}}
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

func (r *Router) maybeEnqueueMedia(ctx context.Context, storageID uuid.UUID, f *data.StorageFile) {
	if f == nil || !r.env.MediaProcessingEnabled {
		return
	}
	if f.Status != "PUBLISHED" || !media_processor.ShouldEnqueue(f) {
		return
	}
	exists, err := r.db.StorageMediaExistsForFile(ctx, f.ID)
	if err != nil || exists {
		return
	}
	_ = r.db.EnqueueMediaProcessing(ctx, storageID, f.ID)
}
