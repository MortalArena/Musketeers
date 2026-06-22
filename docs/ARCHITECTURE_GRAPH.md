# Architecture Graph - Musketeers Project

## Overview
This document provides a complete dependency graph of the Musketeers project, showing all packages, their dependencies, and entry points.

## Entry Points (cmd/*)

### cmd/studio
- **Purpose**: Main studio interface
- **Dependencies**:
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
- **Status**: ✅ Working but NOT using UnifiedAgent

### cmd/agent
- **Purpose**: P2P agent node
- **Dependencies**: pkg/agent_bridge, pkg/node, pkg/crypto, pkg/identity
- **Status**: ✅ Working

### cmd/founder
- **Purpose**: Founder node
- **Dependencies**: pkg/node, pkg/crypto, pkg/identity
- **Status**: ✅ Working

### cmd/gateway
- **Purpose**: HTTP/HTTPS gateway
- **Dependencies**: pkg/gateway, pkg/node
- **Status**: ✅ Working

### cmd/seed
- **Purpose**: Seed node
- **Dependencies**: pkg/node, pkg/crypto, pkg/identity
- **Status**: ✅ Working

## Isolated Packages (No Importers)

### pkg/agent/unified
- **Components**:
  - UnifiedAgent (main coordinator)
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
- **Dependencies**: pkg/agent/automation, pkg/agent/direction, pkg/agent/integration, pkg/agent/subagents, pkg/agent/validation, pkg/session, BadgerDB
- **Importers**: ❌ NONE (completely unused!)
- **Status**: 🔴 ISOLATED - Not used by any entry point

### pkg/providers
- **Components**:
  - Router (smart routing)
  - ProviderRegistry (provider management)
  - APIKeyManager (API key management)
  - FreeModelsTracker (free models tracking)
  - FreeRouter (free model routing)
  - ModelCatalog (model catalog)
  - 34+ builtin providers (OpenAI, Anthropic, Google, DeepSeek, Groq, Perplexity, TogetherAI, Ollama, etc.)
- **Dependencies**: zap, sync, time, sort, fmt
- **Importers**: ❌ NONE (completely unused!)
- **Status**: 🔴 ISOLATED - Not used by any entry point

### api/
- **Components**:
  - Server (REST API server)
  - Dashboard (web dashboard)
  - LocalWSBridge (WebSocket bridge)
- **Dependencies**: pkg/naming, pkg/node, pkg/protocol, pkg/security, libp2p-pubsub
- **Importers**: ❌ NONE (completely unused!)
- **Status**: 🔴 ISOLATED - Not used by any entry point

### pkg/integration
- **Components**:
  - AgentSessionIntegration
  - TaskRouting
  - SessionOrchestrator
- **Dependencies**: pkg/session, pkg/agent, pkg/eventbus
- **Importers**: ❌ NONE (completely unused!)
- **Status**: 🔴 ISOLATED - Not used by any entry point

## Partially Used Packages

### pkg/orchestrator
- **Components**:
  - SessionManager
  - Connector
  - DelegationManager
  - ChatConnector
  - ExternalPlatformManager
- **Importers**: cmd/studio
- **Status**: ✅ Used by cmd/studio
- **Issue**: ⚠️ Duplicates functionality in pkg/integration

### pkg/agent_bridge
- **Components**:
  - SessionManager
  - MultiplexedBridge
  - Server
  - Client
- **Importers**: cmd/studio, cmd/agent
- **Status**: ✅ Used by cmd/studio and cmd/agent
- **Issue**: ⚠️ Should be replaced by UnifiedAgent

## Security Issues

### 1. SSRF in pkg/agent/tools/executor.go
- **Location**: httpRequest function
- **Issue**: http.Client does not use CheckRedirect function
- **Impact**: DNS Rebinding / Redirect bypass possible
- **Severity**: 🔴 HIGH

### 2. Agent Bridge without TLS/Auth
- **Location**: pkg/agent_bridge/server.go
- **Issue**: cmd/studio does not enable TLS (SetTLSConfig not called)
- **Issue**: No authentication (generateSessionID is random)
- **Impact**: Unencrypted communication, unauthorized access
- **Severity**: 🔴 HIGH

### 3. ABAC non-functional
- **Location**: pkg/policy/engine.go
- **Issue**: Only rule is "default-deny"
- **Issue**: No allow rules exist
- **Impact**: System rejects everything
- **Severity**: 🟡 MEDIUM

## Duplicate Functionality

### pkg/integration vs pkg/orchestrator
- **pkg/integration**: AgentSessionIntegration, TaskRouting, SessionOrchestrator
- **pkg/orchestrator**: SessionManager, Connector, DelegationManager, ChatConnector, ExternalPlatformManager
- **Issue**: Similar functionality in two different packages
- **Recommendation**: Merge or remove one

## Summary

### Critical Issues
1. **cmd/studio does not use UnifiedAgent** - The unified system exists but is completely unused
2. **pkg/providers is completely unused** - 34+ AI providers with Smart Router exist but are not used
3. **api/ is completely unused** - Full REST API with Dashboard exists but is not used
4. **pkg/integration is completely unused** - Integration system exists but is not used

### Security Issues
1. **SSRF vulnerability** in pkg/agent/tools/executor.go (no CheckRedirect)
2. **Agent Bridge without TLS/Auth** in cmd/studio
3. **ABAC non-functional** (only default-deny rule)

### Architectural Issues
1. **Two parallel systems**: agent_bridge + orchestrator vs UnifiedAgent
2. **Duplicate functionality**: pkg/integration vs pkg/orchestrator
3. **Isolated packages**: 4 major packages completely unused

### Recommendations
1. Connect cmd/studio to UnifiedAgent
2. Connect cmd/studio to pkg/providers
3. Connect cmd/studio to api/
4. Fix SSRF vulnerability
5. Enable TLS/Auth in Agent Bridge
6. Add allow rules to ABAC
7. Resolve pkg/integration vs pkg/orchestrator duplication
