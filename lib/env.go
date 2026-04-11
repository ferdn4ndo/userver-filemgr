package lib

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Env holds configuration loaded from the process environment.
type Env struct {
	ServerPort        string
	EnvMode           string
	LogLevel          string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxIdleTime time.Duration
	DBConnMaxLifetime time.Duration
	AuthHost          string
	AuthHTTPTimeout   time.Duration
	LocalRoot         string
	DownloadRatio     float64
	PublicBaseURL     string
	TrustedProxyCIDRs string
	CorsDebug         bool
	// CORS: comma-separated origins; empty = allow any origin ("*") for backward compatibility.
	CorsAllowedOrigins string
	// Security headers
	SecurityEnableHSTS bool
	// Rate limiting (per client IP)
	RateLimitEnabled        bool
	RateLimitRPS            float64
	RateLimitBurst          int
	RateLimitUploadURLRPS   float64
	RateLimitUploadURLBurst int
	// Audit log for sensitive actions (no secrets / no raw tokens)
	AuditLogEnabled bool
	// URL fetch (upload-from-url): SSRF protection and limits
	URLFetchMaxBytes     int64
	URLFetchTimeout      time.Duration
	URLFetchAllowHTTP    bool
	URLFetchMaxRedirects int
	// Media worker (in-process; PostgreSQL queue).
	MediaProcessingEnabled  bool
	MediaWorkerCount        int
	MediaWorkerPollMS       int
	FFprobePath             string
	MediaMaxImageBytes      int64
	MediaMaxVideoProbeBytes int64
}

// IsLocal is true when not running in production mode.
func (e Env) IsLocal() bool {
	return !e.IsProduction()
}

// IsProduction mirrors ENV_MODE=prod used by entrypoint.sh.
func (e Env) IsProduction() bool {
	return strings.EqualFold(strings.TrimSpace(e.EnvMode), "prod")
}

