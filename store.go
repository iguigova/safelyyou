package main

import (
	"encoding/csv"
	"os"
	"sync"
	"time"
)

// DeviceStats holds aggregated telemetry data for a single device.
// Memory usage is O(1) per device (~100 bytes), regardless of how long the server runs.
type DeviceStats struct {
	ID string

	// Heartbeat aggregates
	HeartbeatCount int64
	FirstHeartbeat time.Time
	LastHeartbeat  time.Time

	// Upload aggregates
	UploadCount   int64
	UploadTimeSum time.Duration
}

// Store provides thread-safe access to device statistics.
// Uses sync.RWMutex to allow concurrent reads while ensuring exclusive writes.
type Store struct {
	mu      sync.RWMutex
	devices map[string]*DeviceStats
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{
		devices: make(map[string]*DeviceStats),
	}
}

// LoadDevicesFromCSV reads device IDs from a CSV file and initializes them in the store.
// The CSV is expected to have a header row with "device_id" as the first column.
func (s *Store) LoadDevicesFromCSV(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Skip header row (index 0), process data rows
	for i := 1; i < len(records); i++ {
		if len(records[i]) > 0 && records[i][0] != "" {
			deviceID := records[i][0]
			s.devices[deviceID] = &DeviceStats{ID: deviceID}
		}
	}

	return nil
}

// DeviceExists checks if a device ID is registered in the store.
func (s *Store) DeviceExists(deviceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.devices[deviceID]
	return exists
}

// RecordHeartbeat updates heartbeat statistics for a device.
// On first heartbeat: sets both FirstHeartbeat and LastHeartbeat.
// On subsequent heartbeats: only updates LastHeartbeat.
func (s *Store) RecordHeartbeat(deviceID string, sentAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return false
	}

	device.HeartbeatCount++
	if device.FirstHeartbeat.IsZero() {
		device.FirstHeartbeat = sentAt
	}
	device.LastHeartbeat = sentAt

	return true
}

// RecordUploadStat records an upload time measurement for a device.
func (s *Store) RecordUploadStat(deviceID string, uploadTime time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return false
	}

	device.UploadCount++
	device.UploadTimeSum += uploadTime

	return true
}

// StatsResult holds calculated statistics for a device.
type StatsResult struct {
	HasHeartbeats bool
	HasUploads    bool
	Uptime        float64
	AvgUploadTime time.Duration
}

// GetStats calculates statistics for a device.
// Returns uptime percentage and average upload time.
// Handles edge cases:
//   - Single heartbeat: returns 100% uptime (device was online at only observed moment)
//   - Zero uploads: HasUploads is false
func (s *Store) GetStats(deviceID string) (StatsResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, exists := s.devices[deviceID]
	if !exists {
		return StatsResult{}, false
	}

	result := StatsResult{}

	// Calculate uptime if we have heartbeats
	if device.HeartbeatCount > 0 {
		result.HasHeartbeats = true

		if device.HeartbeatCount == 1 {
			// Single heartbeat: device was online at that moment
			result.Uptime = 100.0
		} else {
			// Formula: (count / minutes_between_first_and_last) * 100
			// We add 1 to minutes to include the first minute (fence-post problem)
			minutesBetween := device.LastHeartbeat.Sub(device.FirstHeartbeat).Minutes() + 1
			result.Uptime = (float64(device.HeartbeatCount) / minutesBetween) * 100

			// Cap at 100% (could exceed if multiple heartbeats in same minute)
			if result.Uptime > 100.0 {
				result.Uptime = 100.0
			}
		}
	}

	// Calculate average upload time if we have uploads
	if device.UploadCount > 0 {
		result.HasUploads = true
		result.AvgUploadTime = device.UploadTimeSum / time.Duration(device.UploadCount)
	}

	return result, true
}

// DeviceCount returns the number of registered devices.
func (s *Store) DeviceCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.devices)
}
