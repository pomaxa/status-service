// Package ssrf provides shared SSRF (server-side request forgery) protection
// for the service's outbound HTTP clients: a private-IP classifier and an
// http.Transport whose dialer rejects connections to internal addresses at
// connect time. Validating at dial time (after DNS resolution) is the only
// robust defense: it closes both redirect-to-internal bypasses (every hop
// re-dials) and DNS-rebinding / TOCTOU races (the IP that is checked is the IP
// that is connected to).
package ssrf

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"
)

// ErrBlockedHost is returned when a request targets an internal/private host.
var ErrBlockedHost = errors.New("access to internal/private IP addresses is not allowed")

// IsPrivateIP reports whether ip is loopback, link-local, unspecified,
// multicast, or within a private/internal range (RFC1918, CGNAT, ULA).
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"127.0.0.0/8",
		"100.64.0.0/10", // RFC6598 carrier-grade NAT
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// dialControl runs after DNS resolution, immediately before the socket
// connects, with the concrete IP:port being dialed. It rejects private targets.
func dialControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || IsPrivateIP(ip) {
		return ErrBlockedHost
	}
	return nil
}

// GuardedTransport returns an *http.Transport that refuses to connect to any
// private/internal IP, evaluated at dial time so redirects and DNS rebinding
// cannot escape the check.
func GuardedTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   dialControl,
	}
	return &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// RedirectGuard returns an http.Client.CheckRedirect function that re-validates
// every redirect hop's target host (defense in depth alongside the dial guard)
// and stops following after maxHops. validate is applied to each hop's URL.
func RedirectGuard(maxHops int, validate func(rawURL string) error) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxHops {
			return http.ErrUseLastResponse
		}
		if validate != nil {
			if err := validate(req.URL.String()); err != nil {
				return err
			}
		}
		return nil
	}
}
