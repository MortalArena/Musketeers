# Musketeers Project - Component Context Understanding

## Project Overview

Musketeers is a distributed, decentralized multi-agent orchestration system built on libp2p peer-to-peer networking. The project enables agents to communicate, collaborate, and execute tasks in a secure, distributed environment with built-in identity verification, storage, and orchestration capabilities.

## Core Architecture

### 1. Peer-to-Peer Foundation (libp2p)
- **pkg/node/**: Core node implementation managing libp2p host, DHT, identity, and storage
- **pkg/network/**: Network utilities and connection management
- **pkg/discovery/**: Peer discovery mechanisms (mDNS, bootstrap peers)
- **pkg/naming/**: Decentralized domain name system for .ia domains

### 2. Identity & Security
- **pkg/crypto/**: Cryptographic primitives (Ed25519 signatures, domain separation, key derivation)
- **pkg/identity/**: Identity records with proof-of-work for Sybil resistance
- **pkg/security/**: Security policies and access control
- **pkg/vault/**: Encrypted secret storage with scrypt key derivation

### 3. Agent System
- **pkg/agent/**: Core agent abstractions (48 files)
  - Agent registry, lifecycle management
  - Task execution, result aggregation
  - Multi-stage verification
  - Various agent types (API, CLI, IDE, Local, Browser, Custom)
- **pkg/agent_bridge/**: Bridge connecting agents to the network (15 files)
  - MultiplexedBridge for concurrent agent communication
  - Session management
  - Message routing
- **pkg/capability/**: Capability-based access control
- **pkg/skills/**: Agent skill definitions and management

### 4. Orchestration Layer
- **pkg/orchestrator/**: Central orchestration engine (29 files)
  - A2A (Agent-to-Agent) protocol manager
  - Session management and delegation
  - Role assignment (Leader, Reviewer, Executor, etc.)
  - Result aggregation strategies (first valid, majority, weighted, consensus)
  - Failure handling (retry, reassign, escalate, fallback)
  - Final review with multi-stage verification
  - MCP (Model Context Protocol) manager
  - Email system integration
  - External platform manager (GitHub, Gmail, OpenAI, etc.)
  - Chat connector for channel communication
  - Session event broadcaster
  - Comprehensive logger
  - Storage connector
  - Connector hub integrating all components

### 5. Communication Protocols
- **pkg/protocol/**: Protocol definitions and message structures
  - Bitswap, Direct protocols
  - Channel messages (public, private, encrypted)
  - Direct 1:1 messages with Curve25519 encryption
  - Site manifests and provider records
- **pkg/acp/**: Agent Communication Protocol (6 files)
  - Task request/response/error messages
  - Router for task handlers
  - Built-in tasks: echo, ping, translate, execute
  - Transport over libp2p streams

### 6. Event System
- **pkg/eventbus/**: Event bus for inter-component communication
- **pkg/events/**: Event definitions and types

### 7. Session & Workflow Management
- **pkg/session/**: Session containers and management (16 files)
- **pkg/workflow/**: Workflow execution and checkpoints (7 files)

### 8. Storage & Content
- **pkg/storage/**: Storage abstraction with quota management
- **pkg/content/**: Content addressing and storage (3 files)
- **pkg/mailbox/**: Message mailbox system
- **pkg/ledger/**: Transaction ledger

### 9. External Integrations
- **pkg/providers/**: LLM provider abstractions (32 files)
  - OpenAI, Anthropic, Google, local models
  - Unified provider interface
- **pkg/integration/**: External platform integrations (9 files)
  - GitHub, Gmail, OpenAI, Midjourney, Slack, Discord, etc.

### 10. Runtime & Execution
- **pkg/runtime/**: Runtime environment management (21 files)
- **pkg/sandbox/**: Sandbox execution environment
- **pkg/verification/**: Multi-stage verification system

### 11. Supporting Services
- **pkg/gateway/**: HTTP gateway for content access
- **pkg/registry/**: Agent manifest registry
- **pkg/notifications/**: Notification system
- **pkg/backup/**: Backup utilities
- **pkg/upgrade/**: Upgrade management
- **pkg/search/**: Search functionality
- **pkg/ceo/**: CEO supervisor for network health monitoring
- **pkg/delegation/**: Task delegation management
- **pkg/policy/**: Policy engine for access control

### 12. Common Utilities
- **pkg/common/**: Common utilities (key resolver, etc.)
- **pkg/memory/**: Memory management (6 files)

## Entry Points (cmd/)

### cmd/agent/main.go
Agent client that connects to the Studio Bridge and executes tasks using the unified agent system.

### cmd/founder/main.go
Founder tool for managing .ia domain registration with commit-reveal scheme for security.

### cmd/gateway/main.go
HTTP gateway server for content access over the P2P network.

### cmd/main.go
Simple provider registry demo for testing LLM providers.

### cmd/seed/main.go
Bootstrap seed node for network initialization.

### cmd/studio/main.go
Main orchestration studio that integrates all components:
- Node with identity
- Agent registry with multiple agent adapters
- Session management
- Tool executor with security bounds
- CEO supervisor for health monitoring
- Orchestrator components (session, delegation, verification)
- ACP handlers
- Agent bridge server
- Chat connector
- External platform manager
- HTTP server

## Key Design Patterns

1. **Event-Driven Architecture**: Extensive use of event bus for loose coupling
2. **Capability-Based Security**: Fine-grained access control via capabilities
3. **Multi-Stage Verification**: Syntax, semantics, security, performance, integration checks
4. **Result Aggregation**: Multiple strategies for combining agent results
5. **Role-Based Orchestration**: Agents assigned specific roles (Leader, Reviewer, etc.)
6. **Commit-Reveal**: Secure domain registration to prevent front-running
7. **Proof-of-Work**: Sybil resistance for identity registration
8. **Domain Separation**: Cryptographic domain separation for different contexts
9. **Quota Management**: Storage and resource limits per DID
10. **Adapter Pattern**: Multiple agent adapters (API, CLI, IDE, Local, Browser)

## Security Features

1. **Ed25519 Signatures**: All messages signed for authenticity
2. **Curve25519 Encryption**: Private channel encryption
3. **Scrypt Key Derivation**: Secure key derivation for vault
4. **AES-256-GCM**: Symmetric encryption for secrets
5. **Domain Separation**: Prevents cross-context key reuse
6. **Identity Verification**: Proof-of-work based identity records
7. **Capability Checks**: Fine-grained access control
8. **Tool Execution Bounds**: Limits on file operations and paths
9. **Policy Engine**: Configurable access rules
10. **HMAC-SHA256**: Webhook signature verification

## Arabic Comments

A significant portion of the codebase contains Arabic comments, indicating a multilingual development environment or target audience. This is particularly prevalent in:
- pkg/orchestrator/ (orchestration components)
- pkg/agent/ (agent system)
- pkg/session/ (session management)
- cmd/ (command-line tools)

## Dependencies Between Packages

### Core Foundation
- node ← crypto, identity, discovery, network, naming, storage
- All packages depend on crypto for signing/verification

### Agent System
- agent_bridge ← agent, eventbus
- orchestrator ← agent, agent_bridge, eventbus, session, capability, policy, verification, storage, mailbox, acp
- agent ← capability, skills, verification

### Communication
- acp ← crypto, protocol, common
- protocol ← (minimal dependencies)
- channel ← crypto, eventbus

### Storage
- storage ← (minimal dependencies)
- content ← storage
- mailbox ← storage

### External
- providers ← (minimal dependencies)
- integration ← providers, events

### Orchestration
- orchestrator ← (most other packages as integration hub)

## Current State

The project is a comprehensive multi-agent orchestration system with:
- 365 Go files across 43 packages
- Full P2P networking with libp2p
- Multiple agent types and adapters
- Rich orchestration capabilities
- Strong security model
- Extensive external integrations
- Arabic language support in comments

The codebase appears to be in active development with well-structured packages and comprehensive test coverage.
