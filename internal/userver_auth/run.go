package userver_auth

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

// Run optionally creates a uServer-Auth system and registers a first admin via HTTP.
// It is safe to run when variables are unset (no-op) or when the system already exists (continues if possible).
func Run(w io.Writer, env lib.Env) error {
	if strings.TrimSpace(os.Getenv("SKIP_AUTH_BOOTSTRAP")) == "1" {
		_, _ = fmt.Fprintln(w, "SKIP_AUTH_BOOTSTRAP=1: skipping bootstrap:auth")
		return nil
	}
	sysName := strings.TrimSpace(os.Getenv("FILEMGR_BOOTSTRAP_SYSTEM_NAME"))
	createTok := strings.TrimSpace(os.Getenv("SYSTEM_CREATION_TOKEN"))
	systemAPIKey := strings.TrimSpace(os.Getenv("FILEMGR_SYSTEM_TOKEN"))
	customTok := strings.TrimSpace(os.Getenv("FILEMGR_BOOTSTRAP_CUSTOM_SYSTEM_TOKEN"))
	user := strings.TrimSpace(os.Getenv("FILEMGR_BOOTSTRAP_ADMIN_USERNAME"))
	pass := os.Getenv("FILEMGR_BOOTSTRAP_ADMIN_PASSWORD")

	if createTok == "" && sysName == "" && user == "" && pass == "" {
		_, _ = fmt.Fprintln(w, "Auth bootstrap: no SYSTEM_CREATION_TOKEN / FILEMGR_BOOTSTRAP_* set; nothing to do.")
		return nil
	}

	base := strings.TrimSpace(env.AuthHost)
	if base == "" {
		return fmt.Errorf("bootstrap:auth: set USERVER_AUTH_HOST or AUTH_HOST when using SYSTEM_CREATION_TOKEN / FILEMGR_BOOTSTRAP_*")
	}

	client := &http.Client{Timeout: env.AuthHTTPTimeout}
	ctxBase := strings.TrimRight(base, "/")

	var err error
	systemAPIKey, err = phaseCreateSystem(w, client, ctxBase, createTok, sysName, customTok, systemAPIKey)
	if err != nil {
		return err
	}

	if user == "" && pass == "" {
		if createTok != "" {
			_, _ = fmt.Fprintln(w, "Auth: no FILEMGR_BOOTSTRAP_ADMIN_* — system step done.")
		}
		return nil
	}
	if user == "" || pass == "" {
		return fmt.Errorf("bootstrap:auth: set both FILEMGR_BOOTSTRAP_ADMIN_USERNAME and FILEMGR_BOOTSTRAP_ADMIN_PASSWORD")
	}
	if sysName == "" {
		return fmt.Errorf("bootstrap:auth: FILEMGR_BOOTSTRAP_SYSTEM_NAME is required for register")
	}
	if systemAPIKey == "" {
		return fmt.Errorf("bootstrap:auth: FILEMGR_SYSTEM_TOKEN is required to register (or run system create first)")
	}
	return phaseRegisterAdmin(w, client, ctxBase, sysName, user, pass, systemAPIKey)
}
