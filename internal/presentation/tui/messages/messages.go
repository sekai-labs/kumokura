package messages

import (
	accDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
	syncPorts "github.com/sekai-labs/kumokura/internal/synchronization/ports"
	transferDomain "github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type AccountsLoadedMsg struct {
	Accounts []accDomain.Account
	Err      error
}

type BucketsLoadedMsg struct {
	Buckets []bucketDomain.Bucket
	Err     error
}

type ObjectsLoadedMsg struct {
	Result objPorts.ListObjectsResult
	Err    error
}

type ObjectMetadataLoadedMsg struct {
	Metadata objDomain.ObjectMetadata
	Tags     []objDomain.ObjectTag
	Err      error
}

type ContentPreviewLoadedMsg struct {
	Key     string
	Content string
	Err     error
}

type TransfersLoadedMsg struct {
	Jobs []transferDomain.TransferJob
	Err  error
}

type TransferProgressMsg struct {
	JobID   string
	Status  transferDomain.JobStatus
	Metrics transferDomain.TransferMetrics
}

type ErrorMsg struct {
	Err error
}

type StatusNotificationMsg struct {
	Message string
}

type UploadFinishedMsg struct {
	Bucket string
	Key    string
	Err    error
}

type DownloadFinishedMsg struct {
	Bucket string
	Key    string
	Path   string
	Err    error
}

type FolderUploadFinishedMsg struct {
	Bucket      string
	Prefix      string
	TotalCount  int
	TotalBytes  int64
	FailedCount int
	Err         error
}
type BatchUploadFinishedMsg struct {
	Bucket      string
	Prefix      string
	TotalCount  int
	TotalBytes  int64
	FailedCount int
	Err         error
}

type FolderDownloadFinishedMsg struct {
	Bucket      string
	Prefix      string
	LocalDest   string
	TotalCount  int
	TotalBytes  int64
	FailedCount int
	Err         error
}

type PresignedURLGeneratedMsg struct {
	URL    string
	Key    string
	Copied bool
	Err    error
}

type SyncJobsLoadedMsg struct {
	Jobs []*syncPorts.SyncJobRecord
	Err  error
}
