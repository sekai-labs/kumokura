package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateObjectKey(t *testing.T) {
	err := ValidateObjectKey("valid/path/to/file.txt")
	assert.NoError(t, err)

	err = ValidateObjectKey("")
	assert.ErrorIs(t, err, ErrEmptyObjectKey)

	err = ValidateObjectKey("   ")
	assert.ErrorIs(t, err, ErrEmptyObjectKey)
}
