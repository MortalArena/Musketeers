# Security Analysis

## Overview
This document analyzes security vulnerabilities in the Musketeers project, including SSRF, Agent Bridge TLS/Auth, and ABAC system.

## 1. SSRF Vulnerability in pkg/agent/tools/executor.go

### Location
**File**: pkg/agent/tools/executor.go
**Function**: `httpRequest`
**Lines**: 359-404

### Current Implementation
```go
func (te *ToolExecutor) httpRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    url, ok := params["url"].(string)
    if !ok {
        return nil, fmt.Errorf("المعامل url مطلوب")
    }

    // [SAFETY] فحص SSRF
    if isPrivateURL(url) {
        return nil, fmt.Errorf("SSRF: private/internal URLs not allowed: %s", url)
    }

    method, _ := params["method"].(string)
    if method == "" {
        method = "GET"
    }

    // [HOW] إنشاء طلب مع context للإلغاء
    req, err := http.NewRequestWithContext(ctx, method, url, nil)
    if err != nil {
        return nil, err
    }

    // [HOW] إرسال الطلب
    client := &http.Client{
        Timeout: 30 * time.Second, // [SAFETY] مهلة 30 ثانية
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // [HOW] قراءة الاستجابة
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return map[string]interface{}{
        "status_code": resp.StatusCode,
        "body":        string(body),
    }, nil
}
```

### Vulnerability Details
**Issue**: http.Client does not use CheckRedirect function
**Impact**: DNS Rebinding / Redirect bypass possible
**Severity**: 🔴 HIGH

### How the Vulnerability Works
1. Attacker provides a URL like `https://evil.com/redirect`
2. `isPrivateURL` checks the initial URL and allows it (not private)
3. HTTP client follows redirect to `http://localhost:8080/admin`
4. `isPrivateURL` is NOT called again on the redirect
5. Attacker can access internal services

### Exploit Example
```bash
# Attacker sends:
POST /api/tools/http_request
{
    "url": "https://evil.com/redirect-to-localhost",
    "method": "GET"
}

# evil.com/redirect-to-localhost returns:
HTTP/1.1 302 Found
Location: http://localhost:8080/admin

# HTTP client follows redirect to localhost
# isPrivateURL is NOT called again
# Attacker accesses internal admin panel
```

### Fix Required
```go
func (te *ToolExecutor) httpRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    url, ok := params["url"].(string)
    if !ok {
        return nil, fmt.Errorf("المعامل url مطلوب")
    }

    // [SAFETY] فحص SSRF
    if isPrivateURL(url) {
        return nil, fmt.Errorf("SSRF: private/internal URLs not allowed: %s", url)
    }

    method, _ := params["method"].(string)
    if method == "" {
        method = "GET"
    }

    // [HOW] إنشاء طلب مع context للإلغاء
    req, err := http.NewRequestWithContext(ctx, method, url, nil)
    if err != nil {
        return nil, err
    }

    // [FIX] Add CheckRedirect function
    client := &http.Client{
        Timeout: 30 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if len(via) >= 10 {
                return fmt.Errorf("too many redirects")
            }
            // Check redirect URL
            if isPrivateURL(req.URL.String()) {
                return fmt.Errorf("redirect to private URL not allowed: %s", req.URL.String())
            }
            return nil
        },
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // [HOW] قراءة الاستجابة
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return map[string]interface{}{
        "status_code": resp.StatusCode,
        "body":        string(body),
    }, nil
}
```

### Additional Improvements to isPrivateURL
```go
func isPrivateURL(rawURL string) bool {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return true
    }

    // [SAFETY] منع HTTP (فقط HTTPS)
    if parsed.Scheme != "https" {
        return true
    }

    host := parsed.Hostname()

    // [SAFETY] منع localhost والعناوين الداخلية
    blocked := []string{
        "localhost", "127.", "10.", "192.168.", "172.16.",
        "169.254.", "::1", "[::1]", "0.0.0.0",
    }
    for _, b := range blocked {
        if strings.HasPrefix(host, b) {
            return true
        }
    }

    // [SAFETY] فحص IP address
    ip := net.ParseIP(host)
    if ip != nil {
        if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
            return true
        }
    }

    // [FIX] Check for metadata endpoints (AWS, GCP, Azure)
    metadataEndpoints := []string{
        "metadata.google.internal",
        "169.254.169.254",
        "metadata.azure.net",
    }
    for _, endpoint := range metadataEndpoints {
        if host == endpoint {
            return true
        }
    }

    return false
}
```

## 2. Agent Bridge without TLS/Auth

### Location
**File**: pkg/agent_bridge/server.go
**Function**: `NewServer`, `Start`
**Lines**: 35-100

