# High-Performance Transfer & Sync Engine Engineering

## 1. Executive Summary & Problem Formulation

High-throughput object storage operations face unique bottlenecks distinct from standard local filesystem I/O:
1. **Network Latency & Round-Trip Overhead (RTT)**: S3 operations are bounded by HTTPS handshakes and TLS session negotiation. Performing synchronous serial requests over high-latency WAN connections cripples throughput (e.g., uploading 10,000 50KB files serially at 50ms RTT yields a theoretical maximum of only 20 files/sec, or ~1 MB/s, regardless of available Gigabit bandwidth).
2. **Buffer Allocation & Garbage Collection (GC) Pressure**: High concurrency naive implementations allocate arbitrary byte slices (`make([]byte, partSize)`) per worker. At 64 concurrent 16MB part uploads, this churns 1GB of memory per pipeline wave, triggering Go GC stop-the-world pauses and memory thrashing.
3. **Endpoint Rate Limiting & HTTP 503 Slow Down**: Over-aggressive concurrency causes S3 providers (especially AWS S3 prefix partitions and Cloudflare R2) to return HTTP 503 Slow Down or 429 Too Many Requests, causing cascading retry storms.
4. **Failure Recovery on Multi-Gigabyte Payloads**: Dropping a connection at 98% of a 50GB file upload without checkpointed part tracking forces re-uploading all 50GB from byte zero.

To achieve world-class transfer speeds matching and exceeding tools like `s5cmd` and `rclone`, Kumokura implements an **Adaptive Multipart & Pipelined Concurrency Engine** backed by an ACID SQLite WAL ledger.

---

## 2. Comparative Analysis of High-Performance Transfer Engines

### 2.1 S3 Browser
- **Mechanism**: Custom HTTP connection pooling in .NET. Splits files larger than a user-configured threshold (default 5MB or 16MB) into multipart parts.
- **Concurrency Model**: Static thread pool (user selects 1 to 20 threads).
- **Shortcomings**: Concurrency is static; does not back off during HTTP 503 spikes. Checkpoint recovery relies on ephemeral session logs.

### 2.2 s5cmd (Peak Games)
- **Mechanism**: Go worker pool architecture using channel-based task queues.
- **Concurrency Model**:
  - Global Goroutine worker pool (configurable via `-numworkers`, default 256).
  - Parallel object listing: decoupled directory traversal Goroutines feed upload/download channels asynchronously.
  - Multipart uploads split files into fixed parts (default 50MB) and emit individual `UploadPart` tasks to the worker pool.
- **Key Takeaways for Kumokura**:
  - Decoupling directory listing from transfer execution is essential to eliminate queue starvation.
  - Minimal object metadata instantiation reduces memory overhead during million-file scans.

### 2.3 rclone
- **Mechanism**: Sophisticated token-bucket rate limiter, dynamic chunk sizing, memory buffer pools (`--buffer-size`), and persistent sync state.
- **Concurrency Model**:
  - `--transfers` dictates concurrent file transfers (default 4).
  - `--checkers` dictates concurrent comparison threads (default 8).
  - `--s3-upload-concurrency` dictates concurrent parts per multipart upload (default 4).
- **Key Takeaways for Kumokura**:
  - Differentiating *file concurrency* (concurrent independent files) from *part concurrency* (concurrent parts within a single large file) avoids socket saturation.
  - Token-bucket rate limiting provides precise egress/ingress bandwidth capping.

### 2.4 AWS SDK for Go v2 Transfer Manager (`feature/s3/manager`)
- **Mechanism**:
  - `s3manager.Uploader`: reads from an `io.Reader`, slices chunks of `PartSize` (default 5MB), and dispatches up to `Concurrency` (default 5) parallel `UploadPart` calls.
  - Automatically calculates multipart requirement when source implements `io.Seeker` or provides known size.
- **Limitations**:
  - Default `PartSize` of 5MB is sub-optimal for multi-gigabyte files (S3 limits uploads to at most 10,000 parts; a 5MB part size fails for files > 50GB).
  - Static concurrency per upload; no global AIMD (Additive Increase, Multiplicative Decrease) dynamic rate adjustment.
  - In-memory only; no durable ledger for resuming interrupted uploads across application restarts.

---

## 3. Kumokura Transfer Engine Architecture

