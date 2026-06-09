package domain

import (
	"errors"
	"net"
	"testing"
)

func TestIsPrivateIPDep(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"nil ip", "", false},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.x high", "10.255.255.255", true},
		{"private 172.16", "172.16.0.1", true},
		{"private 172.31", "172.31.255.255", true},
		{"public 172.32", "172.32.0.1", false},
		{"private 192.168", "192.168.1.1", true},
		{"link-local 169.254", "169.254.1.1", true},
		{"loopback 127.0.0.1", "127.0.0.1", true},
		{"loopback 127.x", "127.1.2.3", true},
		{"ipv6 loopback ::1", "::1", true},
		{"ipv6 link-local fe80", "fe80::1", true},
		{"ipv6 unique-local fc00", "fc00::1", true},
		{"ipv6 unique-local fd00", "fd12:3456:789a:1::1", true},
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 1.1.1.1", "1.1.1.1", false},
		{"public ipv6", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
				if ip == nil {
					t.Fatalf("failed to parse IP %q", tt.ip)
				}
			}
			result := isPrivateIPDep(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIPDep(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateHeartbeatURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"valid https", "https://api.example.com/health", nil},
		{"valid http", "http://api.example.com/health", nil},
		{"valid public ip", "https://8.8.8.8/health", nil},
		{"empty url", "", ErrInvalidHeartbeatURL},
		{"malformed url with control char", "http://exa\x7fmple.com", ErrInvalidHeartbeatURL},
		{"no scheme", "example.com/health", ErrInvalidHeartbeatURL},
		{"no host", "https://", ErrInvalidHeartbeatURL},
		{"ftp scheme", "ftp://files.example.com", ErrInvalidHeartbeatURL},
		{"file scheme", "file:///etc/passwd", ErrInvalidHeartbeatURL},
		{"localhost blocked", "http://localhost/health", ErrBlockedHeartbeatURL},
		{"localhost with port", "http://localhost:8080/health", ErrBlockedHeartbeatURL},
		{"private 10.x blocked", "http://10.0.0.1/health", ErrBlockedHeartbeatURL},
		{"private 192.168 blocked", "http://192.168.1.1/health", ErrBlockedHeartbeatURL},
		{"private 172.16 blocked", "http://172.16.0.1/health", ErrBlockedHeartbeatURL},
		{"loopback ip blocked", "http://127.0.0.1/health", ErrBlockedHeartbeatURL},
		{"link-local blocked", "http://169.254.0.1/health", ErrBlockedHeartbeatURL},
		{"ipv6 loopback blocked", "http://[::1]/health", ErrBlockedHeartbeatURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeartbeatURL(tt.url)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validateHeartbeatURL(%q) = %v, want nil", tt.url, err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateHeartbeatURL(%q) = %v, want %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
