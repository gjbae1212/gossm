package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/stretchr/testify/assert"
)

func TestCallProcess(t *testing.T) {
	assert := assert.New(t)

	err := callSubprocess("echo", "hello")
	assert.NoError(err)
}

func TestPrintReady(t *testing.T) {
	assert := assert.New(t)
	printReady("hello")
	_ = assert
}

func TestFindInstanceId(t *testing.T) {
	assert := assert.New(t)
	var err error
	awsSession, err = session.NewSessionWithOptions(session.Options{
		Profile:           "default",
		SharedConfigState: session.SharedConfigEnable,
	})
	if err == nil {
		_, err = findInstanceIdByIp("us-east-1", "")
		assert.NoError(err)
	}
}
