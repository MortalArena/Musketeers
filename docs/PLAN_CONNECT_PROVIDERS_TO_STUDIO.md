# Plan: Connect pkg/providers to cmd/studio

## Overview
This plan details how to connect the Smart Router and 34+ AI providers to cmd/studio, replacing hardcoded API adapters.

## Current State
- cmd/studio uses hardcoded API adapters (API, CLI, IDE, Local, Browser, Custom)
- pkg/providers exists with 34+ providers and Smart Router but is unused
- No intelligent model selection
- No cost optimization
- No fallback logic
- Hardcoded API keys

## Target State
- cmd/studio uses Smart Router
- 34+ AI providers available
- Intelligent model selection
- Cost optimization
- Fallback logic
- API key management

## Required Modifications

### Step 1: Add Import
**File**: cmd/studio/main.go
**Location**: Top of file (after existing imports)

```go
import (
    // ... existing imports ...
    "github.com/MortalArena/Musketeers/pkg/providers"
    "github.com/MortalArena/Musketeers/pkg/providers/builtin/openai"
    "github.com/MortalArena/Musketeers/pkg/providers/builtin/anthropic"
    "github.com/MortalArena/Musketeers/pkg/providers/builtin/google"
    "github.com/MortalArena/Musketeers/pkg/providers/builtin/ollama"
)
```

### Step 2: Create Provider Registry
**File**: cmd/studio/main.go
**Location**: After BadgerDB creation (around line 105)

```go
// إنشاء Provider Registry
providerRegistry := providers.NewRegistry()
log.Info("Provider Registry created")
```

### Step 3: Register Providers
**File**: cmd/studio/main.go
**Location**: After provider registry creation (around line 110)

```go
// تسجيل المزودين
// OpenAI
if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
    providerRegistry.Register(providers.NewOpenAIProvider(apiKey))
    log.Info("OpenAI provider registered")
} else {
    log.Warn("OPENAI_API_KEY not set, OpenAI provider not registered")
}

// Anthropic
if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
    providerRegistry.Register(providers.NewAnthropicProvider(apiKey))
    log.Info("Anthropic provider registered")
} else {
    log.Warn("ANTHROPIC_API_KEY not set, Anthropic provider not registered")
}

// Google
if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
    providerRegistry.Register(providers.NewGoogleProvider(apiKey))
    log.Info("Google provider registered")
} else {
    log.Warn("GOOGLE_API_KEY not set, Google provider not registered")
}

// Ollama (local)
providerRegistry.Register(providers.NewOllamaProvider("http://localhost:11434"))
log.Info("Ollama provider registered")

log.WithField("provider_count", len(providerRegistry.List())).Info("Providers registered")
```

### Step 4: Create Smart Router
**File**: cmd/studio/main.go
**Location**: After provider registration (around line 140)

```go
// إنشاء Smart Router
routerConfig := providers.RouterConfig{
    PreferFreeModels:    true,
    PreferLocalModels:   true,
    MaxRetries:          3,
    Timeout:             30 * time.Second,
    FallbackEnabled:     true,
    CostOptimization:    true,
    LatencyOptimization: false,
}

router := providers.NewRouter(providerRegistry, routerConfig)
log.Info("Smart Router created")
```

### Step 5: Replace Hardcoded API Adapters
**File**: cmd/studio/main.go
**Location**: Replace hardcoded adapter registration (around lines 114-165)

**REMOVE**:
```go
// ❌ Remove this entire section:
// تسجيل الوكلاء الافتراضيين
// API Adapter
apiConfig := &pkgAdapters.APIConfig{
    APIKey:    "sk-test",
    BaseURL:   "https://api.anthropic.com",
    Model:     "claude-3-opus",
    MaxTokens: 4096,
    Timeout:   30 * time.Second,
}
apiAdapter := pkgAdapters.NewAPIAdapter(apiConfig)
agentRegistry.Register(apiAdapter, nil)

// CLI Adapter
cliConfig := &pkgAdapters.CLIConfig{
    Name:    "claude-code",
    Command: "claude",
    Args:    []string{},
}
cliAdapter := pkgAdapters.NewCLIAdapter(cliConfig)
agentRegistry.Register(cliAdapter, nil)

// IDE Adapter
ideConfig := &pkgAdapters.IDEConfig{
    Name:    "cursor",
    IDEType: "cursor",
}
ideAdapter := pkgAdapters.NewIDEAdapter(ideConfig)
agentRegistry.Register(ideAdapter, nil)

// Local Adapter
localConfig := &pkgAdapters.LocalConfig{
    Name:    "ollama",
    Model:   "llama2",
    BaseURL: "http://localhost:11434",
}
localAdapter := pkgAdapters.NewLocalAdapter(localConfig)
agentRegistry.Register(localAdapter, nil)

// Browser Adapter
browserAdapter := pkgAdapters.NewComputerUseAdapter("sk-test")
agentRegistry.Register(browserAdapter, nil)

// Custom Adapter
customAdapter := pkgAdapters.NewCustomAgent("custom", "custom", "custom-model", func(ctx context.Context, task *pkgAgent.AgentTask) (*pkgAgent.TaskExecutionResult, error) {
    return &pkgAgent.TaskExecutionResult{
        Success: true,
        Output:  "Custom agent executed task",
    }, nil
})
customAdapter.Initialize(map[string]interface{}{})
agentRegistry.Register(customAdapter, nil)
```

