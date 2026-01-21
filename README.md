# SafelyYou Device Monitoring API

A Go API server that receives device telemetry (heartbeats, upload stats) and calculates per-device statistics (uptime percentage, average upload time).

## Quick Start

### Prerequisites

- Go 1.21+ installed
- `device-simulator` binary in project root

### Run the Server

```bash
# From the project directory
go run .
```

The server starts on port **6733** and loads devices from `devices.csv`.

### Run the Simulator

In a separate terminal:

```bash
./device-simulator -host 127.0.0.1 -port 6733
```

Results are output to `results.txt` and the console.

### Run Tests

```bash
go test ./...
```

Expected output: 27 tests passing.

## Project Structure

```
safelyyou/
├── main.go           # Entry point, HTTP server setup
├── store.go          # DeviceStats struct, thread-safe Store
├── handlers.go       # HTTP handlers for 3 endpoints
├── store_test.go     # Unit tests (14 tests)
├── handlers_test.go  # Integration tests (13 tests)
├── devices.csv       # Device list (loaded at startup)
├── results.txt       # Simulator output
└── go.mod            # Go module definition
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/devices/{device_id}/heartbeat` | Record device is alive |
| POST | `/api/v1/devices/{device_id}/stats` | Record upload time measurement |
| GET | `/api/v1/devices/{device_id}/stats` | Get uptime % and avg upload time |

---

# Solution Write-Up

## Time Spent & Challenges

**Time spent:** Approximately 3-4 hours total, including:
- Understanding requirements and API spec (~30 min)
- Designing data model and formulas (~30 min)
- Implementation (~1.5 hours)
- Testing and debugging (~1 hour)
- Documentation (~30 min)

**Most difficult part:** Getting the uptime calculation correct. The formula `(heartbeat_count / minutes_between_first_and_last) * 100` has edge cases:

1. **Single heartbeat:** Division by zero if `first == last`. Solution: Return 100% (device was online at that moment).

2. **Fence-post problem:** Should 5 heartbeats over 5 minutes be 100% or 125%? I added `+1` to the denominator to account for the first minute being inclusive. This causes a ~0.2% variance from the simulator's expected values, which the spec notes is acceptable.

3. **Thread safety:** Multiple goroutines handling concurrent requests could corrupt shared state. Used `sync.RWMutex` to allow concurrent reads while ensuring exclusive writes.

## Extending for More Metrics

The current data model uses **aggregates** (counts, sums, timestamps) rather than storing raw events:

```go
type DeviceStats struct {
    ID              string
    HeartbeatCount  int64
    FirstHeartbeat  time.Time
    LastHeartbeat   time.Time
    UploadCount     int64
    UploadTimeSum   time.Duration
}
```

**To add new metrics, I would:**

1. **Add aggregate fields** to `DeviceStats` for the new metric type:
   ```go
   // Example: CPU temperature monitoring
   TempReadingCount int64
   TempSum          float64
   TempMax          float64
   TempMin          float64
   ```

2. **Add a new POST endpoint** to receive the metric data:
   ```go
   POST /api/v1/devices/{device_id}/temperature
   ```

3. **Extend GET /stats response** with calculated values:
   ```json
   {
     "uptime": 99.5,
     "avg_upload_time": "3m7s",
     "avg_temperature": 45.2,
     "max_temperature": 78.1
   }
   ```

**For many metric types**, I would consider:

- **Generic metric storage:** A map of metric name to aggregates, avoiding struct proliferation
- **Time-windowed aggregates:** Keep hourly/daily buckets for trend analysis
- **Separate storage backends:** Move from in-memory to Redis or TimescaleDB for durability and querying

## Runtime Complexity

### Space Complexity: O(D)

- **D** = number of devices
- Each device uses ~100 bytes of fixed storage regardless of how long the server runs
- No raw event storage means memory is bounded

### Time Complexity per Operation:

| Operation | Complexity | Notes |
|-----------|------------|-------|
| POST heartbeat | O(1) | Map lookup + field updates |
| POST stats | O(1) | Map lookup + field updates |
| GET stats | O(1) | Map lookup + arithmetic |
| CSV load | O(N) | N = number of lines in CSV |

### Concurrency

- **Read operations** (`DeviceExists`, `GetStats`): Use `RLock()`, allowing unlimited concurrent readers
- **Write operations** (`RecordHeartbeat`, `RecordUploadStat`): Use `Lock()`, serializing writes per device

The mutex is on the entire store, not per-device. For higher throughput with many devices, I could use:
- Sharded maps (partition by device ID hash)
- Per-device locks (finer granularity)
- Lock-free atomic operations for counters

### Production Considerations

The current implementation is **safe for production** with these caveats documented:

| Concern | Current State | Production Enhancement |
|---------|--------------|----------------------|
| Data persistence | In-memory only | Add periodic snapshots or database |
| Graceful shutdown | Immediate exit | Handle SIGTERM, drain requests |
| Health checks | None | Add `/health` endpoint |
| Metrics | Logging only | Add Prometheus metrics |
| Rate limiting | None | Add per-device rate limits |

These are intentionally omitted to keep the solution focused, but would be straightforward to add.
