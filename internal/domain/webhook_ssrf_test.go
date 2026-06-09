package domain

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"nil ip", "", false},
		{"private 10.x", "10.1.2.3", true},
		{"private 172.16", "172.16.5.5", true},
		{"private 172.31", "172.31.0.1", true},
		{"public 172.15", "172.15.0.1", false},
		{"public 172.32", "172.32.0.1", false},
		{"private 192.168", "192.168.0.1", true},
		{"link-local 169.254", "169.254.169.254", true},
		{"loopback 127.0.0.1", "127.0.0.1", true},
		{"ipv6 loopback ::1", "::1", true},
		{"ipv6 link-local fe80", "fe80::abcd", true},
		{"ipv6 unique-local fc00", "fc00::1", true},
		{"ipv6 unique-local fd00", "fd00::1234", true},
		{"public 8.8.8.8", "8.8.8.8", false},
		{"public 9.9.9.9", "9.9.9.9", false},
		{"public ipv6", "2606:4700:4700::1111", false},
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
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantErr     bool
		errContains string
	}{
		{"valid https", "https://example.com/webhook", false, ""},
		{"valid http", "http://example.com/webhook", false, ""},
		{"valid public ip", "https://8.8.8.8/hook", false, ""},
		{"invalid relative url", "not-a-url", true, "invalid webhook URL"},
		{"localhost blocked", "http://localhost/webhook", true, "localhost"},
		{"localhost with port blocked", "http://localhost:9000/webhook", true, "localhost"},
		{"private 10.x blocked", "http://10.0.0.5/hook", true, "private IP"},
		{"private 192.168 blocked", "https://192.168.1.10/hook", true, "private IP"},
		{"loopback ip blocked", "http://127.0.0.1/hook", true, "private IP"},
		{"link-local blocked", "http://169.254.1.1/hook", true, "private IP"},
		{"ipv6 loopback blocked", "http://[::1]/hook", true, "private IP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if !tt.wantErr {
				if err != nil {
					t.Errorf("validateWebhookURL(%q) = %v, want nil", tt.url, err)
				}
				return
			}
			if err == nil {
				t.Errorf("validateWebhookURL(%q) = nil, want error", tt.url)
				return
			}
			if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
				t.Errorf("validateWebhookURL(%q) error %q should contain %q", tt.url, err.Error(), tt.errContains)
			}
		})
	}
}
