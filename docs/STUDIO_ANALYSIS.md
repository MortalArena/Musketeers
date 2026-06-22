# cmd/studio Analysis

## Overview
cmd/studio is the main entry point for the Musketeers Studio interface. This analysis examines what it does, what it uses, and what it's missing.

## What cmd/studio Does

### Initialization
1. **Key Generation**: Generates Ed25519 key pair using pkg/crypto
2. **Identity Creation**: Creates identity record with 1-year TTL using pkg/identity
3. **Node Creation**: Creates P2P node using pkg/node with:
   - Data directory
   - Listen port 4001
   - Storage quota 2GB
   - Founder public key (optional)
   - Bootstrap peers (optional)
4. **Identity Publishing**: Publishes identity to DHT
5. **Quota Manager**: Creates quota manager with 2GB limit for Studio
6. **Event Bus**: Creates event bus for event handling
7. **BadgerDB**: Opens BadgerDB for persistent storage

### Agent Registration
Registers 6 different agent adapters:
1. **API Adapter**: Anthropic API (claude-3-opus)
2. **CLI Adapter**: claude-code CLI
3. **IDE Adapter**: Cursor IDE
4. **Local Adapter**: Ollama (llama2)
5. **Browser Adapter**: Computer Use adapter
6. **Custom Adapter**: Custom agent with user-defined function

### Session Management
1. **Session Container**: Creates session container with:
   - Name: "Default Session"
   - Owner DID
   - Max agents: 10
   - Project type: "general"
2. **Tool Executor**: Creates tool executor with security limits
3. **CEO Supervisor**: Starts CEO supervisor for network health monitoring

### Orchestrator Components
1. **Session Manager**: Manages sessions with agent registry and event bus
2. **Delegation Manager**: Manages task delegation
3. **Connector**: Connects bridge, event bus, and agent registry
4. **Chat Connector**: Connects chat and channels
5. **External Platform Manager**: Manages external platforms with policy engine

### Verification Components
1. **Multi-Stage Verifier**: Creates verifier with:
   - Syntax verifier
   - Semantics verifier
   - Security verifier
   - Performance verifier
   - Integration verifier

### ACP Handler
1. **Router**: Creates ACP router (registers built-in tasks)

### Agent Bridge Components
1. **Session Manager**: Creates agent bridge session manager
2. **Multiplexed Bridge**: Creates multiplexed bridge
3. **Connector**: Creates connector (already created above)
4. **Server**: Creates agent bridge server on agent-addr (default 127.0.0.1:5001)

### Policy Engine
1. **Engine**: Creates policy engine
2. **Default Rule**: Adds "default-deny" rule (DENY everything)
3. **Capability Manager**: Creates capability manager with policy engine

### HTTP Server
1. **Simple HTTP Server**: Creates HTTP server on addr (default 127.0.0.1:5000)
2. **Handler**: Returns "Musketeers Studio is running" for all requests

## What cmd/studio Uses

### ✅ Used Packages
- pkg/agent (AgentRegistry)
- pkg/agent/adapters (API, CLI, IDE, Local, Browser, Custom adapters)
- pkg/agent/tools (ToolExecutor)
- pkg/agent_bridge (SessionManager, MultiplexedBridge, Server)
- pkg/orchestrator (SessionManager, Connector, DelegationManager, ChatConnector, ExternalPlatformManager)
- pkg/ceo (CEOSupervisor)
- pkg/eventbus (EventBus)
- pkg/policy (Engine)
- pkg/capability (Manager)
- pkg/session (SessionContainer)
- pkg/node (Node)
- pkg/crypto (GenerateKeyPair)
- pkg/identity (IdentityRecord)
- pkg/storage (QuotaManager)
- pkg/verification (MultiStageVerifier)
- pkg/acp (Router)
- BadgerDB

### ❌ Not Used Packages
- pkg/agent/unified (UnifiedAgent) - **CRITICAL**
- pkg/providers (Smart Router, 34+ providers) - **CRITICAL**
- api/ (REST API, Dashboard) - **CRITICAL**
- pkg/integration (AgentSessionIntegration, TaskRouting) - **DUPLICATE**

## What cmd/studio Is Missing

### 1. UnifiedAgent Integration
**Current State**: Uses agent_bridge + orchestrator instead of UnifiedAgent
**Missing**:
- UnifiedSkillManager (skill management)
- UnifiedMemoryManager (memory management)
- SubagentManager (subagent coordination)
- AutomationManager (automation)
- SkillDirector (skill direction)
- MultiLayerValidator (validation)
- Coordinator (central coordination)
- FlowManager (flow management)
- ErrorHandler (error handling)
- CollectiveAgentSystem (collective system)
- SessionEventBus (event bus)
- RealTimeMemorySync (real-time memory sync)
- RealTimeSkillSync (real-time skill sync)
- ProblemSolutionRegistry (problem/solution registry)
- LocalMemoryCache (local memory cache)
- DataCurator (data curation)
- TaskScheduler (task scheduling)
- AgentSyncManager (sync management)

