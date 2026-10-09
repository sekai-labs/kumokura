# ADR 0002: Transfer & Sync Engine Architecture

- **Status**: Accepted
- **Date**: 2026-10-07
- **Authors**: Kumokura Core Engineering Team

---

## 1. Context & Problem Statement

Transfer throughput, reliability, and crash resilience represent the core competitive benchmark for Kumokura. Standard cloud clients frequently suffer from three major failure modes:
1. **Memory Bloat**: In uncontrolled worker pools, uploading 64 parts concurrently with naive slice allocations consumes gigabytes of memory, provoking GC pauses and process crashes.
2. **Transfer Brittleness**: A network drop or process termination during a 50GB file upload invalidates the transfer, forcing the user to restart from byte zero.
3. **Throttling Inelasticity**: Sending high-frequency concurrent requests to S3 prefixes triggers HTTP 503 Slow Down responses, which typical tools handle with naive static retries, leading to retry storms.

Kumokura requires an engine that achieves:
- Throughput comparable to `s5cmd` and `rclone`.
- Strictly bounded memory allocation via zero-allocation buffer reuse.
- Fine-grained multipart chunk checkpointing backed by durable storage.
- Dynamic concurrency adjustment adapting in real time to network throughput and server throttling.

---

## 2. Decision

We implement a dedicated **Adaptive Transfer & Sync Engine** adhering to the following architectural design:

### 2.1 Bimodal Worker Pipeline
Transfer jobs are classified into two dedicated execution pipelines based on object payload size:
- **Small File Pipeline (< 16 MiB)**: Executed as atomic `PutObject` calls. Utilizes a high-throughput persistent HTTP connection pool (up to 500 idle connections) with Keep-Alive to eliminate TCP and TLS handshake overhead.
- **Large File Pipeline ($\ge$ 16 MiB)**: Executed through S3 Multipart Upload API. Managed via coordinated chunk worker pools reading from a single shared OS file descriptor using `io.NewSectionReader`.

### 2.2 Memory-Bounded `sync.Pool` Buffers
To prevent memory leaks and GC latency:
- Byte buffers are pooled via Go `sync.Pool` across standard chunk bucket sizes (5 MiB, 16 MiB, 64 MiB).
- A global memory semaphore enforces a strict upper memory ceiling (default 512 MiB total across all active transfer buffers).
- Workers acquire a permit from the memory semaphore before pulling a buffer from the pool and releasing it immediately upon part upload completion.

### 2.3 Dynamic AIMD Concurrency Governor
Worker concurrency is governed dynamically using an Additive Increase, Multiplicative Decrease (AIMD) algorithm:
- Initial concurrency starts at 8 workers.
- **Additive Increase**: Every sequence of 50 consecutive successful part uploads without latency spikes or HTTP errors increments worker concurrency by $+1$ (up to maximum 64 workers).
- **Multiplicative Decrease**: Encountering an HTTP 503 Slow Down, 429 Too Many Requests, or connection timeout immediately cuts concurrency by 40% ($C_{next} = \max(2, \lfloor C_{curr} \times 0.6 \rfloor)$) and triggers randomized exponential backoff jitter.

### 2.4 ACID SQLite WAL Checkpoint Ledger
Every transfer operation is tracked in an embedded SQLite database operating in WAL mode:
- On upload creation, the multipart `UploadId` and all calculated parts (offset, length, part number) are registered with status `PENDING`.
- As each part uploads successfully, its ETag and status `COMPLETED` are recorded within an atomic SQLite transaction.
- If interrupted, resumption queries the ledger and S3 `ListParts`, skips already completed parts, and resumes uploading exclusively remaining parts.
- Completed multi-part jobs invoke `CompleteMultipartUpload` and transition the ledger record to `COMPLETED`.

### 2.5 Sync Engine Diff Algorithm
Directory synchronization implements a 3-way tree comparison:
- Traverses local filesystem metadata (size, modtime) and remote S3 bucket keys (ETag, size, last-modified).
- Calculates a minimal operation graph: `CREATE`, `UPDATE`, `DELETE`, or `SKIP`.
- Supports `--dry-run` inspection and bi-directional conflict resolution policies (`newer-wins`, `larger-wins`, `remote-wins`, `local-wins`).

---

## 3. Consequences

### Positive:
- **Resilience**: Network dropouts and computer restarts during multi-gigabyte uploads no longer lose transferred data.
- **Predictable Footprint**: Kumokura runs reliably on resource-constrained developer workstations without memory spikes.
- **Optimal Bandwidth Utilization**: AIMD governor automatically maximizes throughput on high-speed gigabit lines while gracefully throttling on rate-limited edge providers.

### Negative / Trade-offs:
- **Database I/O Overhead**: Recording part checkpoints introduces minor SQLite disk writes, though WAL mode reduces this latency to sub-millisecond durations.
- **Cleanup Responsibility**: Aborted or abandoned multipart uploads require periodic pruning via automated lifecycle cleanup rules.
