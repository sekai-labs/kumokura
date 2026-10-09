package messages

import (
	accDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
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
