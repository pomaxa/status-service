package http_checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestNew(t *testing.T) {
	checker := New(10 * time.Second)
	if checker == nil {
		t.Fatal("expected non-nil checker")
	}
	if checker.client == nil {
		t.Fatal("expected non-nil client")
	}
	if checker.client.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", checker.client.Timeout)
	}
}

func TestCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	healthy, latency, err := checker.Check(context.Background(), server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !healthy {
		t.Error("expected healthy=true")
	}
	if latency < 0 {
		t.Error("expected non-negative latency")
	}
}

func TestCheck_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	healthy, _, err := checker.Check(context.Background(), server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if healthy {
		t.Error("expected healthy=false for 500 status")
	}
}

func TestCheckWithConfig_Methods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedMethod string
	}{
		{"GET method", "GET", "GET"},
		{"POST method", "POST", "POST"},
		{"PUT method", "PUT", "PUT"},
		{"HEAD method", "HEAD", "HEAD"},
		{"Empty defaults to GET", "", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			checker := New(5 * time.Second)
			result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
				URL:    server.URL,
				Method: tt.method,
			})

			if !result.Healthy {
				t.Error("expected healthy=true")
			}
			if receivedMethod != tt.expectedMethod {
				t.Errorf("expected method %s, got %s", tt.expectedMethod, receivedMethod)
			}
		})
	}
}

func TestCheckWithConfig_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:    server.URL,
		Method: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "custom-value",
		},
	})

	if !result.Healthy {
		t.Error("expected healthy=true")
	}
	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("expected Authorization header, got %s", receivedHeaders.Get("Authorization"))
	}
	if receivedHeaders.Get("X-Custom") != "custom-value" {
		t.Errorf("expected X-Custom header, got %s", receivedHeaders.Get("X-Custom"))
	}
	if receivedHeaders.Get("User-Agent") != "StatusIncident-HealthChecker/1.0" {
		t.Errorf("expected User-Agent header, got %s", receivedHeaders.Get("User-Agent"))
	}
}

func TestCheckWithConfig_RequestBody(t *testing.T) {
	var receivedBody string
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:    server.URL,
		Method: "POST",
		Body:   `{"check": "deep"}`,
	})

	if !result.Healthy {
		t.Error("expected healthy=true")
	}
	if receivedBody != `{"check": "deep"}` {
		t.Errorf("expected body '{\"check\": \"deep\"}', got '%s'", receivedBody)
	}
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}
}

func TestCheckWithConfig_CustomContentType(t *testing.T) {
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:    server.URL,
		Method: "POST",
		Body:   "<xml>data</xml>",
		Headers: map[string]string{
			"Content-Type": "application/xml",
		},
	})

	if receivedContentType != "application/xml" {
		t.Errorf("expected Content-Type application/xml, got %s", receivedContentType)
	}
}

func TestCheckStatusCode(t *testing.T) {
	checker := New(5 * time.Second)

	tests := []struct {
		name         string
		statusCode   int
		expectStatus string
		expected     bool
	}{
		// Default behavior (empty expect_status = 2xx)
		{"200 with empty expect", 200, "", true},
		{"201 with empty expect", 201, "", true},
		{"204 with empty expect", 204, "", true},
		{"299 with empty expect", 299, "", true},
		{"300 with empty expect", 300, "", false},
		{"500 with empty expect", 500, "", false},

		// Exact status codes
		{"200 expect 200", 200, "200", true},
		{"201 expect 200", 201, "200", false},
		{"200 expect 200,201", 200, "200,201", true},
		{"201 expect 200,201", 201, "200,201", true},
		{"202 expect 200,201", 202, "200,201", false},
		{"200 expect 200,201,204", 200, "200,201,204", true},
		{"204 expect 200,201,204", 204, "200,201,204", true},

		// Wildcard patterns
		{"200 expect 2xx", 200, "2xx", true},
		{"201 expect 2xx", 201, "2xx", true},
		{"299 expect 2xx", 299, "2xx", true},
		{"300 expect 2xx", 300, "2xx", false},
		{"300 expect 3xx", 300, "3xx", true},
		{"301 expect 3xx", 301, "3xx", true},
		{"400 expect 4xx", 400, "4xx", true},
		{"500 expect 5xx", 500, "5xx", true},

		// Mixed patterns
		{"200 expect 2xx,3xx", 200, "2xx,3xx", true},
		{"301 expect 2xx,3xx", 301, "2xx,3xx", true},
		{"400 expect 2xx,3xx", 400, "2xx,3xx", false},
		{"200 expect 200,3xx", 200, "200,3xx", true},
		{"301 expect 200,3xx", 301, "200,3xx", true},

		// Edge cases
		{"200 with spaces", 200, " 200 , 201 ", true},
		{"201 with spaces", 201, " 200 , 201 ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.checkStatusCode(tt.statusCode, tt.expectStatus)
			if result != tt.expected {
				t.Errorf("checkStatusCode(%d, %q) = %v, want %v",
					tt.statusCode, tt.expectStatus, result, tt.expected)
			}
		})
	}
}

