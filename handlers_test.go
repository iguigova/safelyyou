package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Helper to create a test server with pre-populated devices
func setupTestServer() *Server {
	store := NewStore()
	store.devices["device-1"] = &DeviceStats{ID: "device-1"}
	store.devices["device-2"] = &DeviceStats{ID: "device-2"}
	return NewServer(store, nil)
}

// TestPostHeartbeat_Success tests valid heartbeat submission
func TestPostHeartbeat_Success(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{"sent_at": "2024-01-15T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	// Verify heartbeat was recorded
	if server.store.devices["device-1"].HeartbeatCount != 1 {
		t.Error("heartbeat was not recorded")
	}
}

// TestPostHeartbeat_NotFound tests 404 for unknown device
func TestPostHeartbeat_NotFound(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{"sent_at": "2024-01-15T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/unknown-device/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	var resp ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Msg != "device not found" {
		t.Errorf("expected 'device not found', got '%s'", resp.Msg)
	}
}

// TestPostHeartbeat_InvalidJSON tests 400 for malformed JSON
func TestPostHeartbeat_InvalidJSON(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Msg != "invalid JSON" {
		t.Errorf("expected 'invalid JSON', got '%s'", resp.Msg)
	}
}

// TestPostHeartbeat_MissingSentAt tests 400 for missing sent_at field
func TestPostHeartbeat_MissingSentAt(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/heartbeat", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Msg != "sent_at is required" {
		t.Errorf("expected 'sent_at is required', got '%s'", resp.Msg)
	}
}

// TestPostStats_Success tests valid upload stat submission
func TestPostStats_Success(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{"sent_at": "2024-01-15T10:00:00Z", "upload_time": 5000000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/stats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	// Verify upload was recorded
	if server.store.devices["device-1"].UploadCount != 1 {
		t.Error("upload stat was not recorded")
	}
	if server.store.devices["device-1"].UploadTimeSum != 5*time.Second {
		t.Error("upload time was not recorded correctly")
	}
}

// TestPostStats_NotFound tests 404 for unknown device
func TestPostStats_NotFound(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{"sent_at": "2024-01-15T10:00:00Z", "upload_time": 5000000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/unknown-device/stats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestPostStats_InvalidUploadTime tests 400 for non-positive upload_time
func TestPostStats_InvalidUploadTime(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	body := `{"sent_at": "2024-01-15T10:00:00Z", "upload_time": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/stats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Msg != "upload_time must be positive" {
		t.Errorf("expected 'upload_time must be positive', got '%s'", resp.Msg)
	}
}

// TestPostStats_UploadTimeExceedsMax tests 400 for too large upload_time
func TestPostStats_UploadTimeExceedsMax(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	// 2 hours in nanoseconds (exceeds max of 1 hour)
	body := `{"sent_at": "2024-01-15T10:00:00Z", "upload_time": 7200000000000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/device-1/stats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp ErrorResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Msg != "upload_time exceeds maximum" {
		t.Errorf("expected 'upload_time exceeds maximum', got '%s'", resp.Msg)
	}
}

// TestGetStats_Success tests retrieving stats with data
func TestGetStats_Success(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	// First, add some telemetry data
	device := server.store.devices["device-1"]
	device.HeartbeatCount = 5
	device.FirstHeartbeat = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	device.LastHeartbeat = time.Date(2024, 1, 15, 10, 4, 0, 0, time.UTC)
	device.UploadCount = 2
	device.UploadTimeSum = 15 * time.Second

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/device-1/stats", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp StatsResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	// 5 heartbeats over 5 minutes = 100%
	if resp.Uptime != 100.0 {
		t.Errorf("expected uptime 100, got %f", resp.Uptime)
	}

	// 15 seconds / 2 = 7.5 seconds
	if resp.AvgUploadTime != "7.5s" {
		t.Errorf("expected avg_upload_time '7.5s', got '%s'", resp.AvgUploadTime)
	}
}

// TestGetStats_NotFound tests 404 for unknown device
func TestGetStats_NotFound(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/unknown-device/stats", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// TestGetStats_NoData tests 204 when no telemetry has been received
func TestGetStats_NoData(t *testing.T) {
	server := setupTestServer()
	router := server.Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/device-1/stats", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}
}

// TestConfigurationError tests 500 when server has configuration error
func TestConfigurationError(t *testing.T) {
	store := NewStore()
	configErr := errors.New("failed to load devices.csv")
	server := NewServer(store, configErr)
	router := server.Router()

	// Test all endpoints return 500
	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/devices/device-1/heartbeat", `{"sent_at": "2024-01-15T10:00:00Z"}`},
		{http.MethodPost, "/api/v1/devices/device-1/stats", `{"sent_at": "2024-01-15T10:00:00Z", "upload_time": 5000000000}`},
		{http.MethodGet, "/api/v1/devices/device-1/stats", ""},
	}

	for _, e := range endpoints {
		var body *bytes.Buffer
		if e.body != "" {
			body = bytes.NewBufferString(e.body)
		} else {
			body = &bytes.Buffer{}
		}
		req := httptest.NewRequest(e.method, e.path, body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("%s %s: expected status 500, got %d", e.method, e.path, rr.Code)
		}

		var resp ErrorResponse
		_ = json.NewDecoder(rr.Body).Decode(&resp)
		if resp.Msg != "server configuration error: failed to load devices.csv" {
			t.Errorf("expected configuration error message, got '%s'", resp.Msg)
		}
	}
}

// TestExtractDeviceID tests the device ID extraction from URL paths
func TestExtractDeviceID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/devices/abc-123/heartbeat", "abc-123"},
		{"/api/v1/devices/xyz-456/stats", "xyz-456"},
		{"/api/v1/devices/60-6b-44-84-dc-64/heartbeat", "60-6b-44-84-dc-64"},
		{"/invalid", ""},
	}

	for _, tc := range tests {
		result := extractDeviceID(tc.path)
		if result != tc.expected {
			t.Errorf("extractDeviceID(%s): expected '%s', got '%s'", tc.path, tc.expected, result)
		}
	}
}