### Current Implementation
```go
// NewServer ينشئ خادم جسر جديد
func NewServer(n *node.Node, addr string, sessionMgr *SessionManager, multiplexedBrg *MultiplexedBridge, log *logrus.Logger) *Server {
    return &Server{
        node:           n,
        addr:           addr,
        sessionMgr:     sessionMgr,
        multiplexedBrg: multiplexedBrg,
        log:            log,
    }
}

// [SAFETY] SetTLSConfig sets TLS configuration for the server
func (s *Server) SetTLSConfig(certFile, keyFile string) error {
    // Check if certificate files exist
    if _, err := os.Stat(certFile); os.IsNotExist(err) {
        return fmt.Errorf("certificate file not found: %s", certFile)
    }
    if _, err := os.Stat(keyFile); os.IsNotExist(err) {
        return fmt.Errorf("key file not found: %s", keyFile)
    }

    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return fmt.Errorf("failed to load TLS certificate: %w", err)
    }

    s.tlsConfig = &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }
    s.certFile = certFile
    s.keyFile = keyFile
    s.useTLS = true

    s.log.WithField("cert_file", certFile).WithField("key_file", keyFile).Info("TLS configuration loaded")
    return nil
}

// Start يبدأ الخادم
func (s *Server) Start(ctx context.Context) error {
    s.mu.Lock()
    if s.running {
        s.mu.Unlock()
        return fmt.Errorf("server already running")
    }
    s.running = true
    s.shutdownCtx, s.shutdownCancel = context.WithCancel(ctx)
    s.mu.Unlock()

    var listener net.Listener
    var err error

    // [SAFETY] Use TLS if configured
    if s.useTLS && s.tlsConfig != nil {
        listener, err = tls.Listen("tcp", s.addr, s.tlsConfig)
        if err != nil {
            return fmt.Errorf("failed to listen on %s with TLS: %w", s.addr, err)
        }
        s.log.WithField("addr", s.addr).Info("Agent Bridge Server started with TLS")
    } else {
        listener, err = net.Listen("tcp", s.addr)
        if err != nil {
            return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
        }
        s.log.WithField("addr", s.addr).Warn("Agent Bridge Server started without TLS (not recommended for production)")
    }
```

### Vulnerability Details
**Issue 1**: cmd/studio does not enable TLS (SetTLSConfig not called)
**Issue 2**: No authentication (generateSessionID is random)
**Impact**: Unencrypted communication, unauthorized access
**Severity**: 🔴 HIGH

### How the Vulnerability Works
1. cmd/studio creates Agent Bridge server without calling SetTLSConfig
2. Server starts without TLS (unencrypted communication)
3. Anyone can connect to the server
4. No authentication required
5. Attacker can intercept or inject traffic

### Exploit Example
```bash
# Attacker connects to Agent Bridge
nc 127.0.0.1 5001

# Attacker can send arbitrary commands
# No authentication required
# All traffic is unencrypted
```

### Fix Required

#### Step 1: Add TLS Flags to cmd/studio
```go
var (
    addr       = flag.String("addr", "127.0.0.1:5000", "Studio server address")
    agentAddr  = flag.String("agent-addr", "127.0.0.1:5001", "Agent bridge address")
    dataDir    = flag.String("data-dir", "./studio-data", "Data directory")
    bootstrap  = flag.String("bootstrap", "", "Bootstrap peer multiaddr")
    founderPub = flag.String("founder-pub", "", "Founder public key hex")
    verbose    = flag.Bool("verbose", false, "Verbose logging")
    tlsCert    = flag.String("tls-cert", "", "TLS certificate file")
    tlsKey     = flag.String("tls-key", "", "TLS key file")
)
```

#### Step 2: Enable TLS in cmd/studio
```go
// إنشاء خادم الجسر
bridgeServer := agent_bridge.NewServer(n, *agentAddr, sessionMgr, multiplexedBrg, log)

// Enable TLS if certificates are provided
if *tlsCert != "" && *tlsKey != "" {
    if err := bridgeServer.SetTLSConfig(*tlsCert, *tlsKey); err != nil {
        log.WithError(err).Fatal("Failed to set TLS config")
    }
    log.WithField("cert", *tlsCert).WithField("key", *tlsKey).Info("TLS enabled for Agent Bridge")
} else {
    log.Warn("TLS not enabled for Agent Bridge (not recommended for production)")
}

if err := bridgeServer.Start(ctx); err != nil {
    log.WithError(err).Fatal("Failed to start bridge server")
}
defer bridgeServer.Stop()
```

#### Step 3: Add Authentication to Agent Bridge
```go
// In pkg/agent_bridge/server.go

type AuthConfig struct {
    APIKeys    map[string]string // key -> agentID
    EnableAuth bool
}

func (s *Server) SetAuthConfig(authConfig AuthConfig) {
    s.authConfig = authConfig
}

func (s *Server) authenticate(conn net.Conn) (string, error) {
    if !s.authConfig.EnableAuth {
        return generateSessionID(), nil
    }

    // Read API key from connection
    reader := bufio.NewReader(conn)
    apiKey, err := reader.ReadString('\n')
    if err != nil {
        return "", fmt.Errorf("failed to read API key: %w", err)
    }
    apiKey = strings.TrimSpace(apiKey)

    // Verify API key
    agentID, exists := s.authConfig.APIKeys[apiKey]
    if !exists {
        return "", fmt.Errorf("invalid API key")
    }

    return agentID, nil
}
```

