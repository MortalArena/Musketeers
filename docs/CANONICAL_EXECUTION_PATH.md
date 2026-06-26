# Canonical Execution Path

## 1. Purpose

This document defines the **single, unambiguous execution path** for every task in Musketeers.
Before fixing Policy wiring, Persistence, Orchestrator routing, or any other cross-cutting concern,
this path must be established as the source of truth. Every component's role, authority, 
and connection points flow from this path.

---

## 2. The Path

```
                      ┌──────────────────────┐
                      │      Human/Client     │
                      │  (API / WebSocket /   │
                      │   CLI / Channel)      │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Identity & AuthN    │  ← Layer 0: Who is this?
                      │   (DID / Token /      │
                      │    Session Key)       │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Session Manager     │  ← Layer 1: Scope & context
                      │   - Validate session  │
                      │   - Load metadata     │
                      │   - Check state       │
                      │   - Apply quotas      │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Workflow Engine     │  ← Layer 2: What process?
                      │   - Parse intent      │
                      │   - Select workflow   │
                      │   - Decompose task    │
                      │   - Define steps      │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Orchestrator        │  ← Layer 3: Who does it?
                      │   - Find best agent   │
                      │   - Assign roles      │
                      │   - Route subtasks    │
                      │   - Track progress    │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Thinking / Planning │  ← Layer 4: How?
                      │   - Analyze           │
                      │   - Plan              │
                      │   - DAG execution     │
                      │   - Adapt & replan    │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Delegation Layer    │  ← Layer 5: Sub-agent
                      │   - Sub-task routing  │  coordination
                      │   - Merge results     │
                      │   - Handle failures   │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Capability & Policy │  ← Layer 6: May I?
                      │   Pipeline:           │
                      │   ├ Authorization     │
                      │   ├ Capability check  │
                      │   ├ Resource quota    │
                      │   ├ Rate limit        │
                      │   └ Audit log         │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Tool Execution      │  ← Layer 7: Do it
                      │   - Adapter routing   │
                      │   - Tool handler      │
                      │   - Result capture    │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Learning Layer      │  ← Layer 8: Remember
                      │   - Extract insights  │
                      │   - Update memory     │
                      │   - Skill refinement  │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │   Aggregation Layer   │  ← Layer 9: Build response
                      │   - Collect results   │
                      │   - Summarize         │
                      │   - Format output     │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │      Response         │
                      │  (to Human/Client)    │
                      └──────────────────────┘
```

---

## 3. Layer Responsibilities

### Layer 0: Identity & Authentication
- **Responsibility**: Verify the caller's identity (DID, token, API key)
- **Current state**: `pkg/identity` exists, DIDs used for session creation
- **Policy scope**: None — this is pre-authorization
- **Persistence**: None needed

### Layer 1: Session Manager
- **Responsibility**: Create, validate, scope, and track sessions
- **Current state**: `core.UnifiedSessionManager` exists and creates sessions
- **Policy scope**: Session-level quotas, session state validation
- **Persistence**: Save session metadata on creation, state changes, close

### Layer 2: Workflow Engine
- **Responsibility**: Decompose user intent into a structured workflow with steps
- **Current state**: `session.WorkflowEngine` exists but `Execute16StepWorkflow()` is dead code.
  ThinkingEngine has its own parallel 16-step workflow (`ExecuteWith16Steps`, `Execute16StepWorkflow`)
- **Policy scope**: Workflow type validation, step sequencing
- **Persistence**: Save workflow definition, current step, step results

### Layer 3: Orchestrator
- **Responsibility**: Route tasks to the right agent(s), track execution, merge results
- **Current state**: `OrchestratorEngine` exists but `ExecuteTask()` is never called.
  `Connector` handles message routing (EventBus ↔ Bridge) but is not connected to execution.
  The actual execution path bypasses both and goes directly through `UnifiedAgent.ExecuteTaskWithThinking()`
- **Policy scope**: Agent selection policy, delegation rules, capacity management
- **Persistence**: Save task assignments, agent results, delegation chains

### Layer 4: Thinking / Planning
- **Responsibility**: Analyze tasks, generate plans, execute DAG workflows
- **Current state**: `ThinkingEngine` in `pkg/agent/thinking` — the most mature layer.
  Has multiple 16-step flows, DAG execution, agent coordination, learning from sessions
