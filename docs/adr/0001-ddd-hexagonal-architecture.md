# ADR 0001: Domain-Driven Design & Hexagonal Architecture

- **Status**: Accepted
- **Date**: 2026-10-07
- **Authors**: Kumokura Core Engineering Team

---

## 1. Context & Problem Statement

Kumokura is designed to deliver three distinct user-facing surfaces:
1. Command Line Interface (CLI) for shell scripting and automation.
2. Terminal User Interface (TUI) for interactive, keyboard-driven terminal workflows.
3. Desktop Graphical User Interface (GUI) for visual, drag-and-drop desktop management.

Historically, multi-interface tools suffer from severe code duplication and architectural decay. Business logic is often duplicated between CLI handlers and desktop UI controllers, leading to divergent behavior, inconsistent validation, disparate credential management, and independent bug profiles. Furthermore, direct coupling to specific cloud provider SDKs (such as AWS SDK or MinIO SDK) makes testing difficult and complicates future adaptations for non-standard S3 providers.

We require an architectural blueprint that ensures:
- 100% logic sharing across CLI, TUI, and GUI.
- Strict isolation of business rules from UI toolkits, CLI frameworks, and external cloud SDKs.
- Testability without active cloud accounts or live network connections.
- Clean boundaries that accommodate future protocol expansions.

---

## 2. Decision

We adopt **Domain-Driven Design (DDD)** combined with **Hexagonal Architecture (Ports and Adapters)** as the foundational design pattern for Kumokura.

```
                              +-------------------------+
                              |   CLI (Cobra)           |
                              +------------+------------+
                                           |
+-------------------------+                |                +-------------------------+
|   TUI (Bubble Tea v2)   | -------------> | <------------- |   GUI (Fyne v2 Native)  |
+-------------------------+                |                +-------------------------+
                                           v
                              +-------------------------+
                              |  Driving Ports (APIs)   |
                              +-------------------------+
                                           |
                                           v
+-------------------------------------------------------------------------------------+
|                              APPLICATION CORE LAYER                                 |
|                                                                                     |
|   +-----------------------------------------------------------------------------+   |
|   | Use Cases: AccountOps, BucketOps, ObjectOps, TransferService, SyncService   |   |
|   +-----------------------------------------------------------------------------+   |
|                                          |                                          |
|                                          v                                          |
|   +-----------------------------------------------------------------------------+   |
|   | DOMAIN LAYER (Pure Go, Zero External Dependencies)                          |   |
|   | Entities: Account, Bucket, Object, TransferJob, Part, SyncNode              |   |
|   | Value Objects: ByteSize, ETag, StorageClass, S3Key, Checksum                |   |
|   | Domain Events: TransferStarted, PartCompleted, JobFinished                  |   |
|   | Driven Port Interfaces: ObjectStoragePort, LedgerPort, KeyringPort          |   |
|   +-----------------------------------------------------------------------------+   |
+-------------------------------------------------------------------------------------+
                                           |
                                           v
                              +-------------------------+
                              |   Driven Ports (SPIs)   |
                              +-------------------------+
                                           |
       +--------------------+--------------+--------------+--------------------+
       |                    |                             |                    |
       v                    v                             v                    v
+--------------+   +------------------+         +-------------------+   +--------------+
| AWS SDK v2   |   | SQLite WAL       |         | Native Keyring    |   | Local OS     |
| Adapter      |   | Ledger Adapter   |         | Adapter           |   | Filesystem   |
+--------------+   +------------------+         +-------------------+   +--------------+
```

### 2.1 Package Layout
The codebase is structured under `internal/`:
- `internal/core/domain/`: Pure domain models, value objects, domain logic, and driven port interfaces. Standard library only.
- `internal/core/application/`: Application use cases, service orchestrators, transfer coordinators, and event publishers.
- `internal/core/ports/`: Primary (driving) and secondary (driven) port definitions.
- `internal/adapters/`:
  - `storage/s3/`: Secondary adapter implementing `ObjectStoragePort` via AWS SDK for Go v2.
  - `persistence/sqlite/`: Secondary adapter implementing `TransferLedgerPort` and `AccountRepositoryPort` via `modernc.org/sqlite`.
  - `credentials/keyring/`: Secondary adapter implementing `CredentialStorePort` via `zalando/go-keyring`.
  - `fs/`: Secondary adapter implementing `FilesystemPort` for local disk operations.
  - `cli/`: Primary adapter binding Cobra commands to application use cases.
  - `tui/`: Primary adapter binding Bubble Tea v2 / Lip Gloss views to application services.
  - `desktop/`: Primary adapter binding Fyne v2 native UI views to application services.

### 2.2 Bounded Contexts
1. **Identity & Credential Context**:
   - Manages S3 profiles, API keys, secret keys, session tokens, custom endpoints, and secure OS keyring storage.
2. **Object Storage Context**:
   - Manages buckets, prefixes, objects, metadata, tagging, lifecycle policies, and presigned URLs.
3. **Transfer Execution Context**:
   - Manages multipart chunking, dynamic concurrency governance, upload/download progress channels, retry policies, and crash-resilient WAL checkpointing.
4. **Synchronization Context**:
   - Manages local-to-remote and remote-to-local tree diffing, bidirectional sync resolution, deletion propagation, and file hashing.

---

## 3. Consequences

### Positive:
- **Absolute Code Reuse**: All three user interfaces execute the identical underlying business rules, transfer algorithms, and credential logic.
- **High Testability**: Core logic can be unit-tested rapidly using in-memory mock adapters for storage and ledger ports.
- **Provider Swappability**: Replacing or upgrading the underlying storage driver (e.g. adopting AWS SDK v3 in the future) requires modifying only the S3 adapter without altering domain models or UI layers.
- **Architectural Clarity**: Clear boundaries eliminate circular dependencies and ensure high maintainability.

### Negative / Trade-offs:
- **Boilerplate Overhead**: Requires explicit port interfaces, DTO mapping, and adapter layers rather than making direct SDK calls from CLI or UI handlers.
- **Discipline Required**: Developers must strictly avoid importing third-party SDK packages or UI frameworks into domain packages.
