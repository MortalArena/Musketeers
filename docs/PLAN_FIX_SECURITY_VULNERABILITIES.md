# Plan: Fix Security Vulnerabilities

## Overview
This plan details how to fix all identified security vulnerabilities in the Musketeers project.

## Vulnerabilities to Fix

### 1. SSRF in pkg/agent/tools/executor.go
**Severity**: 🔴 HIGH
**Issue**: http.Client does not use CheckRedirect function
**Impact**: DNS Rebinding / Redirect bypass possible

### 2. Agent Bridge without TLS/Auth
**Severity**: 🔴 HIGH
**Issue**: cmd/studio does not enable TLS (SetTLSConfig not called)
**Issue**: No authentication (generateSessionID is random)
**Impact**: Unencrypted communication, unauthorized access

### 3. ABAC non-functional
**Severity**: 🟡 MEDIUM
**Issue**: Only rule is "default-deny" (DENY everything)
**Issue**: No allow rules exist
**Impact**: System rejects everything

## Fix 1: SSRF Vulnerability

### File: pkg/agent/tools/executor.go
**Location**: httpRequest function (lines 359-404)

### Current Code
```go
client := &http.Client{
    Timeout: 30 * time.Second,
}
```

### Fixed Code
```go
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
```

### Additional Fix: Enhance isPrivateURL
**Location**: isPrivateURL function (lines 321-357)

### Current Code
```go
func isPrivateURL(rawURL string) bool {
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return true
    }

    if parsed.Scheme != "https" {
        return true
    }

    host := parsed.Hostname()

    blocked := []string{
        "localhost", "127.", "10.", "192.168.", "172.16.",
        "169.254.", "::1", "[::1]", "0.0.0.0",
    }
    for _, b := range blocked {
        if strings.HasPrefix(host, b) {
            return true
        }
    }

    ip := net.ParseIP(host)
    if ip != nil {
        if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
            return true
        }
    }

    return false
}
```

### Enhanced Code
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

## Fix 2: Agent Bridge TLS/Auth

### Step 1: Add TLS Flags to cmd/studio
**File**: cmd/studio/main.go
**Location**: After existing flags (around line 41)

```go
var (
    // ... existing flags ...
    bridgeTLSCert = flag.String("bridge-tls-cert", "", "TLS certificate file for Agent Bridge")
    bridgeTLSKey  = flag.String("bridge-tls-key", "", "TLS key file for Agent Bridge")
)
```

### Step 2: Enable TLS in cmd/studio
**File**: cmd/studio/main.go
**Location**: After bridge server creation (around line 250)

```go
// إنشاء خادم الجسر
bridgeServer := agent_bridge.NewServer(n, *agentAddr, sessionMgr, multiplexedBrg, log)

// Enable TLS if certificates are provided
if *bridgeTLSCert != "" && *bridgeTLSKey != "" {
    if err := bridgeServer.SetTLSConfig(*bridgeTLSCert, *bridgeTLSKey); err != nil {
        log.WithError(err).Fatal("Failed to set TLS config")
    }
    log.WithField("cert", *bridgeTLSCert).WithField("key", *bridgeTLSKey).Info("TLS enabled for Agent Bridge")
} else {
    log.Warn("TLS not enabled for Agent Bridge (not recommended for production)")
}
```

### Step 3: Add Authentication to Agent Bridge
**File**: pkg/agent_bridge/server.go
**Location**: Add after Server struct (around line 33)

```go
type Server struct {
    // ... existing fields ...
    authConfig AuthConfig
}

type AuthConfig struct {
    APIKeys    map[string]string // key -> agentID
    EnableAuth bool
}

func (s *Server) SetAuthConfig(authConfig AuthConfig) {
    s.authConfig = authConfig
}
```

### Step 4: Add Authentication Logic
**File**: pkg/agent_bridge/server.go
**Location**: Add authentication function (around line 150)

```go
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

### Step 5: Configure Authentication in cmd/studio
**File**: cmd/studio/main.go
**Location**: After TLS configuration (around line 260)

```go
// Configure authentication
authConfig := agent_bridge.AuthConfig{
    APIKeys: map[string]string{
        os.Getenv("AGENT_API_KEY"): "studio-agent",
    },
    EnableAuth: true,
}
bridgeServer.SetAuthConfig(authConfig)
log.Info("Authentication enabled for Agent Bridge")
```

## Fix 3: ABAC Allow Rules

### File: cmd/studio/main.go
**Location**: After policy engine creation (around line 290)

### Current Code
```go
// إضافة قاعدة افتراضية للسماح بالعمليات الأساسية
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

### Fixed Code
```go
// إنشاء Policy Engine
policyEngine := pkgPolicy.NewEngine()

// إضافة قاعدة default-deny
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

// إضافة قواعد allow للعمليات الأساسية
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
        log.WithError(err).Warnf("Failed to add allow rule: %s", rule.Name)
    }
}
log.WithField("rules", len(allowRules)).Info("Allow rules added to policy engine")
```

## Testing Plan

### Test 1: SSRF Fix
```bash
# Test redirect to private URL
curl -X POST http://127.0.0.1:5000/api/tools/http_request \
  -H "Content-Type: application/json" \
  -d '{"url": "https://evil.com/redirect-to-localhost"}'
```
**Expected**: Returns error "redirect to private URL not allowed"

### Test 2: Metadata Endpoint Blocking
```bash
# Test metadata endpoint
curl -X POST http://127.0.0.1:5000/api/tools/http_request \
  -H "Content-Type: application/json" \
  -d '{"url": "http://169.254.169.254/latest/meta-data/"}'
```
**Expected**: Returns error "private/internal URLs not allowed"

### Test 3: Agent Bridge TLS
```bash
# Test TLS connection
openssl s_client -connect 127.0.0.1:5001
```
**Expected**: TLS handshake succeeds

### Test 4: Agent Bridge Authentication
```bash
# Test without API key
nc 127.0.0.1 5001
```
**Expected**: Connection rejected

### Test 5: ABAC Allow Rules
```bash
# Test allowed operation
curl -X POST http://127.0.0.1:5000/api/data/read \
  -H "Authorization: Bearer <token>"
```
**Expected**: Operation allowed

## Verification Checklist

- [ ] SSRF fix implemented (CheckRedirect)
- [ ] Metadata endpoint blocking added
- [ ] Agent Bridge TLS enabled
- [ ] Agent Bridge authentication enabled
- [ ] ABAC allow rules added
- [ ] All tests pass
- [ ] No regressions

## Timeline

- **Fix 1 (SSRF)**: 30 minutes
- **Fix 2 (TLS/Auth)**: 45 minutes
- **Fix 3 (ABAC)**: 15 minutes
- **Testing**: 30 minutes
- **Total**: ~2 hours

## Success Criteria

1. ✅ SSRF vulnerability fixed
2. ✅ Metadata endpoints blocked
3. ✅ Agent Bridge TLS enabled
4. ✅ Agent Bridge authentication enabled
5. ✅ ABAC allow rules added
6. ✅ All tests pass
7. ✅ No regressions

## Notes

- **Critical fixes** - These are high-priority security fixes
- **Testing required** - Each fix must be tested thoroughly
- **No breaking changes** - Fixes are backward compatible
- **Gradual deployment** - Can deploy fixes incrementally