- **Policy scope**: None (pure reasoning)
- **Persistence**: Save thoughts, plans, intermediate results (currently only in-memory)

### Layer 5: Delegation
- **Responsibility**: Decompose tasks into sub-tasks, route to sub-agents, collect results
- **Current state**: `pkg/delegation` package exists (integration initialized), but `DelegationManager` 
  on ThinkingEngine is **always nil** — never wired from `main.go`
- **Policy scope**: Delegation depth limits, trust boundaries, result validation
- **Persistence**: Save delegation trees, sub-task results

### Layer 6: Capability & Policy Pipeline
- **Responsibility**: Authorize and control every capability execution through a middleware chain
- **Current state**: `capability.Manager` exists but `policy.Engine.Evaluate()` is never called there.
  `pipeline.AuthorizationMiddleware` exists but is not wired into `Manager.Execute()`.
  `policyEngine` in `main.go` (lines 694-774) is orphaned — created with 5 rules, never connected
- **Policy scope**: Authorization, capability gating, resource quotas, rate limits, audit
- **Persistence**: Save policy decisions, audit logs, quota consumption

### Layer 7: Tool Execution
- **Responsibility**: Execute actual operations via adapters and tool handlers
- **Current state**: `ToolExecutor.ExecuteTool()` in ThinkingEngine handles file, web, HTTP tools.
  Adapter layer (CLI, IDE, Browser, Custom) registered in AgentRegistry
- **Policy scope**: Tool permissions, argument validation, output sanitization
- **Persistence**: Save tool call results, timing, error details

### Layer 8: Learning
- **Responsibility**: Extract insights from execution, update memory, improve skills
- **Current state**: `CollectiveLearning` in ThinkingEngine, `LearnFromSession` methods exist
  but SessionContainer.Save() is never called — learning is in-memory only
- **Persistence**: Save learned patterns, skill refinements, error patterns

### Layer 9: Aggregation
- **Responsibility**: Collect all results, build coherent response for the caller
- **Current state**: Handled inline in ThinkingEngine methods
- **Persistence**: Save final response

---

## 4. Current vs Target

| Layer | Current Implementation | Dead / Bypassed | Target |
|-------|----------------------|-----------------|--------|
| 0 | `pkg/identity` | — | Keep, add token validation |
| 1 | `core.UnifiedSessionManager` | — | Keep, add Save() on changes |
| 2 | ThinkingEngine self-contained 16-step | `WorkflowEngine.Execute16StepWorkflow()` dead | Merge into single path |
| 3 | **BYPASSED** — `UnifiedAgent.ExecuteTaskWithThinking()` | `OrchestratorEngine.ExecuteTask()` dead, `Connector` unused | Wire OrchestratorEngine as the sole entry |
| 4 | ThinkingEngine (mature) | — | Keep, simplify to one 16-step |
| 5 | `pkg/delegation` exists | `ThinkingEngine.delegationManager` always nil | Wire delegation when needed |
| 6 | **BYPASSED** — `capability.Manager.Execute()` calls capability directly | `policy.Engine.Evaluate()` dead, pipeline dead, `main.go` policy engine orphaned | Build Execution Pipeline middleware |
| 7 | ToolExecutor + adapters | BrowserAdapter stubs return ErrNotImplemented | Keep, implement adapters when needed |
| 8 | Methods exist | `Save()` never called, all in-memory | Hybrid Persistence (dirty + periodic + critical) |
| 9 | Inline in ThinkingEngine | — | Keep, formalize |

---

## 5. How This Unblocks the 3 Blocked P0 Decisions

### P0.5: Policy Wiring

**Before this path**: We don't know where to put policy. Manager.Execute() seems wrong.

**After this path**: Policy goes at Layer 6, as a middleware PIPELINE (not a single call).
The pipeline wraps `Manager.ExecuteInternal()` (the internal method that actually runs the capability).
Callers above Layer 6 (Orchestrator, Workflow, etc.) do NOT see policy — they just call Execute().
Policy is transparent to them.

