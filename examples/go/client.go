// Package statusincident provides a Go client for the Status Incident Service API.
//
// This client supports system management, incident tracking, webhooks,
// heartbeat monitoring, and SLA reporting.
//
// Example usage:
//
//	client := statusincident.NewClient("http://localhost:8080",
//	    statusincident.WithAPIKey("your-api-key"))
//
//	system, err := client.CreateSystem(ctx, &statusincident.CreateSystemRequest{
//	    Name:        "API Gateway",
//	    Description: "Main API Gateway",
//	})
package statusincident

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the Status Incident API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	username   string
	password   string
}

// ClientOption configures the client.
type ClientOption func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithBasicAuth sets basic auth credentials.
func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new Status Incident client.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ==================== Request Helpers ====================

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	u := c.baseURL + "/api" + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	} else if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.ContentLength != 0 {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) get(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	return c.doRequest(ctx, http.MethodGet, endpoint, nil, result)
}

func (c *Client) post(ctx context.Context, endpoint string, body, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, endpoint, body, result)
}

func (c *Client) put(ctx context.Context, endpoint string, body, result interface{}) error {
	return c.doRequest(ctx, http.MethodPut, endpoint, body, result)
}

func (c *Client) delete(ctx context.Context, endpoint string) error {
	return c.doRequest(ctx, http.MethodDelete, endpoint, nil, nil)
}

// ==================== Types ====================

