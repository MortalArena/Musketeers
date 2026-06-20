# Dependency Map - Musketeers Project

## Overview
This document maps the dependencies between all packages in the Musketeers project based on the comprehensive codebase audit of 365 Go files across 43 packages.

## Package Dependency Graph

### Core Infrastructure Packages

#### pkg/common (2 files)
- **Purpose**: Shared interfaces and utilities
- **Dependencies**: None (pure interfaces)
- **Dependents**: pkg/acp, pkg/identity, pkg/crypto, pkg/vault
- **Key Interfaces**:
  - `KeyResolver`: Resolves public Ed25519 keys from DIDs
  - `DIDProvider`: Exposes decentralized identifiers
  - `Signer`: Signs raw bytes
  - `Verifier`: Verifies signatures
  - `Encryptor`: Encrypts plaintext
  - `Decryptor`: Decrypts ciphertext

#### pkg/protocol (1 file)
- **Purpose**: Protocol definitions and constants
- **Dependencies**: None (constants and types)
- **Dependents**: pkg/content, pkg/acp, pkg/storage
- **Key Constants**:
  - `ProtocolBitswap`, `ProtocolDirect`, `ProtocolVersion`
  - `MaxMessageSize`, `MaxChunkSize`, `MaxBlockSize`
  - Message structures: ChannelMessage, EncryptedMessage, DirectMessage, SiteManifest

### Cryptography & Security Packages

#### pkg/crypto (13 files)
- **Purpose**: Cryptographic operations (Ed25519, PoW, encryption)
- **Dependencies**: pkg/common
- **Dependents**: pkg/identity, pkg/acp, pkg/vault
- **Key Functions**:
  - Ed25519 key generation and signing
  - Proof of Work mining and verification
  - Domain-separated signatures
  - Mnemonic phrase generation (BIP39)

#### pkg/identity (10 files)
- **Purpose**: Identity lifecycle management (DIDs, keys, delegation)
- **Dependencies**: pkg/crypto, pkg/common
- **Dependents**: pkg/agent, pkg/session, pkg/orchestrator
- **Key Components**:
  - `IdentityManager`: Creates, updates, activates, deactivates identities
  - `IdentityStore`: Persistent storage with JSON files
  - `DelegationRecord`: Permission delegation
  - `RevocationRecord`: Identity revocation
  - `IdentityLimiter`: Rate limiting for identity creation

#### pkg/vault (8 files)
- **Purpose**: Secure secret storage and management
- **Dependencies**: pkg/crypto
- **Dependents**: pkg/integration
- **Key Components**:
  - `Vault`: Stores, retrieves, deletes secrets with encryption
  - `FileKeyProvider`: Key storage on disk
  - `Encryption`: AES-GCM encryption/decryption
  - Environment variable: `MUSKETEERS_VAULT_PASSPHRASE`

#### pkg/policy (5 files)
- **Purpose**: Access control and policy enforcement
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/capability, pkg/agent_bridge, pkg/integration, pkg/workflow
- **Key Components**:
  - `Engine`: Evaluates requests against rules
  - `ApprovalEngine`: Multi-level approval system
  - `Rule`: Principal, resource, condition, effect (allow/deny)

#### pkg/security (5 files)
- **Purpose**: Security policies and access control
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/agent
- **Key Components**:
  - Security context management
  - Access control policies
  - Security verification

### Agent System Packages

#### pkg/agent (48 files)
- **Purpose**: Agent registry, lifecycle, and management
- **Dependencies**: pkg/identity, pkg/crypto, pkg/common, pkg/security
- **Dependents**: pkg/orchestrator, pkg/agent_bridge, pkg/integration
- **Key Components**:
  - `AgentRegistry`: Manages agent manifests
  - `AgentLifecycleManager`: Agent lifecycle (start/stop/health)
  - `InstanceManager`: Manages agent instances
  - `SkillManager`: Manages agent skills
  - `SubagentManager`: Manages subagents
  - `LearningEngine`: Agent learning capabilities
  - `CollectiveMemory`: Shared memory across agents
  - `QualityChecker`: Quality assurance for agents
  - Agent adapters: API, CLI, IDE, Local, Browser, Custom