**ADD**:
```go
// ✅ Use Smart Router instead:
// Smart Router handles intelligent model selection
log.Info("Smart Router handles model selection")
```

### Step 6: Test Smart Router
**File**: cmd/studio/main.go
**Location**: After router creation (around line 150)

```go
// اختبار Smart Router
testReq := &providers.CompletionRequest{
    Messages: []providers.Message{
        {Role: "user", Content: "Hello"},
    },
    MaxTokens: 100,
}

resp, err := router.Route(ctx, testReq)
if err != nil {
    log.WithError(err).Warn("Failed to route test request")
} else {
    log.WithField("provider", resp.Provider).WithField("model", resp.Model).Info("Test request routed successfully")
}
```

## Dependencies Required
- ✅ `github.com/MortalArena/Musketeers/pkg/providers` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/providers/builtin/openai` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/providers/builtin/anthropic` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/providers/builtin/google` - exists
- ✅ `github.com/MortalArena/Musketeers/pkg/providers/builtin/ollama` - exists

## Potential Risks

### Risk 1: API Keys Not Set
**Impact**: Providers fail to register, no models available
**Mitigation**: Log warnings, allow system to start without providers
**Fallback**: Use hardcoded adapters if providers fail

### Risk 2: Ollama Not Running
**Impact**: Local provider fails, no local models available
**Mitigation**: Handle Ollama connection errors gracefully
**Fallback**: Use cloud providers if Ollama not available

### Risk 3: Smart Router Fails
**Impact**: No model selection, system non-functional
**Mitigation**: Add error handling, fall back to hardcoded adapters
**Fallback**: Restore hardcoded adapters if router fails

## Rollback Plan

### If Integration Fails
1. Revert cmd/studio/main.go to previous version
2. Remove pkg/providers imports
3. Restore hardcoded adapter registration
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
OPENAI_API_KEY=sk-xxx ANTHROPIC_API_KEY=sk-xxx GOOGLE_API_KEY=xxx ./studio --verbose
```
**Expected**: cmd/studio starts without errors, providers registered

### Test 3: Check Provider Registration
```bash
# Check logs for provider registration
```
**Expected**: All providers registered successfully

### Test 4: Test Smart Router
```bash
# Send completion request via API
```
**Expected**: Smart Router routes request successfully

### Test 5: Test Fallback Logic
```bash
# Disable one provider, test fallback
```
**Expected**: Router falls back to next provider

### Test 6: Test Cost Optimization
```bash
# Send multiple requests, check cost tracking
```
**Expected**: Cost optimization works correctly

## Verification Checklist

- [ ] cmd/studio builds successfully
- [ ] cmd/studio starts without errors
- [ ] Provider registry created
- [ ] Providers registered successfully
- [ ] Smart router created
- [ ] Test request routed successfully
- [ ] Fallback logic works
- [ ] Cost optimization works
- [ ] No performance degradation

## Timeline

- **Step 1**: Add import (5 minutes)
- **Step 2**: Create provider registry (10 minutes)
- **Step 3**: Register providers (20 minutes)
- **Step 4**: Create smart router (15 minutes)
- **Step 5**: Replace hardcoded adapters (20 minutes)
- **Step 6**: Test router (15 minutes)
- **Testing**: Build and test (30 minutes)
- **Total**: ~2 hours

## Success Criteria

1. ✅ cmd/studio builds successfully
2. ✅ cmd/studio starts without errors
3. ✅ Provider registry created
4. ✅ Providers registered successfully
5. ✅ Smart router created
6. ✅ Test request routed successfully
7. ✅ Fallback logic works
8. ✅ Cost optimization works
9. ✅ No performance degradation

## Notes

- This is a **gradual migration** - hardcoded adapters can coexist with Smart Router
- **No breaking changes** - can revert easily if needed
- **Incremental testing** - test each step before proceeding
- **Environment variables** - API keys should be set via environment variables
