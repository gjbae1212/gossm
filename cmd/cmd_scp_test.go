package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestScpInit(t *testing.T) {
	assert := assert.New(t)
	exec := viper.GetString("scp-exec")
	assert.Empty(exec)
}
