# SafelyYou Device Monitoring - Implementation Log

## Project Overview

**Goal:** Implement API endpoints to receive telemetry data from devices and calculate:
- Device uptime (via heartbeat signals)
- Upload performance (average video upload time)

**Context:** SafelyYou runs AI detection at the edge (on-site servers at customer facilities). This system monitors the health and performance of that distributed hardware.

---

## Key Concepts

### Edge Computing
Processing data close to where it's generated rather than in a central cloud. Benefits:
- Faster response times
- Works during internet outages
- Less bandwidth usage
- Better privacy (raw video stays on-site)

### Telemetry
Remote measurement data that devices send to report their status. Examples: heartbeats, upload times, CPU usage.

---

## Progress Log

### Step 0: Documentation Setup

**Created project-local instructions file:**
```bash
# CLAUDE.md in project root - automatically read by Claude at session start
```

This file instructs Claude to update this implementation log after each step, ensuring all learning and progress is captured.

---

### Step 1: Environment Setup

**Checked system architecture:**
```bash
uname -m
# Output: x86_64 (means AMD64)
```

**Downloaded the device simulator:**
```bash
curl -L -o device-simulator https://sy-fleet-interview-assets.s3.us-east-2.amazonaws.com/device-simulator-linux-amd64
```
- `-L` = follow redirects
- `-o` = output to file

**Made it executable:**
```bash
chmod +x device-simulator
```

**Checked usage:**
```bash
./device-simulator --help
# Output:
#   -host string    host that stats server runs on (default "127.0.0.1")
#   -port int       port that stats server runs on (default 6733)
```

**What we learned:** The simulator expects an API server running on port 6733. It will send telemetry data and validate our responses.

### Step 2: Device Data

**Downloaded device list:**
```bash
curl -L -o devices.csv https://sy-fleet-interview-assets.s3.us-east-2.amazonaws.com/devices.csv
```

**Contents of devices.csv:**
```
device_id
60-6b-44-84-dc-64
b4-45-52-a2-f1-3c
26-9a-66-01-33-83
18-b8-87-e7-1f-06
38-4e-73-e0-33-59
```

**What we learned:**
- 5 devices to track
- Device IDs are MAC addresses
- Our API needs to accept telemetry for these specific devices

### Concept: MAC Address

A **MAC address** (Media Access Control address) is a unique hardware identifier assigned to every network interface (Wi-Fi card, Ethernet port, etc.).

**Format:** Six pairs of hexadecimal characters separated by dashes or colons
- Example: `60-6b-44-84-dc-64`