#### pkg/agent_bridge (15 files)
- **Purpose**: Communication bridge between Studio and Agents
- **Dependencies**: pkg/policy, pkg/agent
- **Dependents**: pkg/orchestrator
- **Key Components**:
  - `Server`: TCP server for agent connections
  - `Client`: Client for connecting to bridge server
  - `SessionManager`: Manages agent sessions
  - `MultiplexedBridge`: Multi-lane communication (emergency, chat, workflow, file)
  - `TaskProtocol`: Task request/response protocol
  - `Middleware`: Tool request validation with policy engine

#### pkg/session (16 files)
- **Purpose**: Session management for multi-agent workflows
- **Dependencies**: pkg/agent, pkg/identity
- **Dependents**: pkg/orchestrator, pkg/integration
- **Key Components**:
  - `UnifiedSessionManager`: Centralized session management
  - `SessionInfo`: Session metadata and state
  - Session lifecycle: create, pause, resume, complete

#### pkg/orchestrator (29 files)
- **Purpose**: High-level orchestration of agents and workflows
- **Dependencies**: pkg/agent, pkg/agent_bridge, pkg/session, pkg/storage, pkg/providers, pkg/eventbus, pkg/capability, pkg/policy, pkg/verification, pkg/mailbox, pkg/acp
- **Dependents**: None (top-level orchestration)
- **Key Components**:
  - `OrchestratorEngine`: Coordinates system components
  - `SessionManager`: Session creation and delegation
  - `RoleAssigner`: Assigns roles to agents
  - `SessionEventBroadcaster`: Broadcasts session events
  - `StorageConnector`: Links storage to orchestrator
  - `Connector`: Manages external platform connections
  - `EmailManager`: Email system integration
  - `ExternalPlatformManager`: External platform management
  - `MCPManager`: Multi-Agent Communication Protocol
  - `ChatConnector`: Channel communication
  - `ComprehensiveLogger`: System-wide logging
  - `DelegationManager`: Task delegation
  - `FailureHandler`: Failure handling strategies
  - `Aggregator`: Result aggregation
  - `FinalReviewer`: Result verification

### Workflow & Capability Packages

#### pkg/capability (12 files)
- **Purpose**: Capability registration and execution
- **Dependencies**: pkg/policy
- **Dependents**: pkg/integration, pkg/workflow
- **Key Components**:
  - `Manager`: Registers and executes capabilities
  - `Capability`: Interface for capabilities
  - `Command`: Capability command interface
  - `Result`: Capability execution result

#### pkg/registry (3 files)
- **Purpose**: Agent manifest registry
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/integration, pkg/runtime
- **Key Components**:
  - `MemoryRegistry`: In-memory agent manifest storage
  - `DHTRegistry`: DHT-based registry
  - `AgentManifest`: Agent metadata (ID, DID, capabilities, tasks, endpoints)

#### pkg/runtime (21 files)
- **Purpose**: Agent runtime environment
- **Dependencies**: pkg/registry, pkg/capability
- **Dependents**: pkg/integration
- **Key Components**:
  - `AgentRuntime`: Runtime interface (event bus, state store, knowledge store, scheduler)
  - `AgentContext`: Agent execution context
  - `AgentMetadata`: Agent metadata

#### pkg/workflow (7 files)
- **Purpose**: Workflow definition and execution
- **Dependencies**: pkg/capability, pkg/policy
- **Dependents**: pkg/integration
- **Key Components**:
  - `DefaultWorkflowEngine`: Registers and executes workflows
  - `Workflow`: Workflow definition (steps, conditions, loops)
  - `CheckpointManager`: Saves/restores workflow state
  - `Checkpoint`: Workflow state checkpoint with integrity hash

#### pkg/skills (5 files)
- **Purpose**: Agent skill definitions and management
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/agent
- **Key Components**:
  - Skill definitions
  - Skill execution
  - Skill validation

### Integration Packages

#### pkg/integration (9 files)
- **Purpose**: Integration between system components
- **Dependencies**: pkg/agent, pkg/session, pkg/capability, pkg/workflow, pkg/registry, pkg/vault, pkg/storage, pkg/mailbox
- **Dependents**: None (integration layer)
- **Key Components**:
  - `AgentCommunication`: Inter-agent messaging
  - `AgentSessionIntegration`: Links agent registry with session manager
  - `InstanceSessionIntegration`: Links instance manager with session manager
  - `RoleAssignment`: Assigns roles to agents
  - `SessionOrchestrator`: Coordinates session components
  - `TaskRouting`: Routes tasks to appropriate agents
  - `WebhookRouter`: Processes external webhooks with HMAC verification

### Storage & Content Packages

