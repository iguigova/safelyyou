package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

// Request types

type HeartbeatRequest struct {
	SentAt time.Time `json:"sent_at"`
}

type UploadStatRequest struct {
	SentAt     time.Time `json:"sent_at"`
	UploadTime int64     `json:"upload_time"` // nanoseconds
}

// Response types

type StatsResponse struct {
	Uptime        float64 `json:"uptime"`
	AvgUploadTime string  `json:"avg_upload_time"`
}

type ErrorResponse struct {
	Msg string `json:"msg"`
}

// Server holds dependencies for HTTP handlers.
type Server struct {
	store      *Store
	configErr  error // Set if CSV loading failed
}

// NewServer creates a new server with the given store.
func NewServer(store *Store, configErr error) *Server {
	return &Server{
		store:     store,
		configErr: configErr,
	}
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Msg: msg})
}

// extractDeviceID extracts the device ID from a URL path.
// Expected format: /api/v1/devices/{device_id}/heartbeat or /api/v1/devices/{device_id}/stats
func extractDeviceID(path string) string {
	parts := strings.Split(path, "/")
	// /api/v1/devices/{device_id}/endpoint -> ["", "api", "v1", "devices", "{device_id}", "endpoint"]
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// Validation

const maxUploadTime = int64(time.Hour) // 1 hour max for upload time

func validateHeartbeatRequest(req *HeartbeatRequest) error {
	if req.SentAt.IsZero() {
		return errors.New("sent_at is required")
	}
	if req.SentAt.After(time.Now().Add(time.Minute)) { // Allow 1 minute clock skew
		return errors.New("sent_at cannot be in the future")
	}
	return nil
}

func validateUploadStatRequest(req *UploadStatRequest) error {
	// Note: sent_at is optional for stats (simulator sends zero time)
	if req.UploadTime <= 0 {
		return errors.New("upload_time must be positive")
	}
	if req.UploadTime > maxUploadTime {
		return errors.New("upload_time exceeds maximum")
	}
	return nil
}

// Handlers

// HandleHeartbeat processes POST /api/v1/devices/{device_id}/heartbeat
func (s *Server) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	// Check for configuration error
	if s.configErr != nil {
		log.Printf("[ERROR] Configuration error: %v", s.configErr)
		writeError(w, http.StatusInternalServerError, "server configuration error: "+s.configErr.Error())
		return
	}

	deviceID := extractDeviceID(r.URL.Path)
	log.Printf("[REQUEST] POST /api/v1/devices/%s/heartbeat", deviceID)

	// Check if device exists
	if !s.store.DeviceExists(deviceID) {
		log.Printf("[WARN] Device not found: %s", deviceID)
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	// Parse request body
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Invalid JSON: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate request
	if err := validateHeartbeatRequest(&req); err != nil {
		log.Printf("[ERROR] Validation failed: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Record heartbeat
	s.store.RecordHeartbeat(deviceID, req.SentAt)
	w.WriteHeader(http.StatusNoContent)
}

// HandlePostStats processes POST /api/v1/devices/{device_id}/stats
func (s *Server) HandlePostStats(w http.ResponseWriter, r *http.Request) {
	// Check for configuration error
	if s.configErr != nil {
		log.Printf("[ERROR] Configuration error: %v", s.configErr)
		writeError(w, http.StatusInternalServerError, "server configuration error: "+s.configErr.Error())
		return
	}

	deviceID := extractDeviceID(r.URL.Path)
	log.Printf("[REQUEST] POST /api/v1/devices/%s/stats", deviceID)

	// Check if device exists
	if !s.store.DeviceExists(deviceID) {
		log.Printf("[WARN] Device not found: %s", deviceID)
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	// Parse request body
	var req UploadStatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[ERROR] Invalid JSON: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate request
	if err := validateUploadStatRequest(&req); err != nil {
		log.Printf("[ERROR] Validation failed: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Record upload stat
	s.store.RecordUploadStat(deviceID, time.Duration(req.UploadTime))
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetStats processes GET /api/v1/devices/{device_id}/stats
func (s *Server) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	// Check for configuration error
	if s.configErr != nil {
		log.Printf("[ERROR] Configuration error: %v", s.configErr)
		writeError(w, http.StatusInternalServerError, "server configuration error: "+s.configErr.Error())
		return
	}

	deviceID := extractDeviceID(r.URL.Path)
	log.Printf("[REQUEST] GET /api/v1/devices/%s/stats", deviceID)

	// Get stats
	result, exists := s.store.GetStats(deviceID)
	if !exists {
		log.Printf("[WARN] Device not found: %s", deviceID)
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	// If no data collected yet, return 204
	if !result.HasHeartbeats && !result.HasUploads {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Build response
	resp := StatsResponse{
		Uptime:        result.Uptime,
		AvgUploadTime: result.AvgUploadTime.String(),
	}

	writeJSON(w, http.StatusOK, resp)
}

// Router routes requests to the appropriate handler.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// The Go HTTP mux doesn't support path parameters, so we need to handle routing manually
	mux.HandleFunc("/api/v1/devices/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Determine which endpoint based on path suffix and method
		if strings.HasSuffix(path, "/heartbeat") && r.Method == http.MethodPost {
			s.HandleHeartbeat(w, r)
			return
		}

		if strings.HasSuffix(path, "/stats") {
			switch r.Method {
			case http.MethodPost:
				s.HandlePostStats(w, r)
				return
			case http.MethodGet:
				s.HandleGetStats(w, r)
				return
			}
		}

		// Method not allowed or unknown endpoint
		http.NotFound(w, r)
	})

	return mux
}
