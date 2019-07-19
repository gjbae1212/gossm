package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestSshInit(t *testing.T) {
	assert := assert.New(t)
	exec := viper.GetString("ssh-exec")
	assert.Empty(exec)
}
