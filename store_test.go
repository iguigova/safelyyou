package main

import (
	"os"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.devices == nil {
		t.Fatal("devices map not initialized")
	}
}

func TestLoadDevicesFromCSV(t *testing.T) {
	// Create a temporary CSV file
	content := "device_id\nabc-123\nxyz-456\n"
	tmpFile, err := os.CreateTemp("", "devices*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	s := NewStore()
	if err := s.LoadDevicesFromCSV(tmpFile.Name()); err != nil {
		t.Fatalf("LoadDevicesFromCSV failed: %v", err)
	}

	if s.DeviceCount() != 2 {
		t.Errorf("expected 2 devices, got %d", s.DeviceCount())
	}

	if !s.DeviceExists("abc-123") {
		t.Error("device abc-123 should exist")
	}
	if !s.DeviceExists("xyz-456") {
		t.Error("device xyz-456 should exist")
	}
	if s.DeviceExists("not-exists") {
		t.Error("device not-exists should not exist")
	}
}

func TestLoadDevicesFromCSV_FileNotFound(t *testing.T) {
	s := NewStore()
	err := s.LoadDevicesFromCSV("/nonexistent/path/devices.csv")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestRecordHeartbeat(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	t1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 15, 10, 5, 0, 0, time.UTC)

	// First heartbeat
	if !s.RecordHeartbeat("device-1", t1) {
		t.Error("RecordHeartbeat should return true for existing device")
	}

	device := s.devices["device-1"]
	if device.HeartbeatCount != 1 {
		t.Errorf("expected count 1, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(t1) {
		t.Errorf("expected FirstHeartbeat %v, got %v", t1, device.FirstHeartbeat)
	}
	if !device.LastHeartbeat.Equal(t1) {
		t.Errorf("expected LastHeartbeat %v, got %v", t1, device.LastHeartbeat)
	}

	// Second heartbeat
	s.RecordHeartbeat("device-1", t2)

	if device.HeartbeatCount != 2 {
		t.Errorf("expected count 2, got %d", device.HeartbeatCount)
	}
	if !device.FirstHeartbeat.Equal(t1) {
		t.Error("FirstHeartbeat should not change on subsequent heartbeats")
	}
	if !device.LastHeartbeat.Equal(t2) {
		t.Errorf("expected LastHeartbeat %v, got %v", t2, device.LastHeartbeat)
	}
}

func TestRecordHeartbeat_UnknownDevice(t *testing.T) {
	s := NewStore()
	t1 := time.Now()

	if s.RecordHeartbeat("unknown", t1) {
		t.Error("RecordHeartbeat should return false for unknown device")
	}
}

func TestRecordUploadStat(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	upload1 := 5 * time.Second
	upload2 := 10 * time.Second

	// First upload
	if !s.RecordUploadStat("device-1", upload1) {
		t.Error("RecordUploadStat should return true for existing device")
	}

	device := s.devices["device-1"]
	if device.UploadCount != 1 {
		t.Errorf("expected count 1, got %d", device.UploadCount)
	}
	if device.UploadTimeSum != upload1 {
		t.Errorf("expected sum %v, got %v", upload1, device.UploadTimeSum)
	}

	// Second upload
	s.RecordUploadStat("device-1", upload2)

	if device.UploadCount != 2 {
		t.Errorf("expected count 2, got %d", device.UploadCount)
	}
	if device.UploadTimeSum != upload1+upload2 {
		t.Errorf("expected sum %v, got %v", upload1+upload2, device.UploadTimeSum)
	}
}

func TestRecordUploadStat_UnknownDevice(t *testing.T) {
	s := NewStore()

	if s.RecordUploadStat("unknown", 5*time.Second) {
		t.Error("RecordUploadStat should return false for unknown device")
	}
}

func TestStore_GetStats_NoData(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	result, exists := s.GetStats("device-1")
	if !exists {
		t.Error("GetStats should return true for existing device")
	}
	if result.HasHeartbeats {
		t.Error("HasHeartbeats should be false with no heartbeats")
	}
	if result.HasUploads {
		t.Error("HasUploads should be false with no uploads")
	}
}

func TestGetStats_UnknownDevice(t *testing.T) {
	s := NewStore()

	_, exists := s.GetStats("unknown")
	if exists {
		t.Error("GetStats should return false for unknown device")
	}
}

func TestGetStats_SingleHeartbeat(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	t1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	s.RecordHeartbeat("device-1", t1)

	result, _ := s.GetStats("device-1")
	if !result.HasHeartbeats {
		t.Error("HasHeartbeats should be true")
	}
	if result.Uptime != 100.0 {
		t.Errorf("single heartbeat should have 100%% uptime, got %.2f%%", result.Uptime)
	}
}

func TestGetStats_UptimeCalculation(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	// Simulate 5 heartbeats over 10 minutes
	// Expected: 5 / (10 + 1) * 100 = 45.45%
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	s.RecordHeartbeat("device-1", baseTime)                    // minute 0
	s.RecordHeartbeat("device-1", baseTime.Add(2*time.Minute)) // minute 2
	s.RecordHeartbeat("device-1", baseTime.Add(5*time.Minute)) // minute 5
	s.RecordHeartbeat("device-1", baseTime.Add(8*time.Minute)) // minute 8
	s.RecordHeartbeat("device-1", baseTime.Add(10*time.Minute)) // minute 10

	result, _ := s.GetStats("device-1")
	// 5 heartbeats over 11 minutes (0-10 inclusive)
	expected := (5.0 / 11.0) * 100

	if result.Uptime < expected-0.1 || result.Uptime > expected+0.1 {
		t.Errorf("expected uptime ~%.2f%%, got %.2f%%", expected, result.Uptime)
	}
}

func TestGetStats_UptimeCap(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	// Multiple heartbeats in same minute should cap at 100%
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	s.RecordHeartbeat("device-1", baseTime)
	s.RecordHeartbeat("device-1", baseTime.Add(10*time.Second))
	s.RecordHeartbeat("device-1", baseTime.Add(20*time.Second))

	result, _ := s.GetStats("device-1")
	if result.Uptime > 100.0 {
		t.Errorf("uptime should be capped at 100%%, got %.2f%%", result.Uptime)
	}
}

func TestGetStats_AvgUploadTime(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	s.RecordUploadStat("device-1", 5*time.Second)
	s.RecordUploadStat("device-1", 10*time.Second)
	s.RecordUploadStat("device-1", 15*time.Second)

	result, _ := s.GetStats("device-1")
	if !result.HasUploads {
		t.Error("HasUploads should be true")
	}

	// (5 + 10 + 15) / 3 = 10 seconds
	expected := 10 * time.Second
	if result.AvgUploadTime != expected {
		t.Errorf("expected avg upload time %v, got %v", expected, result.AvgUploadTime)
	}
}

func TestGetStats_Combined(t *testing.T) {
	s := NewStore()
	s.devices["device-1"] = &DeviceStats{ID: "device-1"}

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Add heartbeats and uploads
	s.RecordHeartbeat("device-1", baseTime)
	s.RecordHeartbeat("device-1", baseTime.Add(1*time.Minute))
	s.RecordUploadStat("device-1", 5*time.Second)
	s.RecordUploadStat("device-1", 15*time.Second)

	result, exists := s.GetStats("device-1")
	if !exists {
		t.Error("device should exist")
	}
	if !result.HasHeartbeats {
		t.Error("should have heartbeats")
	}
	if !result.HasUploads {
		t.Error("should have uploads")
	}
	// 2 heartbeats over 2 minutes = 100%
	if result.Uptime != 100.0 {
		t.Errorf("expected 100%% uptime, got %.2f%%", result.Uptime)
	}
	// (5 + 15) / 2 = 10 seconds
	if result.AvgUploadTime != 10*time.Second {
		t.Errorf("expected avg 10s, got %v", result.AvgUploadTime)
	}
}
