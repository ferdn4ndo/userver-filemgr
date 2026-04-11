package urlfetch

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrURLNotAllowed is returned when a URL is rejected by SSRF policy.
var ErrURLNotAllowed = errors.New("url not allowed")

func validateFetchURL(u *url.URL, allowHTTP, allowLoopback bool) error {
	if u == nil {
		return fmt.Errorf("urlfetch: nil url")
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "https":
	case "http":
		if !allowHTTP {
			return fmt.Errorf("urlfetch: only https URLs are allowed (set URL_FETCH_ALLOW_HTTP=1 to allow http)")
		}
	default:
		return fmt.Errorf("urlfetch: unsupported URL scheme %q", u.Scheme)
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("urlfetch: empty host")
	}
	if err := validateHostname(host, allowLoopback); err != nil {
		return err
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		if blockedIP(ip, allowLoopback) {
			return fmt.Errorf("%w", ErrURLNotAllowed)
		}
	}
	return nil
}

func validateHostname(host string, allowLoopback bool) error {
	h := strings.ToLower(host)
	switch h {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		if !allowLoopback {
			return fmt.Errorf("%w", ErrURLNotAllowed)
		}
	}
	if h == "metadata.google.internal" || strings.HasSuffix(h, ".metadata.google.internal") {
		return fmt.Errorf("%w", ErrURLNotAllowed)
	}
	if strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".localhost") {
		if !allowLoopback {
			return fmt.Errorf("%w", ErrURLNotAllowed)
		}
	}
	if strings.HasSuffix(h, ".internal") && !allowLoopback {
		return fmt.Errorf("%w", ErrURLNotAllowed)
	}
	return nil
}

func blockedIP(ip net.IP, allowLoopback bool) bool {
	if allowLoopback && ip.IsLoopback() {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 || ip4[0] == 127 {
			return true
		}
	}
	// Unique local IPv6 (fc00::/7)
	if len(ip) == net.IPv6len && (ip[0] == 0xfc || ip[0] == 0xfd) {
		return true
	}
	return false
}

func allResolvedIPsAllowed(addrs []net.IPAddr, allowLoopback bool) bool {
	if len(addrs) == 0 {
		return false
	}
	for _, a := range addrs {
		if blockedIP(a.IP, allowLoopback) {
			return false
		}
	}
	return true
}
