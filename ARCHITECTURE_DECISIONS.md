# Architecture Decisions - Musketeers Project

## Overview
This document captures the architectural decisions made during the design and implementation of the Musketeers project, a decentralized multi-agent system with cryptographic identity management, secure storage, and AI provider integration.

## Core Architectural Principles

### 1. Decentralized Identity System
**Decision**: Use Ed25519 cryptographic keys and DIDs (Decentralized Identifiers) for identity management.

**Rationale**:
- Ed25519 provides strong cryptographic security with small key sizes (32 bytes)
- DIDs are self-sovereign and don't rely on centralized authorities
- Domain-separated signatures prevent replay attacks across different contexts

**Implementation**:
- `pkg/crypto`: Ed25519 key generation, signing, and verification
- `pkg/identity`: Identity lifecycle management with DID generation
- Domain separation tags: `NR-IDENTITY-V1|`, `NR-DELEGATION-V1|`, `NR-REVOCATION-V1|`

### 2. Proof of Work for Identity Validation
**Decision**: Implement PoW (Proof of Work) mining for identity creation to prevent spam.

**Rationale**:
- Prevents automated identity creation attacks
- Adds computational cost to identity creation
- Difficulty can be adjusted based on network conditions

**Implementation**:
- `pkg/crypto/pow.go`: PoW mining and verification
- Difficulty levels configurable per node
- Integrated into identity creation workflow

### 3. Layered Architecture
**Decision**: Organize code into 5 dependency levels with clear separation of concerns.

**Rationale**:
- Promotes maintainability and testability
- Reduces coupling between components
- Enables independent development of layers

**Implementation**:
- Level 0: Infrastructure (common, protocol, policy, registry, storage, providers)
- Level 1: Security (crypto, identity, vault, capability, runtime, content, acp)
- Level 2: Agent System (agent, workflow, mailbox)
- Level 3: Communication (agent_bridge, session)
- Level 4: Orchestration (integration, orchestrator)

### 4. Policy-Based Access Control
**Decision**: Use a rule-based policy engine for access control across the system.

**Rationale**:
- Centralized security policy management
- Flexible rule definition with principals, resources, and conditions
- Multi-level approval support for sensitive operations

**Implementation**:
- `pkg/policy`: Policy engine with rule evaluation
- `pkg/capability`: Capability execution with policy enforcement
- `pkg/agent_bridge`: Tool validation with policy checks
- Multi-level approval system in `pkg/policy/approvals.go`

### 5. Erasure Coding for Data Redundancy
**Decision**: Use Reed-Solomon erasure coding (10 data + 4 parity shards) for storage.

**Rationale**:
- Provides fault tolerance (can tolerate up to 4 shard failures)
- More efficient than full replication
- Enables data reconstruction from partial availability

**Implementation**:
- `pkg/storage/erasure.go`: Reed-Solomon encoding/decoding
- Constants: `DataShards=10`, `ParityShards=4`, `TotalShards=14`

### 6. Quota Management for Storage
**Decision**: Implement per-DID storage quotas with 1GB default free tier.

**Rationale**:
- Prevents storage abuse
- Enables tiered pricing model
- Atomic quota checking prevents over-allocation

**Implementation**:
- `pkg/storage/quota.go`: Quota manager with atomic operations
- Default free tier: 1GB per DID
- `CheckAndAdd` method for atomic quota reservation
- `Release` method for quota cleanup on deletion

### 7. AES-256-GCM for Encryption
**Decision**: Use AES-256-GCM for all encryption operations (vault, API keys, mailbox).

**Rationale**:
- Industry-standard encryption algorithm
- Authenticated encryption prevents tampering
- 256-bit key size provides strong security

**Implementation**:
- `pkg/vault/encryption`: AES-GCM encryption/decryption
- `pkg/providers/api_key_manager`: API key encryption with scrypt key derivation
- `pkg/mailbox`: Message encryption with AES-GCM
- Scrypt parameters: N=131072, r=8, p=1, keylen=32

