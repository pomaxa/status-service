package http_checker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// TestIsPrivateIP exercises the pure isPrivateIP function with a wide range of
// inputs to cover every branch (nil, loopback, link-local, each private CIDR,
// IPv6 unique-local/link-local, and public addresses).
func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"nil ip", "", true /*sentinel handled below*/},

		// Loopback
		{"ipv4 loopback 127.0.0.1", "127.0.0.1", true},
		{"ipv4 loopback range 127.5.5.5", "127.5.5.5", true},
		{"ipv6 loopback ::1", "::1", true},

		// Link-local unicast / multicast
		{"ipv4 link-local 169.254.1.1", "169.254.1.1", true},
		{"ipv6 link-local unicast fe80::1", "fe80::1", true},
		{"ipv6 link-local multicast ff02::1", "ff02::1", true},
		{"ipv4 link-local multicast 224.0.0.1", "224.0.0.1", true},

		// RFC1918 private ranges
		{"10.0.0.0/8 -> 10.1.2.3", "10.1.2.3", true},
		{"172.16.0.0/12 -> 172.16.5.5", "172.16.5.5", true},
		{"172.16.0.0/12 -> 172.31.255.255", "172.31.255.255", true},
		{"192.168.0.0/16 -> 192.168.1.1", "192.168.1.1", true},

		// IPv6 unique-local fc00::/7
		{"ipv6 unique-local fc00::1", "fc00::1", true},
		{"ipv6 unique-local fd12::abcd", "fd12::abcd", true},

		// Public addresses
		{"public ipv4 8.8.8.8", "8.8.8.8", false},
		{"public ipv4 1.1.1.1", "1.1.1.1", false},
		{"public ipv4 172.32.0.1 (just outside 172.16/12)", "172.32.0.1", false},
		{"public ipv6 2001:4860:4860::8888", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil ip" {
				if isPrivateIP(nil) {
					t.Error("expected isPrivateIP(nil) = false")
				}
				return
			}
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tt.ip)
			}
			if got := isPrivateIP(ip); got != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

// TestValidateURL_IPLiteralPrivate covers the branch where the host is a
// parseable private IP literal, which must be blocked.
func TestValidateURL_IPLiteralPrivate(t *testing.T) {
	checker := New(5 * time.Second)

	privateURLs := []string{
		"http://127.0.0.1/health",
		"http://10.0.0.5:8080/",
		"http://192.168.1.1/",
		"http://[::1]/health",
	}

	for _, u := range privateURLs {
		t.Run(u, func(t *testing.T) {
			if err := checker.validateURL(u); err != ErrBlockedHost {
				t.Errorf("validateURL(%q) = %v, want ErrBlockedHost", u, err)
			}
		})
	}
}

// TestValidateURL_IPLiteralPublic covers the branch where the host is a
// parseable public IP literal, which must be allowed (returns nil at line 105).
func TestValidateURL_IPLiteralPublic(t *testing.T) {
	checker := New(5 * time.Second)

	publicURLs := []string{
		"http://8.8.8.8/",
		"http://1.1.1.1:443/health",
		"http://[2001:4860:4860::8888]/",
	}

	for _, u := range publicURLs {
		t.Run(u, func(t *testing.T) {
			if err := checker.validateURL(u); err != nil {
				t.Errorf("validateURL(%q) = %v, want nil", u, err)
			}
		})
	}
}

// TestValidateURL_EmptyHost covers the empty-host blocked branch.
func TestValidateURL_EmptyHost(t *testing.T) {
	checker := New(5 * time.Second)

	// A URL with no host (path-only) yields an empty hostname.
	if err := checker.validateURL("http:///path-only"); err != ErrBlockedHost {
		t.Errorf("expected ErrBlockedHost for empty host, got %v", err)
	}
	if err := checker.validateURL("http://localhost/"); err != ErrBlockedHost {
		t.Errorf("expected ErrBlockedHost for localhost, got %v", err)
	}
}

