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

// Check performs HTTP health check and returns true if healthy
func (c *Checker) Check(ctx context.Context, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("User-Agent", "StatusIncident-HealthChecker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, nil // Network error = unhealthy, but not an error
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}
