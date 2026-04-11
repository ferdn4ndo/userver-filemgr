package urlfetch

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ferdn4ndo/userver-filemgr/lib"
)

func newSecureTransport(env lib.Env) http.RoundTripper {
	base := lib.NewPooledTransport()
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	allowLoopback := !env.IsProduction()

	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if err := validateHostname(host, allowLoopback); err != nil {
			return nil, err
		}
		if ip := net.ParseIP(host); ip != nil {
			if blockedIP(ip, allowLoopback) {
				return nil, fmt.Errorf("%w", ErrURLNotAllowed)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if !allResolvedIPsAllowed(addrs, allowLoopback) {
			return nil, fmt.Errorf("%w", ErrURLNotAllowed)
		}
		ip := addrs[0].IP.String()
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
	}

	base.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	return base
}
