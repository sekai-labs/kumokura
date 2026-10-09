package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
)

type S3ClientAPI interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type S3Scanner struct {
	client S3ClientAPI
	bucket string
}

func NewS3Scanner(client S3ClientAPI, bucket string) *S3Scanner {
	return &S3Scanner{
		client: client,
		bucket: bucket,
	}
}

var _ ports.SyncScanner = (*S3Scanner)(nil)

func (s *S3Scanner) Scan(ctx context.Context, prefix string, filter domain.Filter) (map[string]*domain.FileEntry, error) {
	results := make(map[string]*domain.FileEntry)
	normPrefix := strings.TrimPrefix(prefix, "/")
	if normPrefix != "" && !strings.HasSuffix(normPrefix, "/") {
		normPrefix += "/"
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(normPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects page for bucket %s: %w", s.bucket, err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			if strings.HasSuffix(key, "/") {
				continue
			}

			relPath := strings.TrimPrefix(key, normPrefix)
			cleanRel := strings.TrimPrefix(relPath, "/")
			if cleanRel == ".." || strings.HasPrefix(cleanRel, "../") || strings.Contains(cleanRel, "/../") {
				continue
			}
			if !filter.Matches(relPath) {
				continue
			}

			etag := domain.NormalizeETag(aws.ToString(obj.ETag))
			modTime := aws.ToTime(obj.LastModified).UTC()
			size := aws.ToInt64(obj.Size)

			results[relPath] = &domain.FileEntry{
				RelativePath: relPath,
				Size:         size,
				ModTime:      modTime,
				ETag:         etag,
				Checksum:     etag,
				IsDir:        false,
			}
		}
	}

	return results, nil
}
