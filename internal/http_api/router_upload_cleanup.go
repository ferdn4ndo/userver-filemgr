package http_api

import (
	"context"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
)

// rollbackUploadingFile deletes any object bytes at frow's real_path (best effort) and removes the
// UPLOADING database row. Safe to call multiple times; ignores missing objects / already-deleted rows.
func (r *Router) rollbackUploadingFile(ctx context.Context, st *data.Storage, frow *data.StorageFile) {
	if frow == nil || st == nil {
		return
	}
	if frow.RealPath.Valid && frow.RealPath.String != "" {
		if be, err := r.objects.ForStorage(st.Type.String, st.Credentials); err == nil {
			_ = be.Delete(ctx, frow.RealPath.String)
		}
	}
	_ = r.db.DeleteFailedUploadingFile(ctx, frow.StorageID, frow.ID)
}