**Why use MAC addresses as device IDs?**
- Permanently assigned at manufacturing (doesn't change like an IP address)
- Unique to each physical device
- Reliable way to identify hardware across network changes

**Real-world analogy:** Think of it like a VIN (Vehicle Identification Number) for network devices - a permanent serial number baked into the hardware.

### Step 3: API Specification

**Downloaded OpenAPI spec:**
```bash
curl -L -o openapi.json https://sy-fleet-interview-assets.s3.us-east-2.amazonaws.com/openapi.json
```

### Concept: OpenAPI Specification

**OpenAPI** (formerly Swagger) is a standardized format to describe REST APIs. Think of it as a **blueprint or contract** that defines:
- What endpoints exist
- What data each endpoint accepts
- What responses each endpoint returns
- What errors can occur

**Why use it?**
- **Agreement** - Frontend, backend, QA all work from the same document
- **Auto-documentation** - Tools generate human-readable docs
- **Code generation** - Can auto-create server stubs or client libraries
- **Validation** - Test tools verify your API matches the spec

**Real-world analogy:** Like an architectural blueprint for a house - everyone (builders, electricians, plumbers) works from the same drawings to ensure everything fits together.

### The Three Required Endpoints

**Base URL:** `http://127.0.0.1:6733/api/v1`

#### 1. POST `/devices/{device_id}/heartbeat`
Records that a device is alive.

**Request body:**
```json
{
  "sent_at": "2024-01-15T10:30:00Z"   // ISO 8601 timestamp
}
```
**Response:** `204 No Content` (success, no body)

#### 2. POST `/devices/{device_id}/stats`
Records a video upload time measurement.

**Request body:**
```json
{
  "sent_at": "2024-01-15T10:30:00Z",
  "upload_time": 5000000000           // nanoseconds (5 seconds)
}
```
**Response:** `204 No Content`

#### 3. GET `/devices/{device_id}/stats`
Returns calculated statistics for a device.

**Response body:**
```json
{
  "avg_upload_time": "5m10s",    // Duration string format
  "uptime": 98.5                 // Percentage (0-100)
}
```

### Concept: HTTP Status Codes

| Code | Meaning | When to use |
|------|---------|-------------|
| 200 | OK | Returning data successfully |
| 204 | No Content | Success, but nothing to return |
| 404 | Not Found | Device ID doesn't exist |
| 500 | Server Error | Something broke internally |

### Concept: Nanoseconds

Time measurement in the spec uses **nanoseconds** (billionths of a second).

| Duration | Nanoseconds |
|----------|-------------|
| 1 millisecond | 1,000,000 |
| 1 second | 1,000,000,000 |
| 1 minute | 60,000,000,000 |

**Why nanoseconds?** High precision for performance measurements. Computers can measure time this accurately.

---

## Next Steps

- [x] Determine endpoint contracts (what requests the simulator sends)
- [ ] Choose technology stack for the API
- [ ] Implement POST `/devices/{device_id}/heartbeat`
- [ ] Implement POST `/devices/{device_id}/stats`
- [ ] Implement GET `/devices/{device_id}/stats` with calculations
- [ ] Load devices from CSV on startup
- [ ] Test with simulator

---

## Design Decisions

### Decision 1: Documentation Format

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| Changelog | Standard format, familiar | Designed for version releases, not learning |
| Markdown log | Flexible, supports narrative | Less structured |
| Wiki | Searchable, linkable | Overkill for single project |

**Chosen:** Markdown implementation log

**Reasoning:** Need to track learning journey and present later. A changelog is for version releases (v1.0, v1.1). An implementation log better captures the "why" behind decisions.

---

### Decision 2: Programming Language

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| Python (FastAPI) | Auto-generates docs, great for learning, fast development | Slower runtime, GIL limitations |
| Python (Flask) | Simple, minimal, easy to understand | Manual setup for validation |
| Node.js (Express) | Large ecosystem, async by default | Callback complexity |
| **Go** | Fast, compiled, excellent for APIs, likely matches simulator | Steeper learning curve, more verbose |

**Chosen:** Go

**Reasoning:**
- The simulator is written in Go (common for CLI tools)
- Go's `net/http` and `time` packages are excellent for this use case
- Native duration formatting (e.g., "5m10s") matches the expected output format
- Good learning opportunity for systems programming

---

### Decision 3: Data Storage Strategy

**The memory problem:** Storing every heartbeat (1/minute) and upload stat grows unbounded:
- 5 devices × 1440 min/day × 30 days = 216,000 records/month

**Key insight:** The formulas only need aggregates, not raw events:
- Uptime: `(count / minutes_between_first_and_last) × 100` → needs count, first, last
- Avg upload: `sum / count` → needs sum, count

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| Raw events in-memory | Simple writes | Unbounded memory growth |
| Raw events in SQLite | Persistent, queryable | Complexity, still grows |
| **Aggregates in-memory** | O(1) memory per device, fast | Data lost on restart |
| Aggregates + snapshots | Recoverable | Slightly more complex |

**Chosen:** Aggregates in-memory (with potential for periodic snapshots)

**Reasoning:** We only need counts/sums/timestamps to compute the required statistics. This uses ~100 bytes per device regardless of runtime duration. A periodic snapshot to disk could add crash recovery without full database complexity.

---

### Decision 4: Uptime Calculation - Reset Logic

**Question explored:** Should we reset `first_heartbeat` when a gap is detected?

**Scenario analyzed:**
```
10:00  Heartbeat (first) ✓
10:01  Heartbeat ✓
10:02-10:30  (offline, no heartbeats)
10:31  Heartbeat (back online) ✓
10:32  Heartbeat ✓
10:33  Heartbeat (last) ✓
```

**Option A - No reset (chosen):**
- Count: 5, Minutes: 33
- Uptime: 5/33 × 100 = 15.2%
- Reflects full historical reliability

**Option B - Reset on gap:**
- Would reset first to 10:31
- Count: 3, Minutes: 3
- Uptime: 100%
- Problem: Uptime would ALWAYS be ~100% since we'd only measure "good" periods

**Chosen:** No reset - follow the formula literally

**Reasoning:** The formula `(sumHeartbeats / numMinutesBetweenFirstAndLastHeartbeat) × 100` is designed to capture overall reliability. Resetting would defeat the purpose of measuring uptime. The spec does not require rolling windows or session-based calculations.

---

### Decision 5: Crash Recovery (Snapshots)

**Question:** Should we periodically save aggregates to disk to recover from server crashes?

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| No snapshots | Simpler code, fewer moving parts | All data lost on crash |
| **Periodic snapshots** | Recoverable, minimal code (~20 lines) | Could lose up to N minutes of data |
| Write-ahead log | No data loss | Complex, overkill for exercise |

**Chosen:** Skip for now

**Reasoning:**
- This is a coding exercise; simulator runs are short
- If server crashes mid-test, we'd restart and re-run the simulator anyway
- Can be added later if needed (~20 lines: JSON serialize aggregates every N minutes)

**Production consideration:** In a real system, you'd want either periodic snapshots or a proper database to avoid data loss during deployments or crashes.

---

### Concept: Thread Safety and Mutex

**The problem:** When multiple requests hit the server simultaneously, they may read/write the same data at the same time, causing a **race condition**.

**Example of race condition:**
```
Request 1: Read count (value: 5)
Request 2: Read count (value: 5)
Request 1: Write count = 6
Request 2: Write count = 6  ← Lost update! Should be 7
```

**Solution:** Use a **mutex** (mutual exclusion lock). Only one goroutine can hold the lock at a time.

**Go's sync.RWMutex:**
| Operation | Method | Behavior |
|-----------|--------|----------|
| Read | `RLock()` / `RUnlock()` | Multiple readers allowed simultaneously |
| Write | `Lock()` / `Unlock()` | Exclusive access, blocks everyone else |

**Why RWMutex over Mutex?**
- Pure Mutex: One operation at a time, always
- RWMutex: Multiple reads can happen in parallel (better performance)
- Use RWMutex when reads are more frequent than writes (common in APIs)

**Real-world analogy:** Like a bathroom lock. Multiple people can look in the mirror (read), but only one person can shower (write), and nobody else can enter during a shower.

---

### Concept: Duration Formatting in Go

**The requirement:** Return average upload time as a string like `"5m10s"`.

**Go's `time.Duration`:**
- Internally: an `int64` representing nanoseconds
- Has a `.String()` method that formats automatically

```go
d := time.Duration(310_000_000_000)  // 310 billion nanoseconds
fmt.Println(d.String())              // Output: "5m10s"
```

**Conversion table:**
| Nanoseconds | Meaning | String |
|-------------|---------|--------|
| 1,000,000,000 | 1 second | `"1s"` |
| 60,000,000,000 | 1 minute | `"1m0s"` |
| 310,000,000,000 | 5m 10s | `"5m10s"` |

**Why Go was a good choice:** The built-in duration formatting matches exactly what the spec requires - no custom code needed.

**Go time constants:**
```go
time.Nanosecond  // 1
time.Microsecond // 1,000
time.Millisecond // 1,000,000
time.Second      // 1,000,000,000
time.Minute      // 60,000,000,000
```

---

### Concept: CSV Loading in Go

**Go's `encoding/csv` package** provides built-in CSV parsing:

```go
file, _ := os.Open("devices.csv")
defer file.Close()

reader := csv.NewReader(file)
records, _ := reader.ReadAll()  // Returns [][]string
```

**Concept: defer**

`defer` schedules a function to run when the surrounding function returns:

```go
file, _ := os.Open("data.csv")
defer file.Close()  // Guaranteed to run, even on error

// ... do work with file
// file.Close() automatically called at end
```

**Why defer?**
- Guarantees cleanup even if errors occur
- Prevents resource leaks (forgotten file closes)
- Keeps cleanup code near the resource opening

---

### Decision 6: Device Initialization

**Question:** Should we pre-load devices from CSV, or create them on-demand when telemetry arrives?

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| **Pre-populate from CSV** | Can return 404 for unknown devices, matches spec | Requires CSV at startup |
| On-demand creation | Flexible, no CSV needed | Any device_id would work, no validation |

**Chosen:** Pre-populate from CSV

**Reasoning:** The spec provides a device list and expects 404 for unknown devices. Pre-populating allows proper validation.

---

### Concept: Error Handling

**Required error responses (from OpenAPI spec):**
| Status | When | Body |
|--------|------|------|
| 404 | Unknown device ID | `{"msg": "device not found"}` |
| 500 | Internal error | `{"msg": "description"}` |

**Edge cases to handle:**

1. **Unknown device ID:**
   - Check if device exists in store before processing
   - Return 404 with JSON body

2. **Invalid JSON in request:**
   - Return 400 Bad Request (good practice, though not in spec)

3. **GET stats with no data:**
   - Spec allows 204 No Content as valid response
   - Return 204 if no telemetry received yet

4. **Division by zero:**
   - Uptime: If only 1 heartbeat, `minutesBetween = 0`
     - Fix: Return 100% (device online at only observed moment)
   - Average: If `count = 0`, can't divide
     - Fix: Return 204 No Content

**Concept: HTTP Content-Type header**

When returning JSON, always set the header:
```go
w.Header().Set("Content-Type", "application/json")
```

This tells the client how to parse the response body.

---

### Clarification: OpenAPI vs OpenAI

- **OpenAPI** - A specification format for describing REST APIs (the JSON file we downloaded)
- **OpenAI** - An AI company (makers of ChatGPT)

Different things! We're using **OpenAPI** here.

---

### Decision 7: Code Generation vs Hand-Written

**Question:** Should we auto-generate Go code from the OpenAPI spec?

**Tools available:**
| Tool | What it generates |
|------|-------------------|
| oapi-codegen | Go types and server interfaces |
| go-swagger | Full server scaffolding |
| openapi-generator | Multi-language, verbose output |

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| **Hand-written** | Full control, better for learning, simpler | More code to write |
| oapi-codegen | Less boilerplate, type-safe from spec | Learning curve, "magic" generated code |

**Chosen:** Hand-written handlers

**Reasoning:** This is a learning exercise. Writing handlers by hand provides better understanding of Go's HTTP handling. The codebase is small (~150 lines) so generation overhead isn't worth it.

---

### Decision 8: Testing Strategy

**Question:** What level of testing to implement?

**Options considered:**
| Option | Coverage | Effort |
|--------|----------|--------|
| Unit only | Store methods, calculations | Low |
| **Unit + Integration** | Also full HTTP request/response | Medium |
| Unit + Integration + Table-driven | Multiple test cases per function | High |

**Chosen:** Unit + Integration tests

**Reasoning:** Unit tests verify logic correctness. Integration tests verify the HTTP layer works correctly with the simulator's expectations. Table-driven adds complexity not needed for this scope.

### Concept: Unit vs Integration Tests

**Unit tests:** Test individual functions in isolation
```go
func TestCalculateUptime(t *testing.T) {
    result := calculateUptime(5, time.Minute*10)
    assert(result == 50.0)
}
```

**Integration tests:** Test the full system working together
```go
func TestHeartbeatEndpoint(t *testing.T) {
    req := httptest.NewRequest("POST", "/api/v1/devices/abc/heartbeat", body)
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    assert(rr.Code == 204)
}
```

---

### Decision 9: CSV Loading Failure Handling

**Scenarios:**
1. File missing
2. File corrupt (malformed CSV)
3. File empty (header only)

**Options considered:**
| Option | Behavior | Pros | Cons |
|--------|----------|------|------|
| Fail fast (don't start) | Server refuses to start | Clear failure | Process dies, harder to debug |
| **Start + return 500** | Server starts, all requests return 500 | Process stays up, monitorable | Partially operational |
| Start + return 404 | Server starts, all requests return 404 | Simple | Misleading (implies bad device_id) |

**Chosen:** Start the server, return 500 for all requests if CSV failed

**Reasoning:**
- 500 signals "server misconfiguration" (correct semantics)
- 404 would misleadingly suggest device_id is wrong
- Server process stays up for health checks and debugging
- Monitoring systems can detect 500s and alert operators

**Implementation:**
```go
var csvLoadError error  // Set if CSV loading failed

func handler(w http.ResponseWriter, r *http.Request) {
    if csvLoadError != nil {
        w.WriteHeader(500)
        json.NewEncoder(w).Encode(map[string]string{
            "msg": "server configuration error: " + csvLoadError.Error(),
        })
        return
    }
    // ... normal handling
}
```

**Future consideration:** Graceful degradation and fallback strategies (to be discussed).

---

### Decision 10: Logging Strategy

**Question:** What and how to log?

**Options considered:**
| Option | Pros | Cons |
|--------|------|------|
| **log (stdlib)** | Built-in, simple, no deps | No log levels |
| log/slog | Structured, levels, built-in | More complex |
| zerolog/zap | Fast, flexible | External dependency |

**Chosen:** Simple stdlib log with clear event prefixes

**Events to log:**
| Event | Prefix | Example |
|-------|--------|---------|
| Server startup | `[STARTUP]` | `[STARTUP] Server listening on :6733` |
| CSV load | `[CONFIG]` | `[CONFIG] Loaded 5 devices` |
| Request received | `[REQUEST]` | `[REQUEST] POST /devices/abc/heartbeat` |
| Error | `[ERROR]` | `[ERROR] Invalid JSON in request body` |
| Not found | `[WARN]` | `[WARN] Device xyz not found` |

**Implementation:**
```go
import "log"

log.Println("[STARTUP] Server listening on :6733")
log.Println("[CONFIG] Loaded", len(devices), "devices")
log.Println("[REQUEST] POST", r.URL.Path)
log.Println("[ERROR] Invalid JSON:", err)
log.Println("[WARN] Device not found:", deviceID)
```

**Reasoning:** Prefixes make it easy to grep/filter logs by event type. Stdlib log is sufficient for this scope and adds no dependencies.

---

### Decision 11: Request Validation

**Question:** How strictly should we validate incoming requests?

**Options considered:**
| Option | Checks | Complexity |
|--------|--------|------------|
| Minimal | JSON valid, required fields exist | Low |
| Standard | + positive numbers, valid timestamps | Medium |
| **Strict** | + timestamp not future, reasonable ranges | High |

**Chosen:** Strict validation

**Validation rules:**

| Field | Validation | Error |
|-------|------------|-------|
| Request body | Valid JSON | 400 "invalid JSON" |
| `sent_at` | Required | 400 "sent_at is required" |
| `sent_at` | Valid ISO 8601 | 400 "invalid sent_at format" |
| `sent_at` | Not in future | 400 "sent_at cannot be in the future" |
| `upload_time` | Required | 400 "upload_time is required" |
| `upload_time` | > 0 (must be positive) | 400 "upload_time must be positive" |
| `upload_time` | Reasonable max (e.g., < 1 hour) | 400 "upload_time exceeds maximum" |
| `device_id` | Exists in store | 404 "device not found" |

**Implementation approach:**
```go
func validateHeartbeat(req *HeartbeatRequest) error {
    if req.SentAt.IsZero() {
        return errors.New("sent_at is required")
    }
    if req.SentAt.After(time.Now()) {
        return errors.New("sent_at cannot be in the future")
    }
    return nil
}

func validateUploadStat(req *UploadStatRequest) error {
    if req.SentAt.IsZero() {
        return errors.New("sent_at is required")
    }
    if req.UploadTime <= 0 {
        return errors.New("upload_time must be positive")
    }
    if req.UploadTime > int64(time.Hour) {
        return errors.New("upload_time exceeds maximum")
    }
    return nil
}
```

**Reasoning:** Strict validation catches bad data early, provides clear error messages, and demonstrates defensive programming. Even if the simulator sends valid data, production systems should validate everything.

### Note: Device ID Type

Device IDs (MAC addresses like `60-6b-44-84-dc-64`) are stored as **strings**, not parsed as MAC addresses.

**Reasoning:**
- OpenAPI spec defines device_id as `type: string`
- We treat it as an opaque identifier
- Validation = check if it exists in our store (from CSV)
- No need to validate MAC format - the CSV is our source of truth
- If IDs change format later (e.g., UUIDs), no code changes needed

---

## Questions to Resolve

- [x] What is the exact request/response format for each endpoint? → See Step 3
- [x] How should we calculate "uptime" from heartbeats? → See formula below
- [x] What time window for average upload time? → All time (no window)
- [x] What programming language/framework to use? → Go

---

## Implementation Complete

### Step 4: API Implementation

**Files created:**

| File | Purpose |
|------|---------|
| `go.mod` | Go module definition |
| `store.go` | DeviceStats struct, thread-safe Store with RWMutex |
| `store_test.go` | 14 unit tests for store methods |
| `handlers.go` | HTTP handlers for 3 endpoints |
| `handlers_test.go` | 13 integration tests for HTTP endpoints |
| `main.go` | Entry point, CSV loading, server startup |

**Test results:** 27 tests passing

### Simulator Discovery: `sent_at` Field Behavior

During testing, we discovered the simulator sends different data for each endpoint:

| Endpoint | `sent_at` value sent |
|----------|---------------------|
| POST /heartbeat | Real timestamp (`2024-04-02T09:00:00Z`) |
| POST /stats | Zero time (`0001-01-01T00:00:00Z`) |

**Why this works:**

- **Heartbeats need `sent_at`** - Used to calculate uptime via `lastHeartbeat.Sub(firstHeartbeat)`
- **Stats don't need `sent_at`** - Only `upload_time` is summed for average calculation

**Code adjustment:** Made `sent_at` optional for the stats endpoint validation.

### Final Validation Results

```
DeviceID: 60-6b-44-84-dc-64
    Uptime:        Expected: 99.79167  Actual: 99.58420
    AvgUploadTime: Expected: 3m7.893379134s  Actual: 3m7.893379134s

DeviceID: b4-45-52-a2-f1-3c
    Uptime:        Expected: 100.00000  Actual: 99.79210
    AvgUploadTime: Expected: 3m19.085533836s  Actual: 3m19.085533836s
```

- **Average upload times:** Match exactly
- **Uptimes:** Small variance due to `+1` minute in our formula for fence-post handling

### Uptime Formula Implementation

```go
if device.HeartbeatCount == 1 {
    result.Uptime = 100.0  // Single heartbeat = 100% at that moment
} else {
    minutesBetween := device.LastHeartbeat.Sub(device.FirstHeartbeat).Minutes() + 1
    result.Uptime = (float64(device.HeartbeatCount) / minutesBetween) * 100
    if result.Uptime > 100.0 {
        result.Uptime = 100.0  // Cap at 100%
    }
}
```

### Commands to Run

```bash
# Run tests
go test ./...

# Start server
go run .

# Run simulator validation
./device-simulator -host 127.0.0.1 -port 6733
```