### 8. Multi-Agent Communication Protocol (ACP)
**Decision**: Design a custom protocol (ACP) for inter-agent communication over libp2p.

**Rationale**:
- Standardized message format for agent communication
- Built-in signature verification for security
- Support for task requests, responses, and errors

**Implementation**:
- `pkg/acp`: ACP protocol implementation
- Protocol version: `acp/v1`
- Protocol ID: `/nr/acp/1.0.0`
- Built-in tasks: ping, echo, translate, execute
- Message envelope with signature verification

### 9. Smart Routing for AI Providers
**Decision**: Implement intelligent routing with usage tracking and fallback.

**Rationale**:
- Automatic selection of best provider based on cost, latency, and success rate
- Fallback to alternative providers on failure
- Preference for free and local models when configured

**Implementation**:
- `pkg/providers/router.go`: Smart routing with scoring algorithm
- `pkg/providers/free_router.go`: Free model routing
- `pkg/providers/free_models_tracker.go`: Usage tracking for free models
- Scoring factors: free models, local models, cost optimization, latency optimization, success rate

### 10. Agent Lifecycle Management
**Decision**: Implement comprehensive agent lifecycle with health monitoring.

**Rationale**:
- Ensures agent availability and reliability
- Enables automatic recovery from failures
- Provides observability into agent status

**Implementation**:
- `pkg/agent/lifecycle.go`: Agent lifecycle management
- Health checks with configurable intervals
- Automatic restart on failure
- Metrics collection for monitoring

### 11. Session-Based Multi-Agent Workflows
**Decision**: Use sessions to coordinate multi-agent workflows.

**Rationale**:
- Provides context for multi-agent collaboration
- Enables task distribution and result aggregation
- Supports role-based agent assignment

**Implementation**:
- `pkg/session`: Session management with UnifiedSessionManager
- `pkg/integration/role_assignment.go`: Role assignment (manager, assistant, observer, specialist)
- `pkg/integration/task_routing.go`: Task routing to appropriate agents
- `pkg/integration/session_orchestrator.go`: Session coordination

### 12. Workflow Engine with Checkpointing
**Decision**: Implement a workflow engine with checkpointing for fault tolerance.

**Rationale**:
- Enables complex multi-step workflows
- Checkpointing allows recovery from failures
- Supports conditions, loops, and delays

**Implementation**:
- `pkg/workflow/engine.go`: Workflow execution engine
- `pkg/workflow/checkpoint.go`: Checkpoint management with integrity checks
- Step types: capability, delay, condition
- Execution states: running, completed, failed, cancelled

### 13. Content Addressable Storage
**Decision**: Use content-addressable storage with SHA-256 based CIDs.

**Rationale**:
- Content deduplication
- Integrity verification through CID matching
- Enables distributed content sharing via DHT

**Implementation**:
- `pkg/content/store.go`: Block store with CID verification
- `pkg/content/provider.go`: DHT-based provider registration
- CID calculation: `hex(sha256(data))`
- BadgerDB and in-memory implementations

### 14. Decentralized Mailbox System
**Decision**: Implement a decentralized mailbox for secure message delivery.

**Rationale**:
- Enables asynchronous communication
- Message encryption ensures privacy
- No centralized message broker required

**Implementation**:
- `pkg/mailbox`: Encrypted mailbox with AES-GCM
- Message storage in content-addressable store
- Fetch mechanism for recipient retrieval

### 15. Multi-Lane Communication Bridge
**Decision**: Implement a multiplexed bridge with separate lanes for different traffic types.

**Rationale**:
- Prioritizes critical traffic (emergency lane)
- Separates different communication types (chat, workflow, file)
- Prevents traffic interference

**Implementation**:
- `pkg/agent_bridge/multiplexed_bridge.go`: Multi-lane communication
- Lane types: emergency, chat, workflow, file_upload, file_download
- Per-lane queues with size monitoring

### 16. Human Client Integration
**Decision**: Support human clients in sessions with online preference management.

**Rationale**:
- Enables human-in-the-loop workflows
- Respects user availability preferences
- Provides notification mechanisms

