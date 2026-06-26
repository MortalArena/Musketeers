# Archived Adapters

## Archived Files

| File | Why Archived | Alternative | When to Delete |
|------|-------------|-------------|----------------|
| `api_adapter.go` | Replaced by `pkg/providers` (provider-based LLM access) | `builtin.NewRegistry()` in `cmd/studio/main.go:423` | After providers support full API adapter features |
| `local_adapter.go` | Replaced by `pkg/providers` (Ollama via provider registry) | `builtin.NewRegistry()` includes local providers | After local provider path validated |
| `hook_system.go` | Unused — no production caller | EventBus pattern in `pkg/eventbus` | Never — kept as reference for hook pattern |

## Still in Place (Coupled to Active Code)

These files remain in `pkg/agent/adapters/` with `// DEPRECATED` banners because they reference types
in the same package (CLIConfig, IDEConfig, AgentInstance). Clean removal requires refactoring their
dependencies first:

| File | Why Archived | Alternative |
|------|-------------|-------------|
| `desktop_adapter.go` | Desktop automation not in production scope | Direct OS integration when needed |
| `ide_extension_adapter.go` | IDE extension protocol not implemented | Direct IDE plugin development |
| `instance_manager.go` | Multi-instance management unused | AgentRegistry for single-instance |
| `multi_cli_adapter.go` | Example-only multi-instance CLI | AgentRegistry + CLIAdapter for single |
| `multi_desktop_adapter.go` | Example-only multi-instance desktop | N/A |
| `multi_ide_adapter.go` | Example-only multi-instance IDE | N/A |

## BrowserAdapter

The `BrowserAdapter` in `pkg/agent/adapters/browser_adapter.go` is kept as a placeholder but all
execution methods now return `ErrNotImplemented`. Use Computer Use SDK or Playwright directly
for real browser automation. Keep available for registration in AgentRegistry — it provides
metadata (name, type, capabilities) without pretending to execute.