Implementation:
```go
// capability/manager.go
func (m *Manager) Execute(ctx context.Context, principal Principal, cmd Command) (*Result, error) {
    return m.pipeline.Execute(ctx, principal, cmd, m.executeInternal)
}

func (m *Manager) executeInternal(ctx context.Context, principal Principal, cmd Command) (*Result, error) {
    capability, exists := m.capabilities[cmd.Name()]
    // ... existing code, no change needed
}
```

The pipeline middleware chain processes in order:
1. AuthorizationMiddleware → Evaluates policy.Engine
2. CapabilityMiddleware → Checks if capability exists and is enabled
3. QuotaMiddleware → Checks resource limits
4. AuditMiddleware → Logs the execution
5. → executeInternal

### P0.8: Periodic Save

**Before this path**: We don't know when Save() should fire. Every event? Timer?

**After this path**: Each layer defines its persistence contract:
- Layer 1 (Session): Save on creation, state change, close. No periodic needed.
- Layer 2 (Workflow): Save on step completion, workflow start/end. 30s periodic flush if dirty.
- Layer 3 (Orchestrator): Save on task assignment, completion. Immediate on critical events.
- Layer 4 (Thinking): Save on phase change, thought added. Periodic flush (15s).
- Layer 6 (Policy): Save on policy evaluation result. No periodic needed — replay from audit log.
- Layer 7 (Tool): Save on tool call result. Immediate for side-effect tools.

Hybrid strategy:
```
Critical Event → Immediate Save
(Layer 1 close, Layer 3 task done, Layer 7 side-effect tool)

Non-Critical Change → Mark Dirty
(Layer 4 thought added, Layer 2 step progress)

Periodic Flush (30s) → Save all dirty layers together in one transaction

Shutdown → Final SaveAll with timeout
```

### P0.10: Orchestrator.ExecuteTask()

**Before this path**: Should we call ExecuteTask() from main.go? Or keep UnifiedAgent as entry?

**After this path**: The canonical path MUST go through Orchestrator at Layer 3.
Current path: `main.go → UnifiedAgent.ExecuteTaskWithThinking()` becomes:
`main.go → OrchestratorEngine.ExecuteTask() → ThinkingEngine.ExecuteSteps()`

UnifiedAgent becomes an **internal orchestrator implementation**, not the entry point.
The entry point for every task is:

```go
// orchestrator/engine.go
func (oe *OrchestratorEngine) ExecuteTask(ctx context.Context, task Task) (*Result, error) {
    // 1. Find best agent
    agent := oe.registry.FindBestAgent(task)
    
    // 2. Route to the right executor
    if agent.Type == AgentTypeThinking {
        return oe.thinkingEngine.ExecuteWithWorkflow(ctx, task)
    }
    
    // 3. Or delegate to sub-agents
    return oe.delegateTask(ctx, task, agent)
}
```

---

## 6. Migration Plan

### Phase A: Establish the Path (Next PR)
1. Create `orchestrator/executor.go` — thin adapter that routes from `OrchestratorEngine.ExecuteTask()` 
   to `ThinkingEngine.ExecuteSteps()` (one-directional, no behavioral change)
2. Change `main.go` to call `orchestratorEngine.ExecuteTask()` instead of 
   `unifiedAgent.ExecuteTaskWithThinking()`
3. Build ✨ passes, no behavioral change — just path alignment

### Phase B: Build the Policy Pipeline (Next PR)
1. Create `ExecutionPipeline` struct in `pkg/capability/pipeline/` 
2. Wire it into `Manager.Execute()`
3. Register the orphaned `policyEngine` from `main.go` into the pipeline
4. Start in **logging-only** mode — policy logs what would be denied but allows execution
5. After validation, switch to enforcement mode

### Phase C: Hybrid Persistence (Next PR)
1. Add `Dirty` flag to SessionContainer
2. Add 30s periodic ticker in studio/main.go
3. Implement critical event saves (session close, task complete)
4. Wire `SaveAll()` on shutdown

### Phase D: Cleanup (Ongoing)
1. Remove dead `WorkflowEngine.Execute16StepWorkflow()`
2. Merge the three ThinkingEngine 16-step implementations into one
3. Wire DelegationManager from main.go when delegation is wanted
4. Remove `UnifiedAgent.ExecuteTaskWithThinking()` direct call after Phase A confirmed stable