// TestValidateURL_ParseError covers the url.Parse error return branch.
func TestValidateURL_ParseError(t *testing.T) {
	checker := New(5 * time.Second)

	// Control characters make url.Parse fail.
	if err := checker.validateURL("http://exa\x7fmple.com"); err == nil {
		t.Error("expected parse error, got nil")
	}
}

// TestValidateURL_AllowPrivateSkips covers the early-return allowPrivate branch.
func TestValidateURL_AllowPrivateSkips(t *testing.T) {
	checker := NewWithOptions(5*time.Second, true)
	if err := checker.validateURL("http://127.0.0.1/"); err != nil {
		t.Errorf("expected nil with allowPrivate, got %v", err)
	}
}

// TestValidateURL_PublicDNSResolves exercises the LookupIP success path where a
// resolvable hostname resolves to public addresses and is allowed.
func TestValidateURL_PublicDNSResolves(t *testing.T) {
	// "localhost" is special-cased; use the loopback IP literal indirectly by
	// resolving an httptest server host (127.0.0.1) only when allowPrivate.
	// For the public-resolution branch we rely on a hostname that resolves to a
	// loopback only via /etc/hosts is not portable, so instead validate that a
	// well-known public hostname is allowed when DNS is available. We tolerate
	// DNS unavailability by skipping.
	checker := New(5 * time.Second)
	ips, err := net.LookupIP("dns.google")
	if err != nil || len(ips) == 0 {
		t.Skip("DNS not available in this environment")
	}
	// dns.google resolves to public addresses (8.8.8.8 / 8.8.4.4).
	if err := checker.validateURL("https://dns.google/"); err != nil {
		t.Errorf("expected nil for public host, got %v", err)
	}
}

// TestValidateURL_DNSResolvesPrivate covers the loop branch where a resolved IP
// is private and the host is blocked. We use a hostname that resolves to a
// loopback address. Many systems map "localhost.localdomain" or similar; we
// fall back to skipping if resolution does not yield a private IP.
func TestValidateURL_DNSResolvesPrivate(t *testing.T) {
	checker := New(5 * time.Second)

	candidates := []string{
		"127.0.0.1.nip.io", // public DNS wildcard resolving to 127.0.0.1
		"10.0.0.1.nip.io",  // resolves to 10.0.0.1
		"localhost.localdomain",
		"ip6-localhost",
		"localhost4",
	}
	for _, host := range candidates {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			continue
		}
		private := false
		for _, ip := range ips {
			if isPrivateIP(ip) {
				private = true
				break
			}
		}
		if !private {
			continue
		}
		if err := checker.validateURL("http://" + host + "/"); err != ErrBlockedHost {
			t.Errorf("validateURL(%q) = %v, want ErrBlockedHost", host, err)
		}
		return
	}
	t.Skip("no hostname resolving to a private IP available in this environment")
}

// TestCheckWithConfig_InvalidMethod covers the http.NewRequestWithContext error
// branch (an invalid HTTP method makes request construction fail). allowPrivate
// is enabled so validateURL passes and we reach NewRequestWithContext.
func TestCheckWithConfig_InvalidMethod(t *testing.T) {
	checker := NewWithOptions(5*time.Second, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:    "http://example.com/",
		Method: "INVALID METHOD WITH SPACE",
	})

	if result.Healthy {
		t.Error("expected healthy=false for invalid method")
	}
	if result.Error == nil {
		t.Error("expected error for invalid method")
	}
}

// TestCheckWithConfig_RedirectLoopUsesLastResponse exercises the CheckRedirect
// closure: after 10 redirects it returns ErrUseLastResponse, so the client
// stops following and we evaluate the final 3xx response.
func TestCheckWithConfig_RedirectLoopUsesLastResponse(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		// Always redirect to self, creating an infinite redirect chain.
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()

	checker := NewWithOptions(5*time.Second, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:          server.URL,
		ExpectStatus: "3xx",
	})

	// After the redirect cap is hit, the last 302 response is returned, so a
	// 3xx expectation is satisfied without error.
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.StatusCode != http.StatusFound {
		t.Errorf("expected final status 302, got %d", result.StatusCode)
	}
	if !result.Healthy {
		t.Error("expected healthy=true for 3xx expectation after redirect cap")
	}
	if hits < 10 {
		t.Errorf("expected the redirect loop to fire at least 10 times, got %d", hits)
	}
}