```
+-----------------------------------------------------------------------------------------+
|                                KUMOKURA TRANSFER ENGINE                                 |
+-----------------------------------------------------------------------------------------+
|                                                                                         |
|   +-----------------------+     +----------------------+     +----------------------+   |
|   |  Directory Scanner    | --> | Task Dispatcher      | --> | Dynamic AIMD         |   |
|   |  (Pipelined Channel)  |     | (Priority Work Queue)|     | Concurrency Governor |   |
|   +-----------------------+     +----------------------+     +----------+-----------+   |
|                                                                         |               |
|                                         +-------------------------------+               |
|                                         |                                               |
|               +-------------------------v-------------------------+                     |
|               |              Worker Pool Execution                |                     |
|               |  +--------------------+    +--------------------+ |                     |
|               |  | Small File Worker  |    | Multipart Worker   | |                     |
|               |  | (Single PutObject) |    | (Chunked Pipeline) | |                     |
|               |  +--------------------+    +---------+----------+ |                     |
|               +--------------------------------------|------------+                     |
|                                                      |                                  |
|               +-----------------------+              |   +----------------------+       |
|               | sync.Pool Sized       |<-------------+   | SQLite WAL           |       |
|               | Buffer Allocator      |                  | Checkpoint Ledger    |       |
|               +-----------------------+                  +----------------------+       |
+-----------------------------------------------------------------------------------------+
```

### 3.1 Adaptive Multipart Algorithm & Chunk Sizing

The Amazon S3 multipart upload specification imposes hard bounds:
- **Minimum part size**: 5 MiB (except for the final part).
- **Maximum part size**: 5 GiB.
- **Maximum number of parts per object**: 10,000 parts.
- **Maximum single object size**: 5 TiB.

Kumokura implements an **Adaptive Chunk Calculator** that dynamically computes the optimal part size based on total file size:

```
Function CalculatePartSize(totalSize int64) -> int64:
  BasePartSize = 16 MiB   // Default optimal throughput chunk
  MinPartSize  = 5 MiB
  MaxPartSize  = 5 GiB
  MaxParts     = 10,000

  If totalSize <= BasePartSize:
    Return totalSize  // Non-multipart PutObject

  // Ensure total parts will never exceed 10,000 (with safety margin: target max 8,000 parts)
  RequiredPartSize = Ceil(totalSize / 8,000)

  // Align to next power of 2 or 8MB boundary
  OptimalPartSize = Max(BasePartSize, RequiredPartSize)
  OptimalPartSize = Min(OptimalPartSize, MaxPartSize)

  Return OptimalPartSize
```

#### Chunk Sizing Benchmarks & Strategy:
- **< 16 MiB**: Single `PutObject` operation (zero multipart initialization overhead).
- **16 MiB to 128 GiB**: 16 MiB chunks (~8,000 parts max). Provides high concurrency parallel uploading while keeping memory footprint bounded.
- **128 GiB to 500 GiB**: 64 MiB chunks (~7,800 parts max).
- **500 GiB to 5 TiB**: 256 MiB to 512 MiB chunks (~10,000 parts max).

---

### 3.2 Memory-Bounded Buffer Pool (`sync.Pool`)

To eliminate garbage collection latency and heap churn during high-concurrency transfers, Kumokura employs tiered byte slice buffer pools using Go's `sync.Pool`.

```go
// Conceptual architecture: Zero heap allocation in steady state
type BufferPool struct {
    smallPool sync.Pool // 5 MiB buffers
    stdPool   sync.Pool // 16 MiB buffers
    largePool sync.Pool // 64 MiB buffers
}
```

- Each worker borrows a buffer from the pool, reads the designated byte slice from disk using `io.ReadAtLeast` or `io.NewSectionReader`, executes the HTTP upload request, and returns the buffer to the pool via a `defer pool.Put(buf)`.
- **Global Memory Guard**: A semaphore limits the maximum total memory held across active buffers (e.g., maximum 512 MiB total buffer memory). If high-priority large file uploads threaten to exceed the memory cap, concurrency dynamically yields until buffers are freed.

---

### 3.3 Dynamic Concurrency Controller (AIMD Governor)

Rather than fixing concurrency at a static integer (e.g. 16 workers), Kumokura features an **AIMD (Additive Increase, Multiplicative Decrease)** concurrency governor:

1. **Initial State**: Starts at baseline concurrency (e.g., 8 workers).
2. **Additive Increase**: For every consecutive batch of $N$ successful uploads/part uploads with stable RTT and zero HTTP errors, concurrency increases by $+1$ worker (up to a configured ceiling, e.g., 64).
3. **Multiplicative Decrease**: Upon encountering an HTTP 503 Slow Down, 429 Too Many Requests, or network timeout:
   - Immediate worker concurrency is throttled: $C_{new} = \max(C_{min}, \lfloor C_{current} \times 0.6 \rfloor)$.
   - Workers back off using randomized exponential jitter:
     $$T_{backoff} = 2^{retry} \times 100\text{ms} + \text{rand}(0, 50\text{ms})$$
   - Prevents thundering herd problems on S3 prefix partitions.

---

### 3.4 Small-File vs. Large-File Pipeline Optimization

