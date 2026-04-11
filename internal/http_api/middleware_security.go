package http_api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

func securityHeadersMiddleware(env lib.Env) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if env.SecurityEnableHSTS && (c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func corsOptionsFromEnv(env lib.Env) (origins []string, allowCredentials bool) {
	s := strings.TrimSpace(env.CorsAllowedOrigins)
	if s == "" {
		return []string{"*"}, false
	}
	for _, p := range strings.Split(s, ",") {
		if o := strings.TrimSpace(p); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}, false
	}
	allowCredentials = true
	for _, o := range origins {
		if o == "*" {
			allowCredentials = false
			break
		}
	}
	return origins, allowCredentials
}