**Implementation**:
- `pkg/agent/registry.go`: Human client registration
- `pkg/orchestrator/connector_human_client_test.go`: Human client management
- Online preference: always_online, online_when_active, offline

### 17. External Platform Integration
**Decision**: Provide integration with external platforms (GitHub, Gmail, OpenAI, etc.).

**Rationale**:
- Extends agent capabilities to external services
- Enables automation across platforms
- Webhook support for event-driven workflows

**Implementation**:
- `pkg/orchestrator/external_platforms.go`: External platform manager
- `pkg/integration/webhook_router.go`: Webhook processing with HMAC verification
- Supported platforms: GitHub, Gmail, OpenAI, Midjourney, DALL-E, Slack, Discord, Google Drive, Dropbox, AWS S3

### 18. Collective Memory System
**Decision**: Implement shared memory across agents for knowledge retention.

**Rationale**:
- Enables agents to learn from each other
- Reduces redundant computations
- Provides persistent knowledge storage

**Implementation**:
- `pkg/agent/memory/collective_memory.go`: Shared memory system
- Knowledge storage and retrieval
- Cross-agent knowledge sharing

### 19. Quality Assurance System
**Decision**: Implement a quality checker for agent outputs.

**Rationale**:
- Ensures agent output quality
- Prevents propagation of errors
- Provides feedback for improvement

**Implementation**:
- `pkg/agent/quality/quality_checker.go`: Quality validation
- Output verification rules
- Error detection and reporting

### 20. Learning Engine
**Decision**: Implement a learning engine for agent improvement.

**Rationale**:
- Enables agents to improve over time
- Adapts to user preferences
- Provides continuous optimization

**Implementation**:
- `pkg/agent/learning/learning_engine.go`: Learning capabilities
- Performance tracking
- Adaptive behavior based on feedback

## Security Decisions

### 21. Domain-Separated Signatures
**Decision**: Use domain separation tags for signatures to prevent cross-context replay attacks.

**Rationale**:
- Prevents signature reuse across different contexts
- Adds context to signatures for verification
- Standard cryptographic practice

**Implementation**:
- Tags: `NR-IDENTITY-V1|`, `NR-DELEGATION-V1|`, `NR-REVOCATION-V1|`, `NR-ACP-V1|`
- Applied in `pkg/crypto/signature.go`

### 22. Secret Sharing with Shamir's Scheme
**Decision**: Implement Shamir's Secret Sharing for key recovery.

**Rationale**:
- Enables distributed key management
- No single point of failure
- Configurable threshold for reconstruction

**Implementation**:
- `pkg/crypto/shamir.go`: Shamir's Secret Sharing
- Configurable threshold (n of m shares required)

### 23. Environment Variable for Vault Passphrase
**Decision**: Use `MUSKETEERS_VAULT_PASSPHRASE` environment variable for vault encryption.

**Rationale**:
- Avoids hardcoding secrets
- Standard practice for secret management
- Enables different passphrases per environment

**Implementation**:
- Used in `pkg/vault` and `pkg/providers/api_key_manager`
- Fallback to base64 encoding for backward compatibility

### 24. HMAC-SHA256 for Webhook Verification
**Decision**: Use HMAC-SHA256 for webhook signature verification.

**Rationale**:
- Industry standard for webhook security
- Prevents webhook spoofing
- Shared secret-based verification

**Implementation**:
- `pkg/integration/webhook_router.go`: Webhook signature verification
- Secret key configuration per webhook

## Performance Decisions

### 25. In-Memory Caching
**Decision**: Use in-memory caches for frequently accessed data.

**Rationale**:
- Reduces disk I/O
- Improves response times
- Lowers latency for hot data

**Implementation**:
- `pkg/providers/router.go`: Model cache
- `pkg/identity`: In-memory identity cache with disk persistence
- `pkg/storage`: In-memory block store for testing

### 26. Concurrent Processing
**Decision**: Use goroutines and channels for concurrent operations.

**Rationale**:
- Leverages Go's concurrency model
- Improves throughput
- Enables parallel processing

