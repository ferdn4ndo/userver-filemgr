package userver_auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// phaseCreateSystem calls POST /auth/system when SYSTEM_CREATION_TOKEN is set; returns updated FILEMGR_SYSTEM_TOKEN value.
func phaseCreateSystem(w io.Writer, client *http.Client, ctxBase, createTok, sysName, customTok string, priorSystemKey string) (systemAPIKey string, err error) {
	systemAPIKey = priorSystemKey
	if createTok == "" {
		return systemAPIKey, nil
	}
	if sysName == "" {
		return "", fmt.Errorf("bootstrap:auth: FILEMGR_BOOTSTRAP_SYSTEM_NAME is required when SYSTEM_CREATION_TOKEN is set")
	}
	createBody := map[string]any{"name": sysName}
	if customTok != "" {
		createBody["token"] = customTok
	}
	rawBody, _ := json.Marshal(createBody)
	req, err := http.NewRequest(http.MethodPost, ctxBase+"/auth/system", bytes.NewReader(rawBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+createTok)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("bootstrap:auth POST /auth/system: %w", err)
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
			return "", fmt.Errorf("bootstrap:auth: decode system response: %w", err)
		}
		if out.Token != "" {
			systemAPIKey = out.Token
			persistBootstrapEnv(w, map[string]string{
				"FILEMGR_SYSTEM_TOKEN": systemAPIKey,
			}, true)
		}
		_, _ = fmt.Fprintf(w, "Auth: created system %q; FILEMGR_SYSTEM_TOKEN is written to your env file when persist is enabled (token not logged).\n", out.Name)
		return systemAPIKey, nil
	case http.StatusConflict:
		_, _ = fmt.Fprintf(w, "Auth: system %q already exists.\n", sysName)
		if systemAPIKey == "" {
			_, _ = fmt.Fprintln(w, "Auth: set FILEMGR_SYSTEM_TOKEN to this system's API token if you need POST /auth/register next.")
		}
		return systemAPIKey, nil
	default:
		return "", fmt.Errorf("bootstrap:auth POST /auth/system: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func phaseRegisterAdmin(w io.Writer, client *http.Client, ctxBase, sysName, user, pass, systemAPIKey string) error {
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
		persistAfterRegister(w, systemAPIKey, user, pass, isAdminStr)
		return nil
	case http.StatusConflict:
		_, _ = fmt.Fprintf(w, "Auth: user %q already exists on system %q (OK).\n", user, sysName)
		persistAfterRegister(w, systemAPIKey, user, pass, isAdminStr)
		return nil
	default:
		return fmt.Errorf("bootstrap:auth POST /auth/register: status %d: %s", resp2.StatusCode, strings.TrimSpace(string(body2)))
	}
}

func persistAfterRegister(w io.Writer, systemAPIKey, user, pass, isAdminStr string) {
	persistBootstrapEnv(w, map[string]string{
		"FILEMGR_SYSTEM_TOKEN":             systemAPIKey,
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME": user,
		"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD": pass,
		"FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN": isAdminStr,
	}, false)
}
