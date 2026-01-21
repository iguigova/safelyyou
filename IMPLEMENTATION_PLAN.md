# SafelyYou Device Monitoring API - Implementation Plan

## Overview

Build a Go API server that receives device telemetry (heartbeats, upload stats) and calculates per-device statistics (uptime percentage, average upload time).

## Data Structures

```go
type DeviceStats struct {
    ID              string

    // Heartbeat aggregates
    HeartbeatCount  int64
    FirstHeartbeat  time.Time
    LastHeartbeat   time.Time

    // Upload aggregates
    UploadCount     int64
    UploadTimeSum   time.Duration
}

// Thread-safe storage
type Store struct {
    mu      sync.RWMutex
    devices map[string]*DeviceStats
}
```

**Memory usage:** O(1) per device (~100 bytes), regardless of how long server runs.

## Calculations

**Uptime:**
```go
minutesBetween := lastHeartbeat.Sub(firstHeartbeat).Minutes() + 1
uptime := (float64(heartbeatCount) / minutesBetween) * 100
```

**Average Upload Time:**
```go
avgUploadTime := uploadTimeSum / time.Duration(uploadCount)
// Format as duration string: "5m10s"
```

## API Endpoints

Base URL: `http://127.0.0.1:6733/api/v1`

| Method | Path | Action |
|--------|------|--------|
| POST | `/devices/{device_id}/heartbeat` | Increment count, update first/last timestamps |
| POST | `/devices/{device_id}/stats` | Add to upload sum, increment count |
| GET | `/devices/{device_id}/stats` | Calculate and return uptime + avg_upload_time |

## Implementation Steps

1. [ ] Create Go module and project structure
2. [ ] Implement data store with thread-safe operations
3. [ ] Write unit tests for store (calculations, edge cases)
4. [ ] Load devices from CSV on startup
5. [ ] Implement POST heartbeat endpoint
6. [ ] Implement POST stats endpoint
7. [ ] Implement GET stats endpoint with calculations
8. [ ] Add 404 handling for unknown devices
9. [ ] Write integration tests for HTTP endpoints
10. [ ] Test with device-simulator

## Decisions Made

- **Language:** Go (matches simulator, native duration formatting)
- **Storage:** In-memory aggregates (O(1) per device)
- **Uptime reset:** No reset - use full history per formula
- **Snapshots:** Skip for exercise (document as future enhancement)
- **Thread safety:** Use sync.RWMutex for concurrent request handling
- **Device init:** Pre-populate from CSV (enables 404 for unknown devices)
- **Code generation:** Hand-written (better for learning, full control)
- **Testing:** Unit + Integration tests
- **Logging:** stdlib log with event prefixes ([STARTUP], [CONFIG], [REQUEST], [ERROR], [WARN])
- **Validation:** Strict (required fields, valid timestamps, not future, reasonable ranges)

## Error Handling

| Scenario | Response |
|----------|----------|
| CSV load failed | 500 `{"msg": "server configuration error: ..."}` |
| Unknown device ID | 404 `{"msg": "device not found"}` |
| Invalid JSON | 400 `{"msg": "invalid JSON"}` |
| Missing required field | 400 `{"msg": "field_name is required"}` |
| Timestamp in future | 400 `{"msg": "sent_at cannot be in the future"}` |
| upload_time not positive | 400 `{"msg": "upload_time must be positive"}` |
| upload_time too large | 400 `{"msg": "upload_time exceeds maximum"}` |
| GET stats, no data | 204 No Content |
| Single heartbeat (divide by zero) | Return 100% uptime |
| Zero uploads (divide by zero) | Return 204 No Content |

**CSV failure strategy:** Server starts even if CSV fails, but returns 500 for all requests. This keeps the process up for monitoring/debugging while clearly signaling misconfiguration.

## Files to Create

```
safelyyou/
├── main.go           # Entry point, HTTP server setup
├── store.go          # DeviceStats struct, Store with thread-safe methods
├── store_test.go     # Unit tests for store methods
├── handlers.go       # HTTP handlers for the 3 endpoints
├── handlers_test.go  # Integration tests for HTTP endpoints
├── devices.csv       # (already downloaded)
└── go.mod            # Go module definition
```

## Testing Strategy

**Unit tests (store_test.go):**
- `TestRecordHeartbeat` - first heartbeat sets first/last, subsequent updates last
- `TestRecordUploadStat` - sum and count increment correctly
- `TestCalculateUptime` - formula correctness, single heartbeat edge case
- `TestCalculateAvgUpload` - average calculation, zero count edge case

**Integration tests (handlers_test.go):**
- `TestPostHeartbeat_Success` - valid device returns 204
- `TestPostHeartbeat_NotFound` - unknown device returns 404
- `TestPostStats_Success` - valid upload stat returns 204
- `TestGetStats_Success` - returns correct JSON format
- `TestGetStats_NoData` - returns 204 when no telemetry

Run tests:
```bash
go test ./...
```

## Verification

1. Start the server:
   ```bash
   go run .
   ```

2. Manual test (optional):
   ```bash
   # Test heartbeat
   curl -X POST http://127.0.0.1:6733/api/v1/devices/60-6b-44-84-dc-64/heartbeat \
     -H "Content-Type: application/json" \
     -d '{"sent_at": "2024-01-15T10:00:00Z"}'

   # Test unknown device (should 404)
   curl -X POST http://127.0.0.1:6733/api/v1/devices/unknown/heartbeat \
     -H "Content-Type: application/json" \
     -d '{"sent_at": "2024-01-15T10:00:00Z"}'
   ```

3. Run simulator for full validation:
   ```bash
   ./device-simulator -host 127.0.0.1 -port 6733
   ```

## Future Considerations (Out of Scope)

These topics are worth discussing in a presentation but not implementing for this exercise:

| Topic | Question | Default for Now |
|-------|----------|-----------------|
| Configuration | Should port/CSV path be configurable via flags or env vars? | Hardcoded |
| Graceful shutdown | Handle SIGTERM/SIGINT, drain in-flight requests? | Immediate exit |
| Health check | Add `/health` endpoint for monitoring? | Not included |
| Metrics | Expose Prometheus metrics? | Not included |
| Rate limiting | Protect against request floods? | Not included |
| Periodic snapshots | Save state to disk for crash recovery? | Decided: skip |

These demonstrate awareness of production concerns during presentation.
