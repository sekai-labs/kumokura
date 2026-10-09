# Kumokura System Architecture

## 1. Architectural Philosophy

Kumokura follows strict **Domain-Driven Design (DDD)** and **Hexagonal Architecture (Ports & Adapters)**. The system is designed to provide high-performance S3 management across three primary presentation targets (CLI, TUI, Desktop GUI) while sharing a single unified, storage-agnostic application core.

```
+-----------------------------------------------------------------------------------------+
|                                  PRESENTATION ADAPTERS                                  |
|         CLI (Cobra)       |       TUI (Bubble Tea)       |    Desktop (Fyne v2 Native)  |
+-----------------------------------------------------------------------------------------+
                                             |
                                             v
+-----------------------------------------------------------------------------------------+
|                                    APPLICATION LAYER                                    |
|   AccountService   |   BucketService   |   ObjectService   |   TransferService |  Sync  |
+-----------------------------------------------------------------------------------------+
                                             |
                                             v
+-----------------------------------------------------------------------------------------+
|                                      DOMAIN LAYER                                       |
|    Accounts        |      Buckets      |      Objects      |      Transfers    |  Sync  |
+-----------------------------------------------------------------------------------------+
                                             |
                                             v
+-----------------------------------------------------------------------------------------+
|                                   DRIVEN ADAPTERS                                       |
|   SQLite DB WAL    |   OS Keyring/AES  |   AWS SDK v2 S3   |  MinIO Compat | Filesystem |
+-----------------------------------------------------------------------------------------+
```

## 2. Bounded Contexts

### Accounts & Credentials
- Domain models `Account`, `AccountType`, and `Credentials`.
- Strict validation invariants: Account names must be alphanumeric and bounded (1-64 characters).
- Driven adapter stores credentials securely in OS Keyring with an AES-256-GCM encrypted fallback file for headless Linux environments.

### Providers & Capabilities
- Provides a capability matrix distinguishing full AWS features from provider subsets (e.g. Cloudflare R2 lacks object locking; MinIO recommends path-style addressing).

### Buckets & Objects
- Pure Go domain models for buckets, objects, prefixes, versioning states, and tags.
- Streaming object uploads and downloads via decoupled `io.Reader` and `io.ReadCloser` boundaries.

### Transfers & Buffers
- High-throughput transfer engine featuring a tiered memory buffer pool (`sync.Pool`) preventing heap fragmentation during multi-gigabyte transfers.
- Checkpoint persistence in SQLite enabling instant pause and resume of interrupted multipart uploads.

### Synchronization
- Bidirectional and one-way diff comparison engine supporting modtime skew tolerance (FAT/S3), checksum comparisons, conflict resolution policies, and non-destructive dry-run planning.