#### pkg/storage (4 files)
- **Purpose**: Storage management with erasure coding and quotas
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/content, pkg/mailbox, pkg/orchestrator
- **Key Components**:
  - `ErasureCoder`: Reed-Solomon erasure coding (10 data + 4 parity shards)
  - `QuotaManager`: Storage quota management (1GB default free tier)
  - Constants: `DataShards=10`, `ParityShards=4`, `DefaultFreeTierBytes=1GB`

#### pkg/content (3 files)
- **Purpose**: Content storage and retrieval with DHT
- **Dependencies**: pkg/protocol, pkg/storage
- **Dependents**: pkg/mailbox
- **Key Components**:
  - `ProviderManager`: Registers providers in DHT
  - `Fetcher`: Fetches content from network
  - `BlockStore`: Interface for block storage
  - `BadgerBlockStore`: BadgerDB implementation
  - `MemoryBlockStore`: In-memory implementation
  - `CID`: SHA-256 based content identifier

#### pkg/mailbox (1 file)
- **Purpose**: Decentralized mailbox system
- **Dependencies**: pkg/content
- **Dependents**: pkg/integration
- **Key Components**:
  - `Mailbox`: Encrypted message storage
  - `Message`: Encrypted message with AES-GCM
  - `Send`: Encrypts and stores messages
  - `Fetch`: Retrieves and decrypts messages

### Provider Packages

#### pkg/providers (32 files)
- **Purpose**: AI provider management and routing
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/orchestrator
- **Key Components**:
  - `ProviderRegistry`: Manages all providers
  - `Router`: Smart routing with usage tracking
  - `FreeRouter`: Routes to free models only
  - `ModelCatalog`: Model information catalog
  - `FreeModelsTracker`: Tracks free model usage
  - `APIKeyManager`: Secure API key management with AES-256-GCM
  - Supports 22 official providers + Ollama + custom

### Communication Protocol Packages

#### pkg/acp (6 files)
- **Purpose**: Agent Communication Protocol (ACP)
- **Dependencies**: pkg/crypto, pkg/common, pkg/protocol
- **Dependents**: None (protocol implementation)
- **Key Components**:
  - `Router`: Routes ACP tasks to handlers
  - `Envelope`: ACP message with signature
  - `Transport`: libp2p transport for ACP
  - Built-in tasks: ping, echo, translate, execute
  - Protocol version: `acp/v1`, Protocol ID: `/nr/acp/1.0.0`

### Event System Packages

#### pkg/eventbus (2 files)
- **Purpose**: Event bus for inter-component communication
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/orchestrator, pkg/agent, pkg/session
- **Key Components**:
  - `EventBus`: Publish-subscribe event system
  - Event routing and subscription management

#### pkg/events (5 files)
- **Purpose**: Event definitions and types
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/agent, pkg/orchestrator
- **Key Components**:
  - Event type definitions
  - Event serialization/deserialization

### P2P Network Packages

#### pkg/node (16 files)
- **Purpose**: Core node implementation managing libp2p host, DHT, identity, and storage
- **Dependencies**: pkg/crypto, pkg/identity, pkg/discovery, pkg/naming, pkg/storage
- **Dependents**: cmd/seed, cmd/gateway, cmd/founder, cmd/studio
- **Key Components**:
  - `Node`: libp2p host management
  - DHT integration
  - Identity management
  - Storage integration

#### pkg/discovery (2 files)
- **Purpose**: Peer discovery mechanisms (mDNS, bootstrap peers)
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/node
- **Key Components**:
  - mDNS discovery
  - Bootstrap peer management

#### pkg/network (2 files)
- **Purpose**: Network utilities and connection management
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/node
- **Key Components**:
  - Connection management
  - Network utilities

#### pkg/naming (5 files)
- **Purpose**: Decentralized domain name system for .ia domains
- **Dependencies**: pkg/crypto
- **Dependents**: pkg/node, cmd/founder
- **Key Components**:
  - Domain registration with commit-reveal
  - Domain resolution
  - Domain renewal

### Additional Packages

#### pkg/analytics (1 file)
- **Purpose**: Analytics and metrics collection
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Metrics collection
  - Analytics reporting

#### pkg/backup (1 file)
- **Purpose**: Backup utilities
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Backup creation
  - Backup restoration

#### pkg/ceo (1 file)
- **Purpose**: CEO supervisor for network health monitoring
- **Dependencies**: pkg/eventbus, pkg/agent
- **Dependents**: cmd/studio
- **Key Components**:
  - Health monitoring
  - Network status reporting

