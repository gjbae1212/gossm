package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmd(t *testing.T) {
	assert := assert.New(t)

	err := setMultiTarget()
	assert.Error(err)
	assert.Equal("[err] don't exist region \n", err.Error())
}
