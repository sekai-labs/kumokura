package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name    string
		bucket  string
		wantErr bool
	}{
		{
			name:    "valid standard name",
			bucket:  "my-bucket-123",
			wantErr: false,
		},
		{
			name:    "valid name with dot",
			bucket:  "my.bucket.name",
			wantErr: false,
		},
		{
			name:    "too short",
			bucket:  "ab",
			wantErr: true,
		},
		{
			name:    "too long",
			bucket:  "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789",
			wantErr: true,
		},
		{
			name:    "uppercase characters",
			bucket:  "MyBucket",
			wantErr: true,
		},
		{
			name:    "consecutive dots",
			bucket:  "my..bucket",
			wantErr: true,
		},
		{
			name:    "dot dash sequence",
			bucket:  "my.-bucket",
			wantErr: true,
		},
		{
			name:    "dash dot sequence",
			bucket:  "my-.bucket",
			wantErr: true,
		},
		{
			name:    "starts with dash",
			bucket:  "-mybucket",
			wantErr: true,
		},
		{
			name:    "ends with dash",
			bucket:  "mybucket-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBucketName(tt.bucket)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewBucket(t *testing.T) {
	now := time.Now()
	b, err := NewBucket("valid-bucket-name", "us-east-1", now)
	assert.NoError(t, err)
	assert.Equal(t, "valid-bucket-name", b.Name)
	assert.Equal(t, "us-east-1", b.Region)
	assert.Equal(t, now, b.CreationDate)

	_, err = NewBucket("INVALID", "us-east-1", now)
	assert.Error(t, err)
}
