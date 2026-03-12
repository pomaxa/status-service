package http_checker

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"status-incident/internal/domain"
)

var (
	// ErrBlockedHost is returned when a URL targets a blocked host
	ErrBlockedHost = errors.New("access to internal/private IP addresses is not allowed")
)

// Checker implements domain.HealthChecker
type Checker struct {
	client       *http.Client
	allowPrivate bool
}

// New creates a new HTTP health checker with SSRF protection enabled
func New(timeout time.Duration) *Checker {
	return NewWithOptions(timeout, false)
}

// NewWithOptions creates a new HTTP health checker with configurable options
func NewWithOptions(timeout time.Duration, allowPrivate bool) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		allowPrivate: allowPrivate,
	}
}

// isPrivateIP checks if an IP address is private/internal
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"127.0.0.0/8",
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

// validateURL checks if a URL is safe to access (not targeting internal resources)
func (c *Checker) validateURL(rawURL string) error {
	if c.allowPrivate {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	host := parsed.Hostname()

	if host == "localhost" || host == "" {
		return ErrBlockedHost
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return ErrBlockedHost
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return ErrBlockedHost
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return ErrBlockedHost
		}
	}

	return nil
}

// Check performs HTTP health check and returns healthy status and response time in milliseconds
// This is the legacy method for backward compatibility
func (c *Checker) Check(ctx context.Context, url string) (bool, int64, error) {
	result := c.CheckWithConfig(ctx, domain.HeartbeatConfig{URL: url, Method: "GET"})
	return result.Healthy, result.LatencyMs, result.Error
}

// CheckWithConfig performs HTTP health check with advanced configuration
func (c *Checker) CheckWithConfig(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
	if err := c.validateURL(config.URL); err != nil {
		return domain.HealthCheckResult{Healthy: false, Error: err}
	}

	method := config.Method
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if config.Body != "" && (method == "POST" || method == "PUT") {
		body = strings.NewReader(config.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, config.URL, body)
	if err != nil {
		return domain.HealthCheckResult{Healthy: false, Error: err}
	}

	// Set default User-Agent
	req.Header.Set("User-Agent", "StatusIncident-HealthChecker/1.0")

	// Set custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Set Content-Type for POST/PUT with body
	if config.Body != "" && (method == "POST" || method == "PUT") {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return domain.HealthCheckResult{Healthy: false, LatencyMs: latencyMs}
	}
	defer resp.Body.Close()

	// Check status code
	statusOK := c.checkStatusCode(resp.StatusCode, config.ExpectStatus)

	// Check response body regex if configured
	bodyOK := true
	if config.ExpectBody != "" && statusOK {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
		if err == nil {
			bodyOK = c.checkBodyRegex(string(bodyBytes), config.ExpectBody)
		} else {
			bodyOK = false
		}
	}

	return domain.HealthCheckResult{
		Healthy:    statusOK && bodyOK,
		LatencyMs:  latencyMs,
		StatusCode: resp.StatusCode,
	}
}

// checkStatusCode checks if the status code matches the expected pattern
// Supports: "200", "200,201,204", "2xx", "2xx,3xx"
func (c *Checker) checkStatusCode(statusCode int, expectStatus string) bool {
	if expectStatus == "" {
		// Default: accept 2xx status codes
		return statusCode >= 200 && statusCode < 300
	}

	parts := strings.Split(expectStatus, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check wildcard format like "2xx"
		if len(part) == 3 && part[1] == 'x' && part[2] == 'x' {
			prefix := int(part[0] - '0')
			if statusCode >= prefix*100 && statusCode < (prefix+1)*100 {
				return true
			}
			continue
		}

		// Check exact status code
		if code, err := strconv.Atoi(part); err == nil {
			if statusCode == code {
				return true
			}
		}
	}

	return false
}

// checkBodyRegex checks if the response body matches the regex pattern
func (c *Checker) checkBodyRegex(body, pattern string) bool {
	if pattern == "" {
		return true
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		// Invalid regex pattern - treat as failure
		return false
	}

	return re.MatchString(body)
}
