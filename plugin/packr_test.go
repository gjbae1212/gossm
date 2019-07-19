package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlugin(t *testing.T) {
	assert := assert.New(t)
	_, err := GetPlugin()
	assert.NoError(err)
}

func TestGetPluginFileName(t *testing.T) {
	assert := assert.New(t)
	name := GetPluginFileName()
	assert.NotEmpty(name)
}