// System represents a monitored system.
type System struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
	SLATarget   float64   `json:"sla_target,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Dependency represents a system component/dependency.
type Dependency struct {
	ID                int       `json:"id"`
	SystemID          int       `json:"system_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	HeartbeatURL      string    `json:"heartbeat_url,omitempty"`
	HeartbeatInterval int       `json:"heartbeat_interval,omitempty"`
	LastCheckAt       time.Time `json:"last_check_at,omitempty"`
	LastLatencyMs     int       `json:"last_latency_ms,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// Incident represents an incident.
type Incident struct {
	ID         int        `json:"id"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	SystemIDs  []int      `json:"system_ids"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// Webhook represents a webhook configuration.
type Webhook struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Type      string   `json:"type"`
	Events    []string `json:"events"`
	SystemIDs []int    `json:"system_ids"`
	Enabled   bool     `json:"enabled"`
}

// Maintenance represents a maintenance window.
type Maintenance struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	SystemIDs   []int     `json:"system_ids"`
	Status      string    `json:"status"`
}

// ==================== Request/Response Types ====================

// CreateSystemRequest is the request to create a system.
type CreateSystemRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

// UpdateStatusRequest is the request to update status.
type UpdateStatusRequest struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// CreateDependencyRequest is the request to create a dependency.
type CreateDependencyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// HeartbeatConfigRequest configures heartbeat monitoring.
type HeartbeatConfigRequest struct {
	URL          string            `json:"url"`
	Interval     int               `json:"interval,omitempty"`
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpectStatus string            `json:"expect_status,omitempty"`
	ExpectBody   string            `json:"expect_body,omitempty"`
}

// CreateIncidentRequest is the request to create an incident.
type CreateIncidentRequest struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Severity  string `json:"severity,omitempty"`
	SystemIDs []int  `json:"system_ids,omitempty"`
}

// IncidentUpdateRequest adds an update to an incident.
type IncidentUpdateRequest struct {
	Message string `json:"message"`
	By      string `json:"by,omitempty"`
}

// ResolveIncidentRequest resolves an incident.
type ResolveIncidentRequest struct {
	Message    string `json:"message,omitempty"`
	Postmortem string `json:"postmortem,omitempty"`
}

// CreateWebhookRequest creates a webhook.
type CreateWebhookRequest struct {
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Type      string   `json:"type,omitempty"`
	Events    []string `json:"events,omitempty"`
	SystemIDs []int    `json:"system_ids,omitempty"`
	Enabled   bool     `json:"enabled"`
}

// CreateMaintenanceRequest schedules maintenance.
type CreateMaintenanceRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	SystemIDs   []int     `json:"system_ids,omitempty"`
}

// SLAReportRequest generates an SLA report.
type SLAReportRequest struct {
	Title       string `json:"title"`
	Period      string `json:"period,omitempty"`
	GeneratedBy string `json:"generated_by,omitempty"`
}

// ==================== Systems API ====================

// ListSystems returns all systems.
func (c *Client) ListSystems(ctx context.Context) ([]System, error) {
	var systems []System
	if err := c.get(ctx, "/systems", nil, &systems); err != nil {
		return nil, err
	}
	return systems, nil
}

// GetSystem returns a system by ID.
func (c *Client) GetSystem(ctx context.Context, id int) (*System, error) {
	var system System
	if err := c.get(ctx, fmt.Sprintf("/systems/%d", id), nil, &system); err != nil {
		return nil, err
	}
	return &system, nil
}

// CreateSystem creates a new system.
func (c *Client) CreateSystem(ctx context.Context, req *CreateSystemRequest) (*System, error) {
	var system System
	if err := c.post(ctx, "/systems", req, &system); err != nil {
		return nil, err
	}
	return &system, nil
}

// UpdateSystem updates a system.
func (c *Client) UpdateSystem(ctx context.Context, id int, req *CreateSystemRequest) (*System, error) {
	var system System
	if err := c.put(ctx, fmt.Sprintf("/systems/%d", id), req, &system); err != nil {
		return nil, err
	}
	return &system, nil
}

// DeleteSystem deletes a system.
func (c *Client) DeleteSystem(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/systems/%d", id))
}

// UpdateSystemStatus updates the status of a system.
func (c *Client) UpdateSystemStatus(ctx context.Context, id int, req *UpdateStatusRequest) (*System, error) {
	var system System
	if err := c.post(ctx, fmt.Sprintf("/systems/%d/status", id), req, &system); err != nil {
		return nil, err
	}
	return &system, nil
}

// ==================== Dependencies API ====================

// ListDependencies returns all dependencies of a system.
func (c *Client) ListDependencies(ctx context.Context, systemID int) ([]Dependency, error) {
	var deps []Dependency
	if err := c.get(ctx, fmt.Sprintf("/systems/%d/dependencies", systemID), nil, &deps); err != nil {
		return nil, err
	}
	return deps, nil
}

// CreateDependency creates a new dependency.
func (c *Client) CreateDependency(ctx context.Context, systemID int, req *CreateDependencyRequest) (*Dependency, error) {
	var dep Dependency
	if err := c.post(ctx, fmt.Sprintf("/systems/%d/dependencies", systemID), req, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// UpdateDependencyStatus updates the status of a dependency.
func (c *Client) UpdateDependencyStatus(ctx context.Context, id int, req *UpdateStatusRequest) (*Dependency, error) {
	var dep Dependency
	if err := c.post(ctx, fmt.Sprintf("/dependencies/%d/status", id), req, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// DeleteDependency deletes a dependency.
func (c *Client) DeleteDependency(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/dependencies/%d", id))
}

// ConfigureHeartbeat sets up automatic health checking.
func (c *Client) ConfigureHeartbeat(ctx context.Context, depID int, req *HeartbeatConfigRequest) (*Dependency, error) {
	var dep Dependency
	if err := c.post(ctx, fmt.Sprintf("/dependencies/%d/heartbeat", depID), req, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// DisableHeartbeat disables heartbeat monitoring.
func (c *Client) DisableHeartbeat(ctx context.Context, depID int) error {
	return c.delete(ctx, fmt.Sprintf("/dependencies/%d/heartbeat", depID))
}

// ForceCheck triggers an immediate health check.
func (c *Client) ForceCheck(ctx context.Context, depID int) (*Dependency, error) {
	var dep Dependency
	if err := c.post(ctx, fmt.Sprintf("/dependencies/%d/check", depID), nil, &dep); err != nil {
		return nil, err
	}
	return &dep, nil
}

// ==================== Incidents API ====================

// ListIncidents returns incidents with optional filters.
func (c *Client) ListIncidents(ctx context.Context, status, severity string) ([]Incident, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if severity != "" {
		params.Set("severity", severity)
	}
	var incidents []Incident
	if err := c.get(ctx, "/incidents", params, &incidents); err != nil {
		return nil, err
	}
	return incidents, nil
}

// GetIncident returns an incident by ID.
func (c *Client) GetIncident(ctx context.Context, id int) (*Incident, error) {
	var incident Incident
	if err := c.get(ctx, fmt.Sprintf("/incidents/%d", id), nil, &incident); err != nil {
		return nil, err
	}
	return &incident, nil
}

// CreateIncident creates a new incident.
func (c *Client) CreateIncident(ctx context.Context, req *CreateIncidentRequest) (*Incident, error) {
	var incident Incident
	if err := c.post(ctx, "/incidents", req, &incident); err != nil {
		return nil, err
	}
	return &incident, nil
}

// AddIncidentUpdate adds an update to an incident.
func (c *Client) AddIncidentUpdate(ctx context.Context, id int, req *IncidentUpdateRequest) error {
	return c.post(ctx, fmt.Sprintf("/incidents/%d/updates", id), req, nil)
}

// ResolveIncident resolves an incident.
func (c *Client) ResolveIncident(ctx context.Context, id int, req *ResolveIncidentRequest) (*Incident, error) {
	var incident Incident
	if err := c.post(ctx, fmt.Sprintf("/incidents/%d/resolve", id), req, &incident); err != nil {
		return nil, err
	}
	return &incident, nil
}

// ==================== Webhooks API ====================

// ListWebhooks returns all webhooks.
func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var webhooks []Webhook
	if err := c.get(ctx, "/webhooks", nil, &webhooks); err != nil {
		return nil, err
	}
	return webhooks, nil
}

// CreateWebhook creates a new webhook.
func (c *Client) CreateWebhook(ctx context.Context, req *CreateWebhookRequest) (*Webhook, error) {
	var webhook Webhook
	if err := c.post(ctx, "/webhooks", req, &webhook); err != nil {
		return nil, err
	}
	return &webhook, nil
}

// DeleteWebhook deletes a webhook.
func (c *Client) DeleteWebhook(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/webhooks/%d", id))
}

// TestWebhook sends a test notification.
func (c *Client) TestWebhook(ctx context.Context, id int) error {
	return c.post(ctx, fmt.Sprintf("/webhooks/%d/test", id), nil, nil)
}

// ==================== Maintenance API ====================

// ListMaintenances returns all maintenance windows.
func (c *Client) ListMaintenances(ctx context.Context) ([]Maintenance, error) {
	var maintenances []Maintenance
	if err := c.get(ctx, "/maintenances", nil, &maintenances); err != nil {
		return nil, err
	}
	return maintenances, nil
}

// CreateMaintenance schedules a maintenance window.
func (c *Client) CreateMaintenance(ctx context.Context, req *CreateMaintenanceRequest) (*Maintenance, error) {
	var maintenance Maintenance
	if err := c.post(ctx, "/maintenances", req, &maintenance); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// CancelMaintenance cancels a scheduled maintenance.
func (c *Client) CancelMaintenance(ctx context.Context, id int) error {
	return c.delete(ctx, fmt.Sprintf("/maintenances/%d", id))
}

// ==================== SLA API ====================

// GenerateSLAReport generates an SLA report.
func (c *Client) GenerateSLAReport(ctx context.Context, req *SLAReportRequest) (map[string]interface{}, error) {
	var report map[string]interface{}
	if err := c.post(ctx, "/sla/reports", req, &report); err != nil {
		return nil, err
	}
	return report, nil
}

// ==================== Analytics API ====================

// GetAnalytics returns overall analytics.
func (c *Client) GetAnalytics(ctx context.Context, period string) (map[string]interface{}, error) {
	params := url.Values{}
	if period != "" {
		params.Set("period", period)
	}
	var analytics map[string]interface{}
	if err := c.get(ctx, "/analytics", params, &analytics); err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetSystemAnalytics returns analytics for a specific system.
func (c *Client) GetSystemAnalytics(ctx context.Context, systemID int, period string) (map[string]interface{}, error) {
	params := url.Values{}
	if period != "" {
		params.Set("period", period)
	}
	var analytics map[string]interface{}
	if err := c.get(ctx, fmt.Sprintf("/systems/%d/analytics", systemID), params, &analytics); err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetLogs returns change logs.
func (c *Client) GetLogs(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var logs []map[string]interface{}
	if err := c.get(ctx, "/logs", params, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// ==================== Export/Import API ====================

// ExportData exports all data.
func (c *Client) ExportData(ctx context.Context) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := c.get(ctx, "/export", nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ImportData imports data from a backup.
func (c *Client) ImportData(ctx context.Context, data map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.post(ctx, "/import", data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