#### pkg/channel (7 files)
- **Purpose**: Channel communication management
- **Dependencies**: pkg/eventbus, pkg/crypto
- **Dependents**: pkg/orchestrator
- **Key Components**:
  - Public channels
  - Private channels (encrypted)
  - Session channels

#### pkg/delegation (2 files)
- **Purpose**: Task delegation management
- **Dependencies**: pkg/agent, pkg/session
- **Dependents**: pkg/orchestrator
- **Key Components**:
  - Delegation creation
  - Delegation revocation
  - Delegation completion

#### pkg/gateway (3 files)
- **Purpose**: HTTP gateway for content access
- **Dependencies**: pkg/content
- **Dependents**: cmd/gateway
- **Key Components**:
  - HTTP server
  - Content serving
  - Gateway routing

#### pkg/ledger (4 files)
- **Purpose**: Transaction ledger
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Transaction recording
  - Ledger verification

#### pkg/memory (6 files)
- **Purpose**: Memory management
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/agent
- **Key Components**:
  - Memory storage
  - Memory retrieval
  - Memory cleanup

#### pkg/notifications (1 file)
- **Purpose**: Notification system
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Notification delivery
  - Notification management

#### pkg/plugins (1 file)
- **Purpose**: Plugin system
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Plugin loading
  - Plugin management

#### pkg/sandbox (2 files)
- **Purpose**: Sandbox execution environment
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Isolated execution
  - Resource limits

#### pkg/sdk (6 files)
- **Purpose**: SDK for external integration
- **Dependencies**: pkg/agent, pkg/session
- **Dependents**: None
- **Key Components**:
  - Client SDK
  - API bindings

#### pkg/search (1 file)
- **Purpose**: Search functionality
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Search indexing
  - Query execution

#### pkg/upgrade (1 file)
- **Purpose**: Upgrade management
- **Dependencies**: None (self-contained)
- **Dependents**: None
- **Key Components**:
  - Version checking
  - Upgrade execution

#### pkg/verification (1 file)
- **Purpose**: Multi-stage verification system
- **Dependencies**: None (self-contained)
- **Dependents**: pkg/orchestrator
- **Key Components**:
  - Syntax verification
  - Semantics verification
  - Security verification
  - Performance verification
  - Integration verification

### Empty Packages

#### pkg/telemetry (0 files)
- **Purpose**: Telemetry collection (not implemented)
- **Dependencies**: None
- **Dependents**: None
- **Status**: Doesn't exist

#### pkg/email (0 files)
- **Purpose**: Email system (empty)
- **Dependencies**: None
- **Dependents**: None
- **Status**: Empty directory

#### pkg/hosting (0 files)
- **Purpose**: Hosting features (empty)
- **Dependencies**: None
- **Dependents**: None
- **Status**: Empty directory

## Dependency Levels

### Level 0 (No Dependencies)
- pkg/common (2 files)
- pkg/protocol (1 file)
- pkg/policy (5 files)
- pkg/registry (3 files)
- pkg/storage (4 files)
- pkg/providers (32 files)
- pkg/eventbus (2 files)
- pkg/events (5 files)
- pkg/discovery (2 files)
- pkg/network (2 files)
- pkg/analytics (1 file)
- pkg/backup (1 file)
- pkg/ledger (4 files)
- pkg/memory (6 files)
- pkg/notifications (1 file)
- pkg/plugins (1 file)
- pkg/sandbox (2 files)
- pkg/search (1 file)
- pkg/upgrade (1 file)
- pkg/verification (1 file)
- pkg/security (5 files)
- pkg/skills (5 files)

### Level 1 (Depends on Level 0)
- pkg/crypto (13 files) - depends on pkg/common
- pkg/identity (10 files) - depends on pkg/crypto, pkg/common
- pkg/vault (8 files) - depends on pkg/crypto
- pkg/capability (12 files) - depends on pkg/policy
- pkg/runtime (21 files) - depends on pkg/registry, pkg/capability
- pkg/content (3 files) - depends on pkg/protocol, pkg/storage
- pkg/acp (6 files) - depends on pkg/crypto, pkg/common, pkg/protocol
- pkg/naming (5 files) - depends on pkg/crypto
- pkg/gateway (3 files) - depends on pkg/content
- pkg/ceo (1 file) - depends on pkg/eventbus, pkg/agent
- pkg/channel (7 files) - depends on pkg/eventbus, pkg/crypto

