# API Analysis - api/

## Overview
api/ contains a comprehensive REST API with a web dashboard. This analysis examines how it works and how it can be connected to cmd/studio.

## Components

### 1. Server (REST API Server)
**File**: rest.go (614 lines)

**Purpose**: REST API server for Musketeers

**Configuration**:
```go
type Server struct {
    node        *node.Node
    log         *logrus.Logger
    token       string // local token for authentication
    server      *http.Server
    channels    map[string]*pubsub.Subscription
    messages    map[string][]protocol.ChannelMessage
    channelsMu  sync.RWMutex
    tlsEnabled  bool
    tlsCert     string
    tlsKey      string
    rateLimiter *security.RateLimiter
}
```

**Key Features**:
- REST API endpoints
- TLS support
- Rate limiting
- Authentication
- CORS support
- Channel management
- Message management

**Endpoints**:
- `/api/identity` - Get identity information
- `/api/search` - Search for peers
- `/api/resolve` - Resolve peer addresses
- `/api/content` - Content management
- `/api/acp/task` - Execute ACP task
- `/api/acp/tasks` - List ACP tasks
- `/api/domain/commit` - Domain commit
- `/api/channels/join` - Join channel
- `/api/channels/publish` - Publish to channel
- `/api/channels/list` - List channels
- `/api/channels/messages` - Get channel messages
- `/api/health` - Health check
- `/dashboard` - Web dashboard
- `/` - Root endpoint

**Middleware**:
- CORS middleware
- Authentication middleware
- Rate limiting middleware

**Key Methods**:
- `NewServer(n *node.Node, port int, log *logrus.Logger)` - Creates server
- `NewServerWithTLS(n *node.Node, port int, log *logrus.Logger, tlsEnabled bool, tlsCert, tlsKey string)` - Creates server with TLS
- `SetTLSConfig(certFile, keyFile string)` - Sets TLS configuration
- `Start()` - Starts server
- `Stop()` - Stops server

**How It Works**:
1. Creates HTTP server with mux
2. Registers all endpoints
3. Applies middleware (CORS, auth, rate limiting)
4. Configures TLS if enabled
5. Starts HTTP server
6. Handles requests

### 2. Dashboard (Web Dashboard)
**File**: dashboard.go (162657 bytes)

**Purpose**: Web dashboard for Musketeers

**Key Features**:
- Real-time monitoring
- Agent management
- Channel management
- Task management
- Performance metrics
- System status
- Configuration UI

**Components**:
- HTML templates
- JavaScript client
- CSS styling
- WebSocket support
- Real-time updates

### 3. LocalWSBridge (WebSocket Bridge)
**File**: local_ws_bridge.go (12373 bytes)

**Purpose**: WebSocket bridge for local communication

**Key Features**:
- WebSocket server
- Message routing
- Event broadcasting
- Connection management

## Dependencies

### Internal Dependencies
- pkg/naming
- pkg/node
- pkg/protocol
- pkg/security
- libp2p-pubsub

### External Dependencies
- net/http
- crypto/rand
- crypto/subtle
- encoding/json
- io
- strings
- sync
- time
- logrus

## Current Status

### Importers
❌ **NONE** - The API system is completely unused!

### Why It's Not Used
1. **cmd/studio uses simple HTTP server instead**
2. **No entry point imports api/**
3. **No documentation on how to use it**
4. **No examples of how to integrate it**

## How to Connect to cmd/studio

### Step 1: Import API
```go
import (
    "github.com/MortalArena/Musketeers/api"
)
```

### Step 2: Create API Server
```go
// Create API server
apiPort := 8081
apiServer := api.NewServer(n, apiPort, log)
```

### Step 3: Configure TLS (optional)
```go
// Configure TLS
tlsCert := flag.String("tls-cert", "", "TLS certificate file")
tlsKey := flag.String("tls-key", "", "TLS key file")

if *tlsCert != "" && *tlsKey != "" {
    if err := apiServer.SetTLSConfig(*tlsCert, *tlsKey); err != nil {
        log.WithError(err).Fatal("Failed to set TLS config")
    }
}
```

### Step 4: Start API Server
```go
// Start API server in background
go func() {
    if err := apiServer.Start(); err != nil {
        log.WithError(err).Fatal("API server failed")
    }
}()
log.WithField("port", apiPort).Info("API server started")
```

### Step 5: Stop API Server on Shutdown
```go
// Stop API server on shutdown
defer apiServer.Stop()
```

### Step 6: Remove Simple HTTP Server
```go
// ❌ Remove:
// mux := http.NewServeMux()
// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//     w.WriteHeader(http.StatusOK)
//     w.Write([]byte("Musketeers Studio is running"))
// })
// server := &http.Server{
//     Addr:    *addr,
//     Handler: mux,
// }
// go func() {
//     log.WithField("addr", *addr).Info("HTTP server starting...")
//     if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
//         log.WithError(err).Fatal("HTTP server failed")
//     }
// }()

// ✅ API server handles everything
```

### Step 7: Connect to UnifiedAgent (if using UnifiedAgent)
```go
// Pass unified agent to API server
apiServer.SetUnifiedAgent(unifiedAgent)
```

## Benefits of Using api/

### 1. Full REST API
- Comprehensive API endpoints
- Standard HTTP methods
- JSON responses
- Error handling

### 2. Web Dashboard
- Real-time monitoring
- Agent management UI
- Channel management UI
- Task management UI
- Performance metrics
- System status

### 3. Security Features
- TLS support
- Authentication
- Rate limiting
- CORS support

### 4. Channel Management
- Join channels
- Publish to channels
- List channels
- Get channel messages

### 5. Content Management
- Content upload
- Content download
- Content search
- Content verification

### 6. ACP Integration
- Execute ACP tasks
- List ACP tasks
- Task management

### 7. Health Monitoring
- Health check endpoint
- System status
- Performance metrics

## Comparison with Current Implementation

### Current Implementation (cmd/studio)
```go
// ❌ Simple HTTP server
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
```

**Issues**:
- Only returns "Musketeers Studio is running"
- No REST API
- No dashboard
- No authentication
- No rate limiting
- No TLS
- No channel management
- No content management
- No ACP integration
- No health monitoring

### Should Be (using api/)
```go
// ✅ Full REST API with dashboard
apiPort := 8081
apiServer := api.NewServer(n, apiPort, log)

// Configure TLS
if *tlsCert != "" && *tlsKey != "" {
    if err := apiServer.SetTLSConfig(*tlsCert, *tlsKey); err != nil {
        log.WithError(err).Fatal("Failed to set TLS config")
    }
}

// Start API server
go func() {
    if err := apiServer.Start(); err != nil {
        log.WithError(err).Fatal("API server failed")
    }
}()
log.WithField("port", apiPort).Info("API server started")

// Stop on shutdown
defer apiServer.Stop()
```

**Benefits**:
- Full REST API
- Web dashboard
- Authentication
- Rate limiting
- TLS support
- Channel management
- Content management
- ACP integration
- Health monitoring

## Summary

### Current State
- ✅ API system exists and is well-designed
- ✅ Full REST API implemented
- ✅ Web dashboard implemented
- ✅ Comprehensive feature set
- ❌ Completely unused by any entry point
- ❌ No documentation on how to use it
- ❌ No examples of integration

### Recommendations
1. Connect cmd/studio to api/
2. Replace simple HTTP server with REST API
3. Add documentation on how to use api/
4. Add examples of integration
5. Test api/ with real workloads
6. Add API authentication UI
