# Plan: Connect api/ to cmd/studio

## Overview
This plan details how to connect the REST API to cmd/studio, replacing the simple HTTP server.

## Current State
- cmd/studio uses simple HTTP server returning "Musketeers Studio is running"
- api/ exists with full REST API and Dashboard but is unused
- No REST API endpoints
- No web dashboard
- No authentication
- No rate limiting

## Target State
- cmd/studio uses full REST API
- Web dashboard available
- Authentication enabled
- Rate limiting enabled
- TLS support

## Required Modifications

### Step 1: Add Import
**File**: cmd/studio/main.go
**Location**: Top of file (after existing imports)

```go
import (
    // ... existing imports ...
    "github.com/MortalArena/Musketeers/api"
)
```

### Step 2: Add TLS Flags
**File**: cmd/studio/main.go
**Location**: After existing flags (around line 41)

```go
var (
    // ... existing flags ...
    tlsCert    = flag.String("tls-cert", "", "TLS certificate file for API server")
    tlsKey     = flag.String("tls-key", "", "TLS key file for API server")
    apiPort    = flag.Int("api-port", 8081, "REST API server port")
)
```

### Step 3: Create API Server
**File**: cmd/studio/main.go
**Location**: After bridge server creation (around line 250)

```go
// إنشاء REST API Server
apiServer := api.NewServerWithTLS(n, *apiPort, log, *tlsCert != "", *tlsCert, *tlsKey)
log.WithField("port", *apiPort).Info("API Server created")

// حفظ token للمصادقة
apiToken := apiServer.LocalToken()
log.WithField("token", apiToken[:10]+"...").Info("API authentication token generated")
```

### Step 4: Start API Server
**File**: cmd/studio/main.go
**Location**: After API server creation (around line 260)

```go
// بدء REST API Server في الخلفية
go func() {
    if err := apiServer.Start(); err != nil {
        log.WithError(err).Fatal("API server failed to start")
    }
}()
log.WithField("port", *apiPort).Info("API Server started")
```

### Step 5: Stop API Server on Shutdown
**File**: cmd/studio/main.go
**Location**: In shutdown handler (around line 320)

```go
// إيقاف REST API Server
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
defer shutdownCancel()

if err := apiServer.Stop(shutdownCtx); err != nil {
    log.WithError(err).Warn("Failed to stop API server gracefully")
}
```

### Step 6: Remove Simple HTTP Server
**File**: cmd/studio/main.go
**Location**: Remove simple HTTP server code (around lines 310-330)

**REMOVE**:
```go
// ❌ Remove this entire section:
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Musketeers Studio is running"))
})

server := &http.Server{
    Addr:    *addr,
    Handler: mux,
}

go func() {
    log.WithField("addr", *addr).Info("HTTP server starting...")
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.WithError(err).Fatal("HTTP server failed")
    }
}()

defer server.Shutdown(ctx)
```

## Dependencies Required
- ✅ `github.com/MortalArena/Musketeers/api` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/node` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/security` - exists
- ✅ `github.com/sirupsen/logrus` - exists

## Potential Risks

### Risk 1: Port Conflict
**Impact**: API server port conflicts with existing services
**Mitigation**: Use different port (8081) instead of 5000
**Fallback**: Change port if conflict occurs

### Risk 2: TLS Configuration Failure
**Impact**: API server fails to start if TLS certificates are invalid
**Mitigation**: Make TLS optional, allow HTTP if certificates not provided
**Fallback**: Run without TLS if certificates are invalid

### Risk 3: Authentication Token Loss
**Impact**: Users cannot authenticate if token is lost
**Mitigation**: Log token on startup, save to file
**Fallback**: Regenerate token if lost

## Rollback Plan

### If Integration Fails
1. Revert cmd/studio/main.go to previous version
2. Remove api/ import
3. Restore simple HTTP server code
4. Test cmd/studio works correctly

### Rollback Commands
```bash
git checkout cmd/studio/main.go
go build ./cmd/studio
./studio
```

## Testing Plan

### Test 1: Build
```bash
go build ./cmd/studio
```
**Expected**: Build succeeds without errors

### Test 2: Start cmd/studio
```bash
./studio --verbose
```
**Expected**: cmd/studio starts without errors, API server starts on port 8081

### Test 3: Access Health Endpoint
```bash
curl http://127.0.0.1:8081/api/health
```
**Expected**: Returns health status

### Test 4: Access Dashboard
```bash
curl http://127.0.0.1:8081/dashboard
```
**Expected**: Returns HTML dashboard

### Test 5: Test Authentication
```bash
curl -H "Authorization: Bearer <token>" http://127.0.0.1:8081/api/identity
```
**Expected**: Returns identity information

### Test 6: Test Rate Limiting
```bash
# Send many requests quickly
for i in {1..100}; do curl http://127.0.0.1:8081/api/health; done
```
**Expected**: Rate limiting kicks in after threshold

### Test 7: Test TLS (if enabled)
```bash
curl -k https://127.0.0.1:8081/api/health
```
**Expected**: Returns health status over HTTPS

## Verification Checklist

- [ ] cmd/studio builds successfully
- [ ] cmd/studio starts without errors
- [ ] API server starts on port 8081
- [ ] Health endpoint works
- [ ] Dashboard loads
- [ ] Authentication works
- [ ] Rate limiting works
- [ ] TLS works (if enabled)
- [ ] No port conflicts

## Timeline

- **Step 1**: Add import (5 minutes)
- **Step 2**: Add TLS flags (10 minutes)
- **Step 3**: Create API server (15 minutes)
- **Step 4**: Start API server (10 minutes)
- **Step 5**: Stop API server on shutdown (10 minutes)
- **Step 6**: Remove simple HTTP server (15 minutes)
- **Testing**: Build and test (30 minutes)
- **Total**: ~1.5 hours

## Success Criteria

1. ✅ cmd/studio builds successfully
2. ✅ cmd/studio starts without errors
3. ✅ API server starts on port 8081
4. ✅ Health endpoint works
5. ✅ Dashboard loads
6. ✅ Authentication works
7. ✅ Rate limiting works
8. ✅ TLS works (if enabled)
9. ✅ No port conflicts

## Notes

- This is a **gradual migration** - simple HTTP server can coexist with API server
- **No breaking changes** - can revert easily if needed
- **Incremental testing** - test each step before proceeding
- **Port separation** - API server uses different port to avoid conflicts