**Impact**: 
- No coordinated agent execution
- No shared skills
- No shared memory
- No real-time synchronization
- No problem/solution tracking
- No data curation

### 2. Provider Integration
**Current State**: Uses hardcoded API adapters instead of Smart Router
**Missing**:
- Smart Router (intelligent model selection)
- ProviderRegistry (provider management)
- APIKeyManager (API key management)
- FreeModelsTracker (free models tracking)
- FreeRouter (free model routing)
- ModelCatalog (model catalog)
- 34+ builtin providers (OpenAI, Anthropic, Google, DeepSeek, Groq, Perplexity, TogetherAI, Ollama, etc.)

**Impact**:
- No intelligent model selection
- No cost optimization
- No latency optimization
- No quality optimization
- No free model tracking
- No fallback logic
- Hardcoded API keys

### 3. API Integration
**Current State**: Simple HTTP server returning "Musketeers Studio is running"
**Missing**:
- REST API server with full endpoints
- Dashboard (web interface)
- WebSocket bridge
- Rate limiting
- TLS support
- Authentication

**Impact**:
- No REST API
- No web dashboard
- No WebSocket support
- No rate limiting
- No TLS
- No authentication

### 4. Security Features
**Current State**: 
- TLS not enabled in Agent Bridge
- No authentication in Agent Bridge
- ABAC only has default-deny rule
- SSRF protection incomplete (no CheckRedirect)

**Missing**:
- TLS configuration for Agent Bridge
- Authentication for Agent Bridge
- Allow rules for ABAC
- CheckRedirect for SSRF protection

**Impact**:
- Unencrypted communication
- Unauthorized access
- System rejects everything
- SSRF vulnerability

## Comparison with What It Should Do

### Current Implementation
```go
// ❌ Current: Uses agent_bridge + orchestrator
sessionMgr := agent_bridge.NewSessionManager(log)
multiplexedBrg := agent_bridge.NewMultiplexedBridge(log)
connector := pkgOrchestrator.NewConnector(eb, multiplexedBrg, agentRegistry, zapLogger)

// ❌ Current: Uses hardcoded API adapters
apiAdapter := pkgAdapters.NewAPIAdapter(apiConfig)
agentRegistry.Register(apiAdapter, nil)

// ❌ Current: Simple HTTP server
mux := http.NewServeMux()
mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Musketeers Studio is running"))
})
```

### Should Be
```go
// ✅ Should Use: UnifiedAgent
unifiedAgent := unified.NewUnifiedAgent(
    sessionContainer.ID,
    "studio-agent",
    db,
    zapLogger,
)
if err := unifiedAgent.Initialize(ctx); err != nil {
    log.WithError(err).Fatal("Failed to initialize unified agent")
}

// ✅ Should Use: Smart Router
providerRegistry := providers.NewRegistry()
providerRegistry.Register(providers.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")))
providerRegistry.Register(providers.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY")))
router := providers.NewSmartRouter(providerRegistry)

// ✅ Should Use: REST API
apiServer, err := api.NewServer(apiConfig, unifiedAgent, providerRegistry)
if err != nil {
    log.WithError(err).Fatal("Failed to create API server")
}
go func() {
    if err := apiServer.Start(); err != nil {
        log.WithError(err).Fatal("API server failed")
    }
}()
```

## Summary

### Critical Issues
1. **Does not use UnifiedAgent** - The unified system exists but is completely unused
2. **Does not use pkg/providers** - Smart Router and 34+ providers exist but are not used
3. **Does not use api/** - Full REST API with Dashboard exists but is not used
4. **Simple HTTP server** - Only returns "Musketeers Studio is running"

### Security Issues
1. **TLS not enabled** in Agent Bridge
2. **No authentication** in Agent Bridge
3. **ABAC non-functional** (only default-deny rule)
4. **SSRF vulnerability** (no CheckRedirect)

### Recommendations
1. Replace agent_bridge + orchestrator with UnifiedAgent
2. Replace hardcoded API adapters with Smart Router
3. Replace simple HTTP server with REST API
4. Enable TLS in Agent Bridge
5. Add authentication to Agent Bridge
6. Add allow rules to ABAC
7. Add CheckRedirect for SSRF protection
