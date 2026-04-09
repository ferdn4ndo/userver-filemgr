package userver_auth

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var envLineKey = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// parseEnvValue decodes the right-hand side of a KEY=value line (minimal quoting rules).
func parseEnvValue(raw string) string {
	v := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		inner := v[1 : len(v)-1]
		var b strings.Builder
		for i := 0; i < len(inner); i++ {
			if inner[i] == '\\' && i+1 < len(inner) {
				switch inner[i+1] {
				case '"', '\\':
					b.WriteByte(inner[i+1])
					i++
					continue
				case 'n':
					b.WriteByte('\n')
					i++
					continue
				case 'r':
					b.WriteByte('\r')
					i++
					continue
				}
			}
			b.WriteByte(inner[i])
		}
		return b.String()
	}
	return v
}

func isPlaceholderEnvValue(v string) bool {
	t := strings.TrimSpace(v)
	return strings.Contains(t, "<") && strings.Contains(t, ">")
}

// readLastEnvFileValues returns the last assignment per key (last line wins), values decoded.
func readLastEnvFileValues(path string) (map[string]string, error) {
	out := make(map[string]string)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return out, nil
	}
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		m := envLineKey.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		k := m[1]
		out[k] = parseEnvValue(m[2])
	}
	return out, nil
}

// filterUpdatesSkipProtectedInFile drops keys that already have a non-empty, non-placeholder value on disk.
func filterUpdatesSkipProtectedInFile(path string, updates map[string]string) (map[string]string, error) {
	existing, err := readLastEnvFileValues(path)
	if err != nil {
		return nil, err
	}
	filtered := make(map[string]string)
	for k, v := range updates {
		if v == "" {
			continue
		}
		cur, ok := existing[k]
		if ok {
			cur = strings.TrimSpace(cur)
			if cur != "" && !isPlaceholderEnvValue(cur) {
				continue
			}
		}
		filtered[k] = v
	}
	return filtered, nil
}

// formatEnvValue returns a safe value for a KEY= line (quotes when needed).
func formatEnvValue(s string) string {
	needsQuote := s == ""
	if !needsQuote {
		for _, r := range s {
			if r <= ' ' || r == '#' || r == '"' || r == '\'' || r == '=' || r == '$' {
				needsQuote = true
				break
			}
		}
	}
	if !needsQuote {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func resolveEnvFilePath() string {
	for _, k := range []string{"ENV_FILE", "FILEMGR_ENV_FILE"} {
		if p := strings.TrimSpace(os.Getenv(k)); p != "" {
			return p
		}
	}
	return ".env"
}

func mergeEnvLinesIntoBuilder(lines []string, filtered map[string]string) (*strings.Builder, map[string]bool) {
	out := &strings.Builder{}
	written := make(map[string]bool)
	for _, line := range lines {
		m := envLineKey.FindStringSubmatch(line)
		if m != nil {
			k := m[1]
			if _, ok := filtered[k]; ok {
				if !written[k] {
					fmt.Fprintf(out, "%s=%s\n", k, formatEnvValue(filtered[k]))
					written[k] = true
				}
				continue
			}
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out, written
}

func appendMissingEnvKeys(out *strings.Builder, filtered map[string]string, written map[string]bool) {
	order := []string{
		"FILEMGR_SYSTEM_TOKEN",
		"FILEMGR_BOOTSTRAP_ADMIN_USERNAME",
		"FILEMGR_BOOTSTRAP_ADMIN_PASSWORD",
		"FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN",
	}
	for _, k := range order {
		if v, ok := filtered[k]; ok && !written[k] {
			fmt.Fprintf(out, "%s=%s\n", k, formatEnvValue(v))
			written[k] = true
		}
	}
	for k, v := range filtered {
		if !written[k] {
			fmt.Fprintf(out, "%s=%s\n", k, formatEnvValue(v))
		}
	}
}

// upsertEnvFile replaces or appends KEY=value lines. Duplicate keys in the file collapse to one updated line.
func upsertEnvFile(path string, updates map[string]string) error {
	filtered := make(map[string]string)
	for k, v := range updates {
		if v == "" {
			continue
		}
		filtered[k] = v
	}
	if len(filtered) == 0 {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	var lines []string
	if content != "" {
		lines = strings.Split(content, "\n")
	}

	out, written := mergeEnvLinesIntoBuilder(lines, filtered)
	appendMissingEnvKeys(out, filtered, written)

	return os.WriteFile(path, []byte(strings.TrimSuffix(out.String(), "\n")+"\n"), 0o600)
}

// persistBootstrapEnv merges updates into ENV_FILE / FILEMGR_ENV_FILE (default .env).
// When quiet is false and the write succeeds, logs a single line (never logs secret values).
func persistBootstrapEnv(w io.Writer, updates map[string]string, quiet bool) {
	if strings.TrimSpace(os.Getenv("FILEMGR_SKIP_PERSIST_BOOTSTRAP_ENV")) == "1" {
		return
	}
	has := false
	for _, v := range updates {
		if v != "" {
			has = true
			break
		}
	}
	if !has {
		return
	}
	path := resolveEnvFilePath()
	filtered, err := filterUpdatesSkipProtectedInFile(path, updates)
	if err != nil {
		_, _ = fmt.Fprintf(w, "Auth: skipping env persist (could not read %s: %v)\n", path, err)
		return
	}
	if len(filtered) == 0 {
		return
	}
	if err := upsertEnvFile(path, filtered); err != nil {
		_, _ = fmt.Fprintf(w, "Auth: could not persist bootstrap env to %s: %v\n", path, err)
		return
	}
	if !quiet {
		_, _ = fmt.Fprintf(w, "Auth: persisted bootstrap fields to %s (existing non-empty values in the file were not overwritten).\n", path)
	}
}