### Level 2 (Depends on Level 1)
- pkg/agent (48 files) - depends on pkg/identity, pkg/crypto, pkg/common, pkg/security
- pkg/workflow (7 files) - depends on pkg/capability, pkg/policy
- pkg/mailbox (1 file) - depends on pkg/content
- pkg/node (16 files) - depends on pkg/crypto, pkg/identity, pkg/discovery, pkg/naming, pkg/storage
- pkg/delegation (2 files) - depends on pkg/agent, pkg/session
- pkg/sdk (6 files) - depends on pkg/agent, pkg/session

### Level 3 (Depends on Level 2)
- pkg/agent_bridge (15 files) - depends on pkg/policy, pkg/agent
- pkg/session (16 files) - depends on pkg/agent, pkg/identity
- pkg/integration (9 files) - depends on pkg/agent, pkg/session, pkg/capability, pkg/workflow, pkg/registry, pkg/vault, pkg/storage, pkg/mailbox

### Level 4 (Depends on Level 3)
- pkg/orchestrator (29 files) - depends on pkg/agent, pkg/agent_bridge, pkg/session, pkg/storage, pkg/providers, pkg/eventbus, pkg/capability, pkg/policy, pkg/verification, pkg/mailbox, pkg/acp

## Circular Dependencies
**None detected** - The dependency graph is acyclic (DAG).

## Key Cross-Package Dependencies

### Identity System
- pkg/crypto → pkg/identity → pkg/agent → pkg/orchestrator
- pkg/crypto → pkg/identity → pkg/session → pkg/integration

### Security System
- pkg/crypto → pkg/vault → pkg/integration
- pkg/policy → pkg/capability → pkg/workflow → pkg/integration
- pkg/policy → pkg/agent_bridge → pkg/orchestrator

### Storage System
- pkg/storage → pkg/content → pkg/mailbox → pkg/integration
- pkg/storage → pkg/orchestrator

### Provider System
- pkg/providers → pkg/orchestrator

### Communication System
- pkg/crypto → pkg/acp (independent protocol)
- pkg/common → pkg/acp
- pkg/protocol → pkg/acp

## External Dependencies

### Go Standard Library
- crypto (ed25519, aes, cipher, rand, sha256)
- encoding (json, hex, base64)
- fmt, sync, time, context
- os, path/filepath
- net/http, net/url
- io

### Third-Party Libraries
- github.com/libp2p/go-libp2p (P2P networking)
- github.com/libp2p/go-libp2p-kad-dht (DHT)
- github.com/dgraph-io/badger/v4 (Key-value store)
- github.com/klauspost/reedsolomon (Erasure coding)
- github.com/sirupsen/logrus (Logging)
- go.uber.org/zap (Structured logging)
- golang.org/x/crypto/scrypt (Key derivation)
- github.com/MortalArena/Musketeers/pkg/* (Internal packages)

## Summary

- **Total Packages**: 43 (40 active, 3 empty)
- **Total Files**: 365 Go files
- **Dependency Levels**: 5 levels
- **Circular Dependencies**: 0
- **Architecture**: Clean, layered architecture with clear separation of concerns
- **Core Pillars**: Identity, Security, Agent System, Storage, Communication, Providers, P2P Networking, Event System, Orchestration

### Package Distribution by File Count
- **Large Packages (20+ files)**: pkg/agent (48), pkg/providers (32), pkg/runtime (21), pkg/orchestrator (29)
- **Medium Packages (10-19 files)**: pkg/crypto (13), pkg/identity (10), pkg/node (16), pkg/session (16), pkg/agent_bridge (15), pkg/capability (12)
- **Small Packages (5-9 files)**: pkg/events (5), pkg/memory (6), pkg/sdk (6), pkg/security (5), pkg/skills (5), pkg/naming (5), pkg/channel (7)
- **Tiny Packages (1-4 files)**: All remaining packages (23 packages)
- **Empty Packages**: pkg/telemetry (doesn't exist), pkg/email (empty), pkg/hosting (empty)

### Entry Points (cmd/)
- **cmd/agent/main.go**: Agent client connecting to Studio Bridge
- **cmd/founder/main.go**: Founder tool for .ia domain management
- **cmd/gateway/main.go**: HTTP gateway for content access
- **cmd/main.go**: Provider registry demo
- **cmd/seed/main.go**: Bootstrap seed node
- **cmd/studio/main.go**: Main orchestration studio (comprehensive integration)
