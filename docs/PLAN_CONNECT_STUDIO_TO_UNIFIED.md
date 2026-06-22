# Plan: Connect cmd/studio to UnifiedAgent

## Overview
This plan details how to connect cmd/studio to the UnifiedAgent system, replacing the current agent_bridge + orchestrator approach.

## Current State
- cmd/studio uses agent_bridge + orchestrator
- UnifiedAgent exists but is completely unused
- No coordination between agents
- No shared skills
- No shared memory

## Target State
- cmd/studio uses UnifiedAgent
- Coordinated agent execution
- Shared skills
- Shared memory
- Real-time synchronization

## Required Modifications

### Step 1: Add Import
**File**: cmd/studio/main.go
**Location**: Top of file (after existing imports)

```go
import (
    // ... existing imports ...
    "github.com/MortalArena/Musketeers/pkg/agent/unified"
)
```

### Step 2: Create UnifiedAgent Instance
**File**: cmd/studio/main.go
**Location**: After sessionContainer creation (around line 179)

```go
// إنشاء UnifiedAgent
unifiedAgent := unified.NewUnifiedAgent(
    sessionContainer.ID,
    kp.DID,
    db,
    zapLogger,
)
log.Info("UnifiedAgent created")
```

### Step 3: Initialize UnifiedAgent
**File**: cmd/studio/main.go
**Location**: After UnifiedAgent creation (around line 185)

```go
// تهيئة UnifiedAgent
if err := unifiedAgent.Initialize(ctx); err != nil {
    log.WithError(err).Fatal("Failed to initialize unified agent")
}
log.Info("UnifiedAgent initialized")
```

### Step 4: Register Existing Agents in UnifiedAgent
**File**: cmd/studio/main.go
**Location**: After agent registration (around line 165)

```go
// تسجيل الوكلاء في UnifiedAgent
for _, adapter := range agentRegistry.GetAll() {
    if err := unifiedAgent.RegisterAgent(ctx, adapter.DID(), adapter.Type(), adapter.LLMType(), adapter.Specializations()); err != nil {
        log.WithError(err).Warnf("Failed to register agent %s in unified system", adapter.DID())
    }
}
log.WithField("agent_count", agentRegistry.GetCount()).Info("Agents registered in unified system")
```

### Step 5: Replace agent_bridge + orchestrator with UnifiedAgent
**File**: cmd/studio/main.go
**Location**: Replace agent_bridge + orchestrator initialization (around lines 195-250)

**REMOVE**:
```go
// ❌ Remove this entire section:
// إنشاء Session Manager
sessionMgr := agent_bridge.NewSessionManager(log)

// إنشاء Multiplexed Bridge
multiplexedBrg := agent_bridge.NewMultiplexedBridge(log)

// إنشاء Connector
connector := pkgOrchestrator.NewConnector(eb, multiplexedBrg, agentRegistry, zapLogger)

// إنشاء Delegation Manager
delegationMgr := pkgOrchestrator.NewDelegationManager(eb, agentRegistry, zapLogger)

// إنشاء Chat Connector
chatConnector := pkgOrchestrator.NewChatConnector(eb, agentRegistry, zapLogger)

// إنشاء External Platform Manager
externalPlatformMgr := pkgOrchestrator.NewExternalPlatformManager(eb, policyEngine, zapLogger)
```

**ADD**:
```go
// ✅ Use UnifiedAgent instead:
// UnifiedAgent handles all coordination internally
log.Info("UnifiedAgent handles agent coordination")
```

### Step 6: Test UnifiedAgent Execution
**File**: cmd/studio/main.go
**Location**: After UnifiedAgent initialization (around line 190)

```go
// اختبار تنفيذ مهمة
testTask := "تحليل ملفات المشروع"
result, err := unifiedAgent.ExecuteTask(ctx, testTask)
if err != nil {
    log.WithError(err).Warn("Failed to execute test task")
} else {
    log.WithField("success", result.Success).WithField("confidence", result.Confidence).Info("Test task executed")
}
```

## Dependencies Required
- ✅ `github.com/MortalArena/Musketeers/pkg/agent/unified` - exists
- ✅ `github.com/dgraph-io/badger/v4` - exists
- ✅ `go.uber.org/zap` - exists

## Potential Risks

### Risk 1: Breaking agent_bridge for cmd/agent
**Impact**: cmd/agent also uses agent_bridge
**Mitigation**: Keep agent_bridge for cmd/agent, only replace in cmd/studio
**Fallback**: Revert changes if cmd/agent breaks

### Risk 2: UnifiedAgent initialization failure
**Impact**: cmd/studio fails to start
**Mitigation**: Add error handling and graceful degradation
**Fallback**: Fall back to agent_bridge + orchestrator if UnifiedAgent fails

### Risk 3: Performance degradation
**Impact**: Slower agent execution
**Mitigation**: Monitor performance metrics
**Fallback**: Disable UnifiedAgent if performance is unacceptable

## Rollback Plan

### If Integration Fails
1. Revert cmd/studio/main.go to previous version
2. Remove UnifiedAgent import
3. Restore agent_bridge + orchestrator code
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
**Expected**: cmd/studio starts without errors, UnifiedAgent initializes successfully

### Test 3: Execute Task
```bash
# Send task via API or CLI
```
**Expected**: UnifiedAgent executes task successfully

### Test 4: Check Agent Registration
```bash
# Check logs for agent registration
```
**Expected**: All agents registered in unified system

### Test 5: Check System Summary
```bash
# Query system summary via API
```
**Expected**: System summary shows all subsystems ready

## Verification Checklist

- [ ] cmd/studio builds successfully
- [ ] cmd/studio starts without errors
- [ ] UnifiedAgent initializes successfully
- [ ] All agents registered in unified system
- [ ] Task execution works correctly
- [ ] System summary shows all subsystems ready
- [ ] No performance degradation
- [ ] cmd/agent still works (uses agent_bridge)

## Timeline

- **Step 1**: Add import (5 minutes)
- **Step 2**: Create UnifiedAgent instance (10 minutes)
- **Step 3**: Initialize UnifiedAgent (10 minutes)
- **Step 4**: Register agents (15 minutes)
- **Step 5**: Replace agent_bridge + orchestrator (30 minutes)
- **Step 6**: Test execution (15 minutes)
- **Testing**: Build and test (30 minutes)
- **Total**: ~2 hours

## Success Criteria

1. ✅ cmd/studio builds successfully
2. ✅ cmd/studio starts without errors
3. ✅ UnifiedAgent initializes successfully
4. ✅ All agents registered in unified system
5. ✅ Task execution works correctly
6. ✅ System summary shows all subsystems ready
7. ✅ No performance degradation
8. ✅ cmd/agent still works (uses agent_bridge)

## Notes

- This is a **gradual migration** - agent_bridge + orchestrator can coexist with UnifiedAgent
- **No breaking changes** - can revert easily if needed
- **Incremental testing** - test each step before proceeding
- **Monitor performance** - ensure no degradation
