package ssrf

import (
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},     // loopback
		{"10.1.2.3", true},      // RFC1918 10/8
		{"172.16.5.5", true},    // RFC1918 172.16/12
		{"192.168.1.1", true},   // RFC1918 192.168/16
		{"169.254.1.1", true},   // link-local
		{"0.0.0.0", true},       // unspecified (regression: was allowed)
		{"::", true},            // IPv6 unspecified
		{"::1", true},           // IPv6 loopback
		{"100.64.0.1", true},    // CGNAT
		{"224.0.0.1", true},     // multicast
		{"fe80::1", true},       // IPv6 link-local
		{"fc00::1", true},       // IPv6 ULA
		{"8.8.8.8", false},      // public
		{"1.1.1.1", false},      // public
		{"203.0.113.10", false}, // public (TEST-NET-3, routable shape)
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := IsPrivateIP(ip); got != tt.want {
			t.Errorf("IsPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
	if IsPrivateIP(nil) {
		t.Error("IsPrivateIP(nil) should be false")
	}
}

// TestDialControl verifies the connect-time guard without opening sockets: it
// is a pure function of the post-resolution address string.
func TestDialControl(t *testing.T) {
	blocked := []string{"127.0.0.1:80", "169.254.169.254:80", "10.0.0.5:443", "[::1]:8080", "0.0.0.0:80"}
	for _, addr := range blocked {
		if err := dialControl("tcp", addr, nil); err == nil {
			t.Errorf("dialControl(%q) = nil, want blocked", addr)
		}
	}
	allowed := []string{"8.8.8.8:443", "1.1.1.1:80", "[2606:4700:4700::1111]:443"}
	for _, addr := range allowed {
		if err := dialControl("tcp", addr, nil); err != nil {
			t.Errorf("dialControl(%q) = %v, want allowed", addr, err)
		}
	}
	// A non-IP host (resolution should have produced an IP; a literal name is invalid)
	if err := dialControl("tcp", "example.com:80", nil); err == nil {
		t.Error("dialControl with non-IP host should be blocked")
	}
}

func TestRedirectGuard(t *testing.T) {
	validate := func(rawURL string) error {
		u, _ := url.Parse(rawURL)
		if u.Hostname() == "169.254.169.254" {
			return ErrBlockedHost
		}
		return nil
	}
	guard := RedirectGuard(10, validate)

	// Hop to a blocked metadata host is rejected.
	req, _ := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/", nil)
	if err := guard(req, nil); err == nil {
		t.Error("RedirectGuard should reject redirect to blocked host")
	}

	// Hop to a public host is allowed.
	req2, _ := http.NewRequest("GET", "http://example.com/next", nil)
	if err := guard(req2, nil); err != nil {
		t.Errorf("RedirectGuard rejected allowed host: %v", err)
	}

	// Hop limit is enforced regardless of host.
	via := make([]*http.Request, 10)
	if err := guard(req2, via); err != http.ErrUseLastResponse {
		t.Errorf("RedirectGuard at hop limit = %v, want ErrUseLastResponse", err)
	}
}