// TestCheckWithConfig_SingleRedirectFollowed exercises the CheckRedirect closure
// path where len(via) < 10 and the redirect is followed normally.
func TestCheckWithConfig_SingleRedirectFollowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/end", http.StatusFound)
	})
	mux.HandleFunc("/end", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("arrived"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	checker := NewWithOptions(5*time.Second, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:        server.URL + "/start",
		ExpectBody: "arrived",
	})

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 after following redirect, got %d", result.StatusCode)
	}
	if !result.Healthy {
		t.Error("expected healthy=true after following single redirect")
	}
}

// TestCheckWithConfig_RequestTimeout exercises the request-timeout error path
// using a delayed server and a short client timeout.
func TestCheckWithConfig_RequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewWithOptions(50*time.Millisecond, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL: server.URL,
	})

	if result.Healthy {
		t.Error("expected healthy=false on timeout")
	}
	if result.LatencyMs < 0 {
		t.Error("expected non-negative latency")
	}
}

// TestCheckStatusCode_EmptyAndInvalidParts covers the empty-part continue branch
// and the strconv.Atoi failure branch in checkStatusCode.
func TestCheckStatusCode_EmptyAndInvalidParts(t *testing.T) {
	checker := New(5 * time.Second)

	tests := []struct {
		name         string
		statusCode   int
		expectStatus string
		expected     bool
	}{
		// Empty parts between commas are skipped; 200 still matches.
		{"empty parts skipped, match", 200, "200,,", true},
		{"only empty parts", 200, ",, ,", false},
		// Non-numeric, non-wildcard parts fail Atoi and are ignored.
		{"invalid atoi part only", 200, "abc", false},
		{"invalid atoi part with valid", 200, "abc,200", true},
		{"invalid atoi no match", 404, "abc,200", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checker.checkStatusCode(tt.statusCode, tt.expectStatus); got != tt.expected {
				t.Errorf("checkStatusCode(%d, %q) = %v, want %v",
					tt.statusCode, tt.expectStatus, got, tt.expected)
			}
		})
	}
}

// TestCheckWithConfig_BodyReadFromTruncated exercises the ExpectBody path where
// the body is read and matched against a regex after a large (truncated) body.
func TestCheckWithConfig_BodyReadLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write a marker, then a large filler that pushes past nothing critical.
		w.Write([]byte("MARKER"))
		filler := make([]byte, 2048)
		for i := range filler {
			filler[i] = 'x'
		}
		w.Write(filler)
	}))
	defer server.Close()

	checker := NewWithOptions(5*time.Second, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:        server.URL,
		ExpectBody: "MARKER",
	})

	if !result.Healthy {
		t.Error("expected healthy=true when body contains MARKER")
	}
}

// TestCheckWithConfig_BodyReadError covers the io.ReadAll error branch (line
// "else { bodyOK = false }"). The server declares a large Content-Length but
// writes fewer bytes and then forcibly closes the underlying TCP connection,
// so reading the response body returns an unexpected-EOF error.
func TestCheckWithConfig_BodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("response writer does not support hijacking")
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		// Send a 200 with a Content-Length far larger than the body we write,
		// then close the connection so the client read fails mid-stream.
		bufrw.WriteString("HTTP/1.1 200 OK\r\n")
		bufrw.WriteString("Content-Length: 1000\r\n")
		bufrw.WriteString("\r\n")
		bufrw.WriteString("short") // only 5 of the promised 1000 bytes
		bufrw.Flush()
		conn.Close()
	}))
	defer server.Close()

	checker := NewWithOptions(5*time.Second, true)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL: server.URL,
		// ExpectBody forces the body to be read; status is 2xx so statusOK is
		// true and we enter the body-read block.
		ExpectBody: "short",
	})

	// The body read errors out, so bodyOK is set to false and the result is
	// unhealthy despite the matching status.
	if result.Healthy {
		t.Error("expected healthy=false when body read fails")
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("expected statusCode=200, got %d", result.StatusCode)
	}
}
