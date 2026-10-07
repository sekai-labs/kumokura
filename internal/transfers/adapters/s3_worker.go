package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type S3TransferAPI interface {
	CreateMultipartUpload(ctx context.Context, params *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(ctx context.Context, params *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CompleteMultipartUpload(ctx context.Context, params *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(ctx context.Context, params *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

type S3TransferWorker struct {
	client     S3TransferAPI
	bufferPool *TieredBufferPool
}

func NewS3TransferWorker(client S3TransferAPI, bufferPool *TieredBufferPool) *S3TransferWorker {
	return &S3TransferWorker{
		client:     client,
		bufferPool: bufferPool,
	}
}

func (w *S3TransferWorker) InitiateMultipartUpload(ctx context.Context, bucket, key string) (string, error) {
	out, err := w.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UploadId), nil
}

func (w *S3TransferWorker) UploadPartChunk(ctx context.Context, bucket, key, uploadID string, part domain.TransferPart, file *os.File) (string, error) {
	bufPtr := w.bufferPool.Get(int(part.Size))
	defer w.bufferPool.Put(int(part.Size), bufPtr)

	slice := (*bufPtr)[:part.Size]
	_, err := file.ReadAt(slice, part.Offset)
	if err != nil && err != io.EOF {
		return "", err
	}

	out, err := w.client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(part.PartNumber),
		Body:       bytes.NewReader(slice),
	})
	if err != nil {
		return "", err
	}

	return aws.ToString(out.ETag), nil
}

func (w *S3TransferWorker) CompleteMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []domain.TransferPart) error {
	completedParts := make([]s3types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completedParts = append(completedParts, s3types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		})
	}

	_, err := w.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	return err
}

func (w *S3TransferWorker) AbortMultipartUpload(ctx context.Context, bucket, key, uploadID string) error {
	_, err := w.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}

func (w *S3TransferWorker) DownloadRangeChunk(ctx context.Context, bucket, key string, part domain.TransferPart, destFile *os.File) error {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", part.Offset, part.Offset+part.Size-1)
	out, err := w.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()

	bufPtr := w.bufferPool.Get(int(part.Size))
	defer w.bufferPool.Put(int(part.Size), bufPtr)

	slice := (*bufPtr)[:part.Size]
	_, err = io.ReadFull(out.Body, slice)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}

	_, err = destFile.WriteAt(slice, part.Offset)
	return err
}
