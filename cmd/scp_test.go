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

func TestScp(t *testing.T) {
	assert := assert.New(t)
	err := setSCP()
	assert.Error(err)
	assert.Equal("[err] [required] exec argument", err.Error())

	viper.Set("region", "ap-northeast-2")
	viper.Set("scp-exec", "invalid")
	err = setSCP()
	assert.Error(err)
	assert.Equal("[err] invalid exec argument", err.Error())

	viper.Set("scp-exec", "bb aa")
	err = setSCP()
	assert.Error(err)
	assert.Equal("[err] invalid scp args", err.Error())

	viper.Set("scp-exec", "file aa@unknown")
	err = setSCP()
	assert.Error(err)
	assert.Equal("[err] invalid server domain name", err.Error())
}