#### Step 4: Configure Authentication in cmd/studio
```go
// Configure authentication
authConfig := agent_bridge.AuthConfig{
    APIKeys: map[string]string{
        os.Getenv("AGENT_API_KEY"): "studio-agent",
    },
    EnableAuth: true,
}
bridgeServer.SetAuthConfig(authConfig)
```

## 3. ABAC Non-functional

### Location
**File**: pkg/policy/engine.go
**Function**: `Evaluate`
**Lines**: 43-52

### Current Implementation in cmd/studio
```go
// إنشاء ExternalPlatformManager لإدارة المنصات الخارجية
// ملاحظة: ExternalPlatformManager يتطلب capability.Manager
// [SAFETY] إنشاء policy.Engine حقيقي بدلاً من nil
policyEngine := pkgPolicy.NewEngine()
// إضافة قاعدة افتراضية للسماح بالعمليات الأساسية
defaultRule := pkgPolicy.Rule{
    Name:     "default-deny",
    Priority: 0,
    Effect:   pkgPolicy.EffectDeny,
    Principals: []pkgPolicy.Principal{
        {DID: "*"},
    },
    Resources: []pkgPolicy.Resource{
        {Type: "*", Action: "*"},
    },
}
if err := policyEngine.AddRule(defaultRule); err != nil {
    log.WithError(err).Warn("Failed to add default policy rule")
}
```

### Vulnerability Details
**Issue**: Only rule is "default-deny" (DENY everything)
**Issue**: No allow rules exist
**Impact**: System rejects everything
**Severity**: 🟡 MEDIUM

### How the Vulnerability Works
1. Policy engine only has one rule: "default-deny"
2. Rule denies all principals, all resources, all actions
3. No allow rules exist
4. All requests are denied
5. System is non-functional

### Fix Required
```go
// Create policy engine
policyEngine := pkgPolicy.NewEngine()

// Add default-deny rule
defaultRule := pkgPolicy.Rule{
    Name:     "default-deny",
    Priority: 0,
    Effect:   pkgPolicy.EffectDeny,
    Principals: []pkgPolicy.Principal{
        {DID: "*"},
    },
    Resources: []pkgPolicy.Resource{
        {Type: "*", Action: "*"},
    },
}
if err := policyEngine.AddRule(defaultRule); err != nil {
    log.WithError(err).Warn("Failed to add default policy rule")
}

// Add allow rules for basic operations
allowRules := []pkgPolicy.Rule{
    {
        Name:     "allow-read-own-data",
        Priority: 10,
        Effect:   pkgPolicy.EffectAllow,
        Principals: []pkgPolicy.Principal{
            {DID: "*"},
        },
        Resources: []pkgPolicy.Resource{
            {Type: "data", Action: "read"},
        },
    },
    {
        Name:     "allow-write-own-data",
        Priority: 10,
        Effect:   pkgPolicy.EffectAllow,
        Principals: []pkgPolicy.Principal{
            {DID: "*"},
        },
        Resources: []pkgPolicy.Resource{
            {Type: "data", Action: "write"},
        },
    },
    {
        Name:     "allow-execute-tasks",
        Priority: 10,
        Effect:   pkgPolicy.EffectAllow,
        Principals: []pkgPolicy.Principal{
            {DID: "*"},
        },
        Resources: []pkgPolicy.Resource{
            {Type: "task", Action: "execute"},
        },
    },
    {
        Name:     "allow-join-channels",
        Priority: 10,
        Effect:   pkgPolicy.EffectAllow,
        Principals: []pkgPolicy.Principal{
            {DID: "*"},
        },
        Resources: []pkgPolicy.Resource{
            {Type: "channel", Action: "join"},
        },
    },
    {
        Name:     "allow-publish-channels",
        Priority: 10,
        Effect:   pkgPolicy.EffectAllow,
        Principals: []pkgPolicy.Principal{
            {DID: "*"},
        },
        Resources: []pkgPolicy.Resource{
            {Type: "channel", Action: "publish"},
        },
    },
}

for _, rule := range allowRules {
    if err := policyEngine.AddRule(rule); err != nil {
        log.WithError(err).Warn("Failed to add allow rule")
    }
}
```

## Summary

### Critical Security Issues
1. **SSRF vulnerability** in pkg/agent/tools/executor.go (no CheckRedirect)
2. **Agent Bridge without TLS/Auth** in cmd/studio
3. **ABAC non-functional** (only default-deny rule)

### Recommended Fixes
1. Add CheckRedirect function to http.Client in pkg/agent/tools/executor.go
2. Enable TLS in Agent Bridge in cmd/studio
3. Add authentication to Agent Bridge
4. Add allow rules to ABAC in cmd/studio
5. Add metadata endpoint blocking to isPrivateURL
6. Increase TLS version to TLS 1.3
7. Add certificate validation
8. Add rate limiting to Agent Bridge

### Priority
1. 🔴 HIGH: Fix SSRF vulnerability
2. 🔴 HIGH: Enable TLS in Agent Bridge
3. 🔴 HIGH: Add authentication to Agent Bridge
4. 🟡 MEDIUM: Fix ABAC system
