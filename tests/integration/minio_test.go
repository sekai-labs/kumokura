package integration_test

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/adapters/s3client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockS3Server struct {
	mu      sync.Mutex
	buckets map[string]time.Time
}

func newMockS3Server() *mockS3Server {
	return &mockS3Server{
		buckets: make(map[string]time.Time),
	}
}

type listAllMyBucketsResult struct {
	XMLName xml.Name      `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListAllMyBucketsResult"`
	Owner   mockOwner     `xml:"Owner"`
	Buckets []bucketEntry `xml:"Buckets>Bucket"`
}

type mockOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type bucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

func (s *mockS3Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	bucket := ""
	if len(parts) > 0 && parts[0] != "" {
		bucket = parts[0]
	}

	switch r.Method {
	case http.MethodGet:
		if bucket == "" {
			var entries []bucketEntry
			for name, created := range s.buckets {
				entries = append(entries, bucketEntry{
					Name:         name,
					CreationDate: created.Format(time.RFC3339),
				})
			}
			res := listAllMyBucketsResult{
				Owner: mockOwner{
					ID:          "minioadmin",
					DisplayName: "minioadmin",
				},
				Buckets: entries,
			}
			w.Header().Set("Content-Type", "application/xml")
			_ = xml.NewEncoder(w).Encode(res)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodPut:
		if bucket != "" {
			s.buckets[bucket] = time.Now().UTC()
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusBadRequest)

	case http.MethodDelete:
		if bucket != "" {
			delete(s.buckets, bucket)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusBadRequest)

	case http.MethodHead:
		if bucket != "" {
			if _, ok := s.buckets[bucket]; ok {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getMinIOEndpoint(t *testing.T) (string, func()) {
	resp, err := http.Get("http://127.0.0.1:9000/minio/health/live")
	if err == nil {
		resp.Body.Close()
		return "http://127.0.0.1:9000", func() {}
	}

	server := httptest.NewServer(newMockS3Server())
	return server.URL, server.Close
}

func TestS3LifecycleIntegration(t *testing.T) {
	endpoint, cleanup := getMinIOEndpoint(t)
	defer cleanup()

	factory, err := s3client.NewClientFactory(nil)
	require.NoError(t, err)

	acc, err := accountdomain.NewAccount(
		"minio-test-id",
		"MinIO Local Test",
		accountdomain.TypeMinIO,
		endpoint,
		"us-east-1",
		true,
	)
	require.NoError(t, err)

	creds, err := accountdomain.NewCredentials("minioadmin", "minioadmin", "")
	require.NoError(t, err)

	ctx := context.Background()
	client, err := factory.Build(ctx, acc, creds)
	require.NoError(t, err)
	require.NotNil(t, client)

	bucketName := "kumokura-integration-test-bucket"

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})
	require.NoError(t, err)

	listOut, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	found := false
	for _, b := range listOut.Buckets {
		if b.Name != nil && *b.Name == bucketName {
			found = true
			break
		}
	}
	assert.True(t, found, "created bucket should appear in bucket list")

	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &bucketName,
	})
	require.NoError(t, err)

	listOutAfter, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	foundAfter := false
	for _, b := range listOutAfter.Buckets {
		if b.Name != nil && *b.Name == bucketName {
			foundAfter = true
			break
		}
	}
	assert.False(t, foundAfter, "deleted bucket should not appear in bucket list")
}