// NewEnv loads environment variables.
func NewEnv() Env {
	ratio, _ := strconv.ParseFloat(getenvDefault("DOWNLOAD_EXP_BYTES_SECS_RATIO", "4.25"), 64)
	if ratio <= 0 {
		ratio = 4.25
	}
	port := getenvFirstNonEmpty([]string{"APP_PORT", "PORT", "FLASK_PORT"}, "5000")
	mediaWorkers, _ := strconv.Atoi(getenvDefault("MEDIA_WORKER_COUNT", "2"))
	if mediaWorkers < 0 {
		mediaWorkers = 0
	}
	pollMS, _ := strconv.Atoi(getenvDefault("MEDIA_WORKER_POLL_MS", "1500"))
	if pollMS < 200 {
		pollMS = 200
	}
	maxImg, _ := strconv.ParseInt(getenvDefault("MEDIA_MAX_IMAGE_BYTES", fmt.Sprintf("%d", 80<<20)), 10, 64)
	if maxImg <= 0 {
		maxImg = 80 << 20
	}
	maxVid, _ := strconv.ParseInt(getenvDefault("MEDIA_MAX_VIDEO_PROBE_BYTES", fmt.Sprintf("%d", 256<<20)), 10, 64)
	if maxVid <= 0 {
		maxVid = 256 << 20
	}
	authTimeoutSecs, _ := strconv.Atoi(getenvDefault("USERVER_AUTH_TIMEOUT_SECS", "15"))
	if authTimeoutSecs < 5 {
		authTimeoutSecs = 5
	}
	if authTimeoutSecs > 120 {
		authTimeoutSecs = 120
	}
	authHost := strings.TrimRight(getenvFirstNonEmpty([]string{"USERVER_AUTH_HOST", "AUTH_HOST"}, ""), "/")

	dbMaxOpen, _ := strconv.Atoi(getenvDefault("DB_MAX_OPEN_CONNS", "25"))
	if dbMaxOpen <= 0 {
		dbMaxOpen = 25
	}
	dbMaxIdle, _ := strconv.Atoi(getenvDefault("DB_MAX_IDLE_CONNS", "10"))
	if dbMaxIdle < 0 {
		dbMaxIdle = 0
	}
	dbIdleSecs, _ := strconv.Atoi(getenvDefault("DB_CONN_MAX_IDLE_SECS", "90"))
	if dbIdleSecs < 0 {
		dbIdleSecs = 90
	}
	dbLifeSecs, _ := strconv.Atoi(getenvDefault("DB_CONN_MAX_LIFETIME_SECS", "0"))
	var dbConnMaxLife time.Duration
	if dbLifeSecs > 0 {
		dbConnMaxLife = time.Duration(dbLifeSecs) * time.Second
	}

	rps, _ := strconv.ParseFloat(getenvDefault("RATE_LIMIT_RPS", "30"), 64)
	if rps <= 0 {
		rps = 30
	}
	burst, _ := strconv.Atoi(getenvDefault("RATE_LIMIT_BURST", "60"))
	if burst <= 0 {
		burst = 60
	}
	urlRPS, _ := strconv.ParseFloat(getenvDefault("RATE_LIMIT_UPLOAD_URL_RPS", "5"), 64)
	if urlRPS <= 0 {
		urlRPS = 5
	}
	urlBurst, _ := strconv.Atoi(getenvDefault("RATE_LIMIT_UPLOAD_URL_BURST", "10"))
	if urlBurst <= 0 {
		urlBurst = 10
	}

	urlMax, _ := strconv.ParseInt(getenvDefault("URL_FETCH_MAX_BYTES", fmt.Sprintf("%d", 512<<20)), 10, 64)
	if urlMax <= 0 {
		urlMax = 512 << 20
	}
	urlTimeoutSecs, _ := strconv.Atoi(getenvDefault("URL_FETCH_TIMEOUT_SECS", "120"))
	if urlTimeoutSecs < 5 {
		urlTimeoutSecs = 5
	}
	if urlTimeoutSecs > 600 {
		urlTimeoutSecs = 600
	}
	urlRedirects, _ := strconv.Atoi(getenvDefault("URL_FETCH_MAX_REDIRECTS", "8"))
	if urlRedirects < 0 {
		urlRedirects = 0
	}
	if urlRedirects > 20 {
		urlRedirects = 20
	}

	rateOn := !strings.EqualFold(strings.TrimSpace(os.Getenv("RATE_LIMIT_ENABLED")), "0") &&
		!strings.EqualFold(strings.TrimSpace(os.Getenv("RATE_LIMIT_ENABLED")), "false")

	return Env{
		ServerPort:        port,
		EnvMode:           getenvDefault("ENV_MODE", "development"),
		LogLevel:          getenvDefault("LOG_LEVEL", "info"),
		DBHost:            os.Getenv("POSTGRES_HOST"),
		DBPort:            getenvDefault("POSTGRES_PORT", "5432"),
		DBUser:            os.Getenv("POSTGRES_USER"),
		DBPassword:        os.Getenv("POSTGRES_PASS"),
		DBName:            os.Getenv("POSTGRES_DB"),
		DBSSLMode:         strings.TrimSpace(os.Getenv("POSTGRES_SSLMODE")),
		DBMaxOpenConns:    dbMaxOpen,
		DBMaxIdleConns:    dbMaxIdle,
		DBConnMaxIdleTime: time.Duration(dbIdleSecs) * time.Second,
		DBConnMaxLifetime: dbConnMaxLife,
		AuthHost:          authHost,
		AuthHTTPTimeout:   time.Duration(authTimeoutSecs) * time.Second,
		LocalRoot:         getenvFirstNonEmpty([]string{"LOCAL_STORAGE_ROOT", "LOCAL_TEST_STORAGE_ROOT"}, "/storages/local"),
		DownloadRatio:     ratio,
		PublicBaseURL:     strings.TrimRight(os.Getenv("APP_PUBLIC_BASE_URL"), "/"),
		TrustedProxyCIDRs: os.Getenv("TRUSTED_PROXY_CIDRS"),
		CorsDebug: strings.EqualFold(strings.TrimSpace(os.Getenv("CORS_DEBUG")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("CORS_DEBUG")), "true"),
		CorsAllowedOrigins: strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")),
		SecurityEnableHSTS: strings.EqualFold(strings.TrimSpace(os.Getenv("SECURITY_ENABLE_HSTS")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("SECURITY_ENABLE_HSTS")), "true"),
		RateLimitEnabled:        rateOn,
		RateLimitRPS:            rps,
		RateLimitBurst:          burst,
		RateLimitUploadURLRPS:   urlRPS,
		RateLimitUploadURLBurst: urlBurst,
		AuditLogEnabled: !strings.EqualFold(strings.TrimSpace(os.Getenv("AUDIT_LOG_ENABLED")), "0") &&
			!strings.EqualFold(strings.TrimSpace(os.Getenv("AUDIT_LOG_ENABLED")), "false"),
		URLFetchMaxBytes: urlMax,
		URLFetchTimeout:  time.Duration(urlTimeoutSecs) * time.Second,
		URLFetchAllowHTTP: strings.EqualFold(strings.TrimSpace(os.Getenv("URL_FETCH_ALLOW_HTTP")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("URL_FETCH_ALLOW_HTTP")), "true"),
		URLFetchMaxRedirects: urlRedirects,
		MediaProcessingEnabled: !strings.EqualFold(strings.TrimSpace(os.Getenv("MEDIA_PROCESSING_ENABLED")), "0") &&
			!strings.EqualFold(strings.TrimSpace(os.Getenv("MEDIA_PROCESSING_ENABLED")), "false"),
		MediaWorkerCount:        mediaWorkers,
		MediaWorkerPollMS:       pollMS,
		FFprobePath:             resolveFFprobePath(),
		MediaMaxImageBytes:      maxImg,
		MediaMaxVideoProbeBytes: maxVid,
	}
}

func resolveFFprobePath() string {
	if p := strings.TrimSpace(os.Getenv("FFPROBE_PATH")); p != "" {
		return p
	}
	if path, err := exec.LookPath("ffprobe"); err == nil {
		return path
	}
	return ""
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getenvFirstNonEmpty(keys []string, def string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return def
}