**Implementation**:
- `pkg/content/retrieval.go`: Parallel fetching from multiple providers
- `pkg/storage/quota_test.go`: Concurrent quota operations
- Extensive use of `sync.RWMutex` for thread safety

### 27. Connection Pooling
**Decision**: Reuse connections for external service communication.

**Rationale**:
- Reduces connection overhead
- Improves performance
- Prevents connection exhaustion

**Implementation**:
- HTTP client reuse in providers
- libp2p connection management
- Session reuse in agent bridge

## Observability Decisions

### 28. Structured Logging
**Decision**: Use structured logging with zap and logrus.

**Rationale**:
- Enables log parsing and analysis
- Provides context in logs
- Supports log aggregation

**Implementation**:
- `go.uber.org/zap` for structured logging
- `github.com/sirupsenlogrus` for general logging
- Component-specific log fields

### 29. Metrics Collection
**Decision**: Collect metrics for monitoring and alerting.

**Rationale**:
- Enables performance monitoring
- Supports capacity planning
- Provides operational insights

**Implementation**:
- `pkg/agent/lifecycle.go`: Metrics collection
- `pkg/providers/router.go`: Usage statistics tracking
- `pkg/orchestrator`: Component-level metrics

### 30. Audit Logging
**Decision**: Implement audit logging for security-sensitive operations.

**Rationale**:
- Provides accountability
- Enables forensic analysis
- Supports compliance requirements

**Implementation**:
- `pkg/runtime`: Audit log interface
- Integration with capability execution
- Session and identity operations logging

## Testing Decisions

### 31. Table-Driven Tests
**Decision**: Use table-driven tests for comprehensive test coverage.

**Rationale**:
- Reduces test code duplication
- Makes test cases explicit
- Easier to add new test cases

**Implementation**:
- Extensive use in all packages
- Test coverage for edge cases
- Concurrent operation testing

### 32. Mock Implementations
**Decision**: Provide mock implementations for testing.

**Rationale**:
- Enables unit testing without external dependencies
- Improves test reliability
- Reduces test execution time

**Implementation**:
- Mock policy engine in tests
- Mock mailbox in webhook tests
- In-memory implementations for storage

### 33. Integration Tests
**Decision**: Include integration tests for component interactions.

**Rationale**:
- Validates component integration
- Catches integration issues early
- Provides end-to-end validation

**Implementation**:
- `pkg/integration/integration_test.go`: Runtime, policy, capability, workflow integration
- `pkg/agent/integration/integration_test.go`: Collective agent system integration
- Multi-component test scenarios

## Deployment Decisions

### 34. Configuration via Environment Variables
**Decision**: Use environment variables for configuration.

**Rationale**:
- Standard practice for containerized deployments
- Enables configuration without code changes
- Supports different environments (dev, staging, prod)

**Implementation**:
- `MUSKETEERS_VAULT_PASSPHRASE` for vault passphrase
- Provider-specific API key environment variables
- Configurable timeouts and limits

### 35. Graceful Shutdown
**Decision**: Implement graceful shutdown for all components.

**Rationale**:
- Prevents data loss
- Ensures clean resource cleanup
- Provides better user experience

**Implementation**:
- Context-based cancellation
- Deferred cleanup in goroutines
- Connection closure on shutdown

## Event System Decisions

### 36. Event-Driven Architecture
**Decision**: Implement an event bus for loose coupling between components.

**Rationale**:
- Enables asynchronous communication
- Reduces coupling between components
- Supports scalable architecture
- Facilitates monitoring and observability

**Implementation**:
- `pkg/eventbus`: Publish-subscribe event system
- `pkg/events`: Event type definitions
- Integration with orchestrator, agent, session, CEO, channel
- Event routing and subscription management

### 37. Session Event Broadcasting
**Decision**: Broadcast session events to all agents to prevent "blindness".

**Rationale**:
- Ensures agents are aware of session state changes
- Prevents agents from missing important events
- Enables coordinated multi-agent workflows

