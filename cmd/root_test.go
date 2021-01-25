package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
	}{
		"nil": {},
	}

	for _, _ = range tests {
		Execute()
		_ = assert
	}
}
