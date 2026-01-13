package http_checker

import (
	"context"
	"net/http"
	"time"
)

// Checker implements domain.HealthChecker
type Checker struct {
	client *http.Client
}

// New creates a new HTTP health checker
func New(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow up to 10 redirects
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// Check performs HTTP health check and returns healthy status and response time in milliseconds
func (c *Checker) Check(ctx context.Context, url string) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0, err
	}

	req.Header.Set("User-Agent", "StatusIncident-HealthChecker/1.0")

	start := time.Now()
	resp, err := c.client.Do(req)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return false, latencyMs, nil // Network error = unhealthy, but not an error
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 300, latencyMs, nil
}
