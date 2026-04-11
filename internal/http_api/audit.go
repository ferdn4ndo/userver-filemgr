package http_api

import (
	"github.com/gin-gonic/gin"
)

func (r *Router) audit(c *gin.Context, action string, kvs ...string) {
	if !r.env.AuditLogEnabled {
		return
	}
	if len(kvs)%2 != 0 {
		kvs = append(kvs, "")
	}
	args := []any{"action", action, "path", c.Request.URL.Path, "method", c.Request.Method}
	if u := ctxUser(c); u != nil {
		args = append(args, "user_id", u.ID.String(), "username", u.Username)
	}
	for i := 0; i < len(kvs); i += 2 {
		args = append(args, kvs[i], kvs[i+1])
	}
	r.log.Infow("audit", args...)
}