The performance profile of small files (< 1MB) is completely different from large files (> 500MB):
- **Small-File Bottleneck**: HTTP handshakes, header processing, TCP slow-start, TLS renegotiation, and S3 metadata catalog latency.
- **Large-File Bottleneck**: Bandwidth saturation, TCP window sizing, socket buffer capacity, disk read throughput.

Kumokura solves this via **Bimodal Worker Scheduling**:
1. **Directory Scanner Pipeline**: Traversal runs in dedicated Goroutines, categorizing files into two internal ring buffers:
   - `SmallFileQueue` (< 16 MiB)
   - `LargeFileQueue` (>= 16 MiB)
2. **HTTP Transport Optimization for Small Files**:
   - `MaxIdleConns`: 500
   - `MaxIdleConnsPerHost`: 100
   - `IdleConnTimeout`: 90 seconds
   - TLS session resumption enabled to eliminate TLS full-handshake latency.
   - Pipelined batch workers pull up to 64 small files concurrently, executing single-call `PutObject` operations.
3. **Multipart Pipelining for Large Files**:
   - Up to $M$ parts of the same file are transferred concurrently.
   - Readers operate via concurrent `io.NewSectionReader` off a single read-only OS file descriptor, eliminating redundant file handles.

---

### 3.5 SQLite WAL Checkpoint Ledger & Crash Recovery

Every transfer job is registered in an embedded SQLite database using Write-Ahead Logging (`PRAGMA journal_mode=WAL;`).

#### Schema Design:
- **`transfer_jobs`**: Job ID, source path, target bucket/key, total size, status (`QUEUED`, `RUNNING`, `PAUSED`, `COMPLETED`, `FAILED`), created/updated timestamps.
- **`transfer_parts`**: Job ID, part number, byte offset, part size, S3 Upload ID, part ETag, status (`PENDING`, `UPLOADING`, `COMPLETED`, `FAILED`).

#### Resumption Lifecycle:
1. When initiating a transfer > 16 MiB, Kumokura calls `CreateMultipartUpload`, obtains an `UploadId`, and records all planned parts in `transfer_parts` within a single SQLite transaction.
2. When each part completes, its ETag is recorded with status `COMPLETED`.
3. If Kumokura is killed, disconnected, or crashes:
   - On resume, Kumokura queries `transfer_parts WHERE job_id = ? AND status = 'COMPLETED'`.
   - Checks S3 `ListParts` for the active `UploadId`.
   - Skips all verified completed parts.
   - Resumes uploading strictly the remaining `PENDING` or interrupted parts.
   - Executes `CompleteMultipartUpload` with the full assembled list of ETags.
4. Auto-Abort Orphaned Uploads: Includes an abort policy to prune unfinished uploads older than 7 days to prevent S3 storage cost accumulation.

---

## 4. Reproducible Benchmark Methodology

To ensure Kumokura maintains high-throughput standards against `s5cmd`, `rclone`, and AWS CLI, our automated performance test suite (`tests/performance/`) follows a standardized, repeatable protocol:

### 4.1 Test Workloads & Datasets
1. **Workload A (Many Small Files)**:
   - 10,000 files $\times$ 10 KiB each (~100 MiB total).
   - Tests: Directory scanner throughput, HTTP connection reuse, metadata latency.
2. **Workload B (Medium Mixed Dataset)**:
   - 1,000 files $\times$ 1 MiB + 100 files $\times$ 10 MiB (~2 GiB total).
   - Tests: Queue scheduling and bimodal pipeline efficiency.
3. **Workload C (Single Giant File)**:
   - 1 file $\times$ 10 GiB.
   - Tests: Multipart chunking, buffer pool reuse, AIMD saturation, memory footprint stability.
4. **Workload D (Directory Sync Simulation)**:
   - 5,000 files on remote, 5,000 files on local, with 500 modified, 200 deleted, 300 newly created.
   - Tests: Fast listing, ETag comparison, minimal operation diff computation.

### 4.2 Target Test Environments
- **Local Isolated MinIO**: Local containerized MinIO instance pinned to dedicated CPU cores with simulated network latency (via Linux `tc-netem`, e.g., 20ms RTT, 100 Mbps to 10 Gbps).
- **Public Cloud Endpoints**: AWS S3 (us-east-1), Cloudflare R2, Wasabi.

### 4.3 Metrics Recorded
- **Wall-Clock Duration (s)**: Total elapsed time from invocation to completion.
- **Throughput (MB/s & Objects/s)**: Effective bandwidth achieved.
- **Peak RSS Memory (MB)**: Maximum resident set size recorded via `ps` / runtime memstats.
- **Total Allocations / Mallocs**: Recorded via Go `testing.B` with `-benchmem`.
- **API Call Count**: Recorded via mock HTTP round-tripper to detect redundant list calls.