func TestCheckBodyRegex(t *testing.T) {
	checker := New(5 * time.Second)

	tests := []struct {
		name     string
		body     string
		pattern  string
		expected bool
	}{
		// Empty pattern always passes
		{"empty pattern", "anything", "", true},

		// Simple patterns
		{"contains ok", `{"status": "ok"}`, "ok", true},
		{"contains healthy", `{"healthy": true}`, "healthy", true},
		{"does not contain", `{"status": "error"}`, "ok", false},

		// JSON patterns
		{"json status ok", `{"status": "ok"}`, `"status":\s*"ok"`, true},
		{"json status error", `{"status": "error"}`, `"status":\s*"ok"`, false},
		{"json healthy true", `{"healthy": true}`, `"healthy":\s*true`, true},
		{"json healthy false", `{"healthy": false}`, `"healthy":\s*true`, false},

		// Complex patterns
		{"multiline json", "{\n  \"status\": \"ok\"\n}", `"status":\s*"ok"`, true},
		{"version pattern", `{"version": "1.2.3"}`, `"version":\s*"\d+\.\d+\.\d+"`, true},

		// Invalid regex
		{"invalid regex", "test", "[invalid", false},

		// Edge cases
		{"empty body with pattern", "", "test", false},
		{"body with special chars", `{"msg": "test [1]"}`, `test \[1\]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.checkBodyRegex(tt.body, tt.pattern)
			if result != tt.expected {
				t.Errorf("checkBodyRegex(%q, %q) = %v, want %v",
					tt.body, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestCheckWithConfig_ExpectStatus(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		expectStatus string
		wantHealthy  bool
	}{
		{"200 expect 2xx", 200, "2xx", true},
		{"500 expect 2xx", 500, "2xx", false},
		{"201 expect 200,201", 201, "200,201", true},
		{"500 expect 200,201", 500, "200,201", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			checker := New(5 * time.Second)
			result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
				URL:          server.URL,
				ExpectStatus: tt.expectStatus,
			})

			if result.Healthy != tt.wantHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.wantHealthy, result.Healthy)
			}
			if result.StatusCode != tt.serverStatus {
				t.Errorf("expected statusCode=%d, got %d", tt.serverStatus, result.StatusCode)
			}
		})
	}
}

func TestCheckWithConfig_ExpectBody(t *testing.T) {
	tests := []struct {
		name        string
		serverBody  string
		expectBody  string
		wantHealthy bool
	}{
		{"match simple", `{"status": "ok"}`, "ok", true},
		{"match regex", `{"status": "ok"}`, `"status":\s*"ok"`, true},
		{"no match", `{"status": "error"}`, `"status":\s*"ok"`, false},
		{"empty expect", `{"anything": true}`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			checker := New(5 * time.Second)
			result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
				URL:        server.URL,
				ExpectBody: tt.expectBody,
			})

			if result.Healthy != tt.wantHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.wantHealthy, result.Healthy)
			}
		})
	}
}

func TestCheckWithConfig_NetworkError(t *testing.T) {
	checker := New(1 * time.Second)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL: "http://localhost:99999", // Invalid port
	})

	if result.Healthy {
		t.Error("expected healthy=false for network error")
	}
	// Latency may be 0 if connection refused immediately
	if result.LatencyMs < 0 {
		t.Error("expected non-negative latency")
	}
}

func TestCheckWithConfig_InvalidURL(t *testing.T) {
	checker := New(5 * time.Second)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL: "://invalid",
	})

	if result.Healthy {
		t.Error("expected healthy=false for invalid URL")
	}
	if result.Error == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestCheckWithConfig_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := checker.CheckWithConfig(ctx, domain.HeartbeatConfig{
		URL: server.URL,
	})

	if result.Healthy {
		t.Error("expected healthy=false for cancelled context")
	}
}

func TestCheckWithConfig_FullIntegration(t *testing.T) {
	// Create a server that validates all aspects of the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate method
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Validate authorization header
		if r.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Validate content type
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"healthy": true, "version": "1.0.0"}`))
	}))
	defer server.Close()

	checker := New(5 * time.Second)
	result := checker.CheckWithConfig(context.Background(), domain.HeartbeatConfig{
		URL:      server.URL,
		Method:   "POST",
		Headers:  map[string]string{"Authorization": "Bearer secret"},
		Body:     `{"check": "full"}`,
		ExpectStatus: "200",
		ExpectBody:   `"healthy":\s*true`,
	})

	if !result.Healthy {
		t.Error("expected healthy=true for full integration test")
	}
	if result.StatusCode != 200 {
		t.Errorf("expected statusCode=200, got %d", result.StatusCode)
	}
}
