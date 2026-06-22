# Plan: Resolve Duplications (integration vs orchestrator)

## Overview
This plan details how to resolve the duplication between pkg/integration and pkg/orchestrator.

## Current State
- pkg/integration exists but is completely unused
- pkg/orchestrator is used by cmd/studio
- Both packages have similar functionality
- Unclear which package should be used

## Comparison Analysis

### pkg/integration Components
**File**: pkg/integration/integration.go

**Components**:
1. **AgentSessionIntegration** - Manages agent sessions
2. **TaskRouting** - Routes tasks to agents
3. **SessionOrchestrator** - Orchestrates sessions

**Key Features**:
- Session management
- Task routing
- Agent coordination
- Integration with external systems

**Dependencies**:
- pkg/session
- pkg/agent
- pkg/eventbus

**Importers**: ❌ NONE (completely unused)

### pkg/orchestrator Components
**Files**: pkg/orchestrator/*.go

**Components**:
1. **SessionManager** - Manages sessions
2. **Connector** - Connects bridge, event bus, and agent registry
3. **DelegationManager** - Manages task delegation
4. **ChatConnector** - Connects chat and channels
5. **ExternalPlatformManager** - Manages external platforms

**Key Features**:
- Session management
- Task delegation
- Chat integration
- External platform management
- Bridge connection

**Dependencies**:
- pkg/session
- pkg/agent
- pkg/eventbus
- pkg/policy

**Importers**: ✅ cmd/studio

## Functional Overlap

| Feature | pkg/integration | pkg/orchestrator | Overlap |
|---------|----------------|------------------|---------|
| Session Management | AgentSessionIntegration | SessionManager | ✅ YES |
| Task Routing | TaskRouting | DelegationManager | ✅ YES |
| Agent Coordination | SessionOrchestrator | Connector | ✅ YES |
| External Integration | SessionOrchestrator | ExternalPlatformManager | ✅ YES |
| Chat Integration | None | ChatConnector | ❌ NO |
| Bridge Connection | None | Connector | ❌ NO |

## Recommendation

### Option 1: Keep pkg/orchestrator, Remove pkg/integration
**Rationale**:
- pkg/orchestrator is actively used by cmd/studio
- pkg/orchestrator has more features (chat, bridge, external platforms)
- pkg/integration is completely unused
- Removing pkg/integration eliminates duplication

**Pros**:
- ✅ Removes unused code
- ✅ Eliminates confusion
- ✅ Reduces maintenance burden
- ✅ No breaking changes (pkg/integration unused)

**Cons**:
- ❌ Loses pkg/integration functionality (if needed in future)
- ❌ Need to verify no hidden dependencies

**Risk**: 🟢 LOW - pkg/integration is unused

### Option 2: Merge pkg/integration into pkg/orchestrator
**Rationale**:
- Combine best features from both packages
- Create unified orchestration system
- Preserve all functionality

**Pros**:
- ✅ Preserves all functionality
- ✅ Creates unified system
- ✅ Can pick best features from both

**Cons**:
- ❌ More complex migration
- ❌ Potential conflicts
- ❌ More testing required

**Risk**: 🟡 MEDIUM - Merge complexity

### Option 3: Keep Both, Document Differences
**Rationale**:
- Keep both packages for different use cases
- Document when to use each
- Avoid breaking changes

**Pros**:
- ✅ No code changes
- ✅ No breaking changes
- ✅ Preserves all functionality

**Cons**:
- ❌ Confusion remains
- ❌ Maintenance burden
- ❌ Duplication persists

**Risk**: 🟢 LOW - No changes

## Recommended Action

### **Option 1: Keep pkg/orchestrator, Remove pkg/integration**

**Justification**:
1. pkg/integration is completely unused (no importers)
2. pkg/orchestrator is actively used by cmd/studio
3. pkg/orchestrator has more features
4. Removing unused code is best practice
5. Low risk (no breaking changes)

## Implementation Plan

### Step 1: Verify No Hidden Dependencies
**Command**:
```bash
grep -r "pkg/integration" --include="*.go" .
```
**Expected**: No results (or only in pkg/integration itself)

### Step 2: Verify No Documentation References
**Command**:
```bash
grep -r "integration" --include="*.md" docs/
```
**Expected**: No critical references to pkg/integration

### Step 3: Verify Git History
**Command**:
```bash
git log --all --full-history --oneline -- pkg/integration
```
**Expected**: Understand history before deletion

### Step 4: Get Explicit User Approval
**Action**: Ask user for explicit approval before deletion

### Step 5: Delete pkg/integration
**Command**:
```bash
rm -rf pkg/integration
```

### Step 6: Update Documentation
**Files**: README.md, ARCHITECTURE.md
**Action**: Remove references to pkg/integration

### Step 7: Commit Changes
**Command**:
```bash
git add -A
git commit -m "Remove unused pkg/integration (duplicate of pkg/orchestrator)"
```

## Rollback Plan

### If Deletion Causes Issues
1. Restore pkg/integration from git
2. Verify no breaking changes
3. Investigate why deletion caused issues

### Rollback Commands
```bash
git checkout HEAD~1 -- pkg/integration
```

## Testing Plan

### Test 1: Build
```bash
go build ./...
```
**Expected**: Build succeeds without errors

### Test 2: Test cmd/studio
```bash
go build ./cmd/studio
./studio
```
**Expected**: cmd/studio works correctly

### Test 3: Test All Entry Points
```bash
go build ./cmd/agent
go build ./cmd/founder
go build ./cmd/gateway
go build ./cmd/seed
```
**Expected**: All entry points build successfully

### Test 4: Run Tests
```bash
go test ./...
```
**Expected**: All tests pass

## Verification Checklist

- [ ] No hidden dependencies found
- [ ] No documentation references found
- [ ] Git history reviewed
- [ ] Explicit user approval obtained
- [ ] pkg/integration deleted
- [ ] Documentation updated
- [ ] Build succeeds
- [ ] cmd/studio works correctly
- [ ] All entry points work correctly
- [ ] All tests pass

## Timeline

- **Step 1**: Verify dependencies (10 minutes)
- **Step 2**: Verify documentation (10 minutes)
- **Step 3**: Review git history (10 minutes)
- **Step 4**: Get user approval (waiting on user)
- **Step 5**: Delete pkg/integration (5 minutes)
- **Step 6**: Update documentation (15 minutes)
- **Step 7**: Commit changes (5 minutes)
- **Testing**: Build and test (30 minutes)
- **Total**: ~1.5 hours (excluding user approval wait time)

## Success Criteria

1. ✅ No hidden dependencies found
2. ✅ No documentation references found
3. ✅ Explicit user approval obtained
4. ✅ pkg/integration deleted
5. ✅ Documentation updated
6. ✅ Build succeeds
7. ✅ cmd/studio works correctly
8. ✅ All entry points work correctly
9. ✅ All tests pass

## Alternative: If User Prefers to Keep pkg/integration

If user prefers to keep pkg/integration, we can:

### Option A: Document Use Cases
- Document when to use pkg/integration
- Document when to use pkg/orchestrator
- Add clear examples

### Option B: Merge into UnifiedAgent
- Merge pkg/integration into pkg/agent/unified
- Merge pkg/orchestrator into pkg/agent/unified
- Create unified orchestration system

### Option C: Rename for Clarity
- Rename pkg/integration to pkg/integration_legacy
- Rename pkg/orchestrator to pkg/orchestrator_active
- Add deprecation notice to pkg/integration

## Notes

- **Triple review required** - Must verify no dependencies before deletion
- **User approval required** - Must get explicit approval before deletion
- **Low risk** - pkg/integration is unused
- **Best practice** - Removing unused code is recommended