**Implementation**:
- `pkg/orchestrator/session_event_broadcaster.go`: Session event broadcasting
- Events: task assigned, task completed, artifact shared, progress updates, errors
- Concurrency-safe broadcast channels

## P2P Network Decisions

### 38. libp2p for P2P Networking
**Decision**: Use libp2p for peer-to-peer networking.

**Rationale**:
- Industry-standard P2P library
- Provides built-in encryption and authentication
- Supports multiple transport protocols
- Enables DHT-based discovery

**Implementation**:
- `pkg/node`: Core node implementation with libp2p host
- `pkg/discovery`: mDNS and bootstrap peer discovery
- TLS support for secure connections
- DHT integration for content discovery

### 39. Decentralized Domain Name System
**Decision**: Implement a decentralized DNS for .ia domains with commit-reveal.

**Rationale**:
- Enables human-readable domain names
- Prevents front-running with commit-reveal
- Decentralized and censorship-resistant
- Supports domain renewal

**Implementation**:
- `pkg/naming`: Decentralized DNS implementation
- Commit-reveal scheme for domain registration
- Domain resolution via DHT
- Domain renewal mechanism

### 40. Peer Discovery Mechanisms
**Decision**: Implement multiple peer discovery mechanisms.

**Rationale**:
- Ensures network bootstrapping
- Provides redundancy in discovery
- Supports different network conditions

**Implementation**:
- `pkg/discovery`: mDNS for local network discovery
- Bootstrap peer configuration for initial connection
- DHT-based peer discovery

## Communication Decisions

### 41. Channel-Based Communication
**Decision**: Implement channels for different communication types.

**Rationale**:
- Separates different communication contexts
- Enables encryption for private channels
- Supports session-based communication

**Implementation**:
- `pkg/channel`: Channel communication system
- Public channels for open communication
- Private channels with Ed25519 encryption
- Session channels for workflow communication

### 42. End-to-End Encryption for Private Channels
**Decision**: Use Ed25519 for private channel encryption.

**Rationale**:
- Ensures message privacy
- Prevents eavesdropping
- Provides authentication

**Implementation**:
- Ed25519 key pairs for channel encryption
- Message encryption before sending
- Decryption only by authorized participants

## Orchestration Decisions

### 43. Multi-Stage Verification
**Decision**: Implement multi-stage verification for agent results.

**Rationale**:
- Ensures result quality
- Catches errors at multiple stages
- Provides confidence scoring

**Implementation**:
- `pkg/verification`: Multi-stage verification system
- Stages: syntax, semantics, security, performance, integration
- Integration with orchestrator aggregator and final reviewer

### 44. Result Aggregation Strategies
**Decision**: Implement multiple result aggregation strategies.

**Rationale**:
- Supports different use cases
- Enables flexible result combination
- Provides confidence scoring

**Implementation**:
- `pkg/orchestrator/aggregator.go`: Result aggregation
- Strategies: first valid, majority, weighted, consensus
- Integration with multi-stage verifier

### 45. Role-Based Task Execution
**Decision**: Implement role-based task assignment and execution.

**Rationale**:
- Enables specialized agent roles
- Supports complex workflows
- Provides clear responsibility separation

**Implementation**:
- `pkg/orchestrator/role_assigner.go`: Role assignment
- Roles: manager, assistant, observer, specialist
- Capability validation per role

### 46. Task Delegation System
**Decision**: Implement task delegation between agents.

**Rationale**:
- Enables agent collaboration
- Supports permission-based delegation
- Provides delegation lifecycle management

**Implementation**:
- `pkg/delegation`: Task delegation management
- Delegation creation, revocation, completion
- Permission and constraint management

### 47. Failure Handling Strategies
**Decision**: Implement multiple failure handling strategies.

**Rationale**:
- Enables flexible error recovery
- Supports different failure scenarios
- Provides configurable retry logic

**Implementation**:
- `pkg/orchestrator/failure_handler.go`: Failure handling
- Strategies: retry, reassign, escalate, fallback, skip, manual review
- Configurable retry limits and escalation rules

### 48. CEO Supervisor for Health Monitoring
**Decision**: Implement a CEO supervisor for network health monitoring.

