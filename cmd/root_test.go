package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)
	Execute()
	_ = assert
}

func TestMakeSession(t *testing.T) {
	assert := assert.New(t)

	_, p, err := makeSession("", "")
	assert.NoError(err)
	assert.Equal("default", p)

	_, p, err = makeSession("", "test-account")
	assert.NoError(err)
	assert.Equal("test-account", p)

	os.Setenv("AWS_PROFILE", "test-account")
	_, p, err = makeSession("", "")
	assert.NoError(err)
	assert.Equal("test-account", p)
}
