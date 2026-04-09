package userver_auth

import (
	"bytes"
	"encoding/json"
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

	// Phase 1: create system (only when SYSTEM_CREATION_TOKEN is set; requires system name).
	if createTok != "" {
		if sysName == "" {
			return fmt.Errorf("bootstrap:auth: FILEMGR_BOOTSTRAP_SYSTEM_NAME is required when SYSTEM_CREATION_TOKEN is set")
		}
		createBody := map[string]any{"name": sysName}
		if customTok != "" {
			createBody["token"] = customTok
		}
		rawBody, _ := json.Marshal(createBody)
		req, err := http.NewRequest(http.MethodPost, ctxBase+"/auth/system", bytes.NewReader(rawBody))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Token "+createTok)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("bootstrap:auth POST /auth/system: %w", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

		switch resp.StatusCode {
		case http.StatusCreated:
			var out struct {
				Token string `json:"token"`
				Name  string `json:"name"`
			}
			if err := json.Unmarshal(body, &out); err != nil {
				return fmt.Errorf("bootstrap:auth: decode system response: %w", err)
			}
			if out.Token != "" {
				systemAPIKey = out.Token
				persistBootstrapEnv(w, map[string]string{
					"FILEMGR_SYSTEM_TOKEN": systemAPIKey,
				}, true)
			}
			_, _ = fmt.Fprintf(w, "Auth: created system %q; FILEMGR_SYSTEM_TOKEN is written to your env file when persist is enabled (token not logged).\n", out.Name)
		case http.StatusConflict:
			_, _ = fmt.Fprintf(w, "Auth: system %q already exists.\n", sysName)
			if systemAPIKey == "" {
				_, _ = fmt.Fprintln(w, "Auth: set FILEMGR_SYSTEM_TOKEN to this system's API token if you need POST /auth/register next.")
			}
		default:
			return fmt.Errorf("bootstrap:auth POST /auth/system: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	// Phase 2: register first admin (optional).
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

	isAdmin := true
	if v := strings.TrimSpace(os.Getenv("FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN")); v == "0" || strings.EqualFold(v, "false") {
		isAdmin = false
	}
	isAdminStr := "1"
	if !isAdmin {
		isAdminStr = "0"
	}
	regBody, _ := json.Marshal(map[string]any{
		"username":     user,
		"system_name":  sysName,
		"system_token": systemAPIKey,
		"password":     pass,
		"is_admin":     isAdmin,
	})
	req2, err := http.NewRequest(http.MethodPost, ctxBase+"/auth/register", bytes.NewReader(regBody))
	if err != nil {
		return err
	}
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	if err != nil {
		return fmt.Errorf("bootstrap:auth POST /auth/register: %w", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(io.LimitReader(resp2.Body, 1<<20))
	switch resp2.StatusCode {
	case http.StatusCreated:
		_, _ = fmt.Fprintf(w, "Auth: registered user %q on system %q (is_admin=%v).\n", user, sysName, isAdmin)
		persistBootstrapEnv(w, map[string]string{
			"FILEMGR_SYSTEM_TOKEN":             systemAPIKey,
			"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": user,
			"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD": pass,
			"FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN": isAdminStr,
		}, false)
		return nil
	case http.StatusConflict:
		_, _ = fmt.Fprintf(w, "Auth: user %q already exists on system %q (OK).\n", user, sysName)
		persistBootstrapEnv(w, map[string]string{
			"FILEMGR_SYSTEM_TOKEN":             systemAPIKey,
			"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": user,
			"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD": pass,
			"FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN": isAdminStr,
		}, false)
		return nil
	default:
		return fmt.Errorf("bootstrap:auth POST /auth/register: status %d: %s", resp2.StatusCode, strings.TrimSpace(string(body2)))
	}
}