**Rationale**:
- Provides network-wide health monitoring
- Enables proactive issue detection
- Supports automated health checks

**Implementation**:
- `pkg/ceo`: CEO supervisor implementation
- Health check registration as admin agent
- Periodic health monitoring

## Integration Decisions

### 49. Comprehensive Connector
**Decision**: Implement a central connector integrating all system components.

**Rationale**:
- Provides single integration point
- Simplifies component communication
- Enables centralized management

**Implementation**:
- `pkg/orchestrator/connector.go`: Central connector
- Integration with event bus, agent bridge, agent registry
- Integration with MCP, A2A, email, session, storage

### 50. MCP Protocol Integration
**Decision**: Integrate Model Context Protocol (MCP) for tool management.

**Rationale**:
- Standard protocol for tool invocation
- Enables resource reading
- Supports prompt retrieval

**Implementation**:
- `pkg/orchestrator/mcp_protocol.go`: MCP manager
- Server registration (GitHub, Postgres, Slack)
- Tool invocation, resource reading, prompt retrieval

### 51. A2A Protocol Integration
**Decision**: Integrate Agent-to-Agent (A2A) protocol for agent communication.

**Rationale**:
- Standardized agent communication
- Supports session management
- Enables artifact sharing

**Implementation**:
- `pkg/orchestrator/a2a_protocol.go`: A2A manager
- Agent registration, session creation, message sending
- Task assignment and completion

## Storage and Content Decisions

### 52. Storage Connector Integration
**Decision**: Integrate storage system with orchestrator to prevent file isolation.

**Rationale**:
- Prevents file isolation in components
- Enables centralized file management
- Supports quota management

**Implementation**:
- `pkg/orchestrator/storage_connector.go`: Storage connector
- File storing, retrieving, deleting, listing
- Quota management integration

### 53. HTTP Gateway for Content Access
**Decision**: Implement HTTP gateway for content access over P2P network.

**Rationale**:
- Enables HTTP access to P2P content
- Supports web integration
- Provides content serving

**Implementation**:
- `pkg/gateway`: HTTP gateway implementation
- Content serving from P2P network
- TLS support for secure access

## Utility Package Decisions

### 54. SDK for External Integration
**Decision**: Provide SDK for external developer integration.

**Rationale**:
- Enables external developers to integrate
- Provides client libraries
- Supports external use cases

**Implementation**:
- `pkg/sdk`: SDK implementation
- Client SDK and API bindings
- Integration with agent and session systems

### 55. Isolated Utility Packages
**Decision**: Implement isolated utility packages for future extensibility.

**Rationale**:
- Provides foundation for future features
- Enables modular development
- Supports optional functionality

**Implementation**:
- `pkg/analytics`: Analytics and metrics
- `pkg/backup`: Backup utilities
- `pkg/ledger`: Transaction ledger
- `pkg/notifications`: Notification system
- `pkg/plugins`: Plugin system
- `pkg/sandbox`: Sandbox execution
- `pkg/search`: Search functionality
- `pkg/upgrade`: Upgrade management

## Summary

The Musketeers project follows a layered, decentralized architecture with strong emphasis on:
- **Security**: Ed25519 cryptography, AES-256-GCM encryption, policy-based access control, domain separation
- **Reliability**: Erasure coding, checkpointing, health monitoring, graceful shutdown, failure handling
- **Scalability**: Concurrent processing, caching, connection pooling, event-driven architecture
- **Observability**: Structured logging, metrics collection, audit logging, comprehensive monitoring
- **Maintainability**: Clean architecture, comprehensive testing, clear separation of concerns, interface-first design
- **Extensibility**: Plugin system, SDK, isolated utility packages, protocol integration
- **Decentralization**: P2P networking, DHT, decentralized DNS, identity system

These decisions enable the system to operate as a decentralized, secure, and scalable multi-agent platform with AI provider integration, robust storage capabilities, comprehensive orchestration, and extensive integration options.

**Total Architectural Decisions**: 55 decisions documented across all system components.
