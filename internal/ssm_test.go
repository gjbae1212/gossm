package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestFindInstances(t *testing.T) {
	assert := assert.New(t)

	cfg, err := NewConfig(context.Background(), "", "", "", "", "")
	assert.NoError(err)

	tests := map[string]struct {
		ctx   context.Context
		cfg   aws.Config
		isErr bool
	}{
		"success": {
			ctx:   context.Background(),
			cfg:   cfg,
			isErr: false,
		},
	}

	for _, t := range tests {
		result, err := FindInstances(t.ctx, t.cfg)
		assert.Equal(t.isErr, err != nil)
		fmt.Println(len(result))
	}
}
func TestFindInstanceIdsWithConnectedSSM(t *testing.T) {
	assert := assert.New(t)

	cfg, err := NewConfig(context.Background(), "", "", "", "", "")
	assert.NoError(err)

	tests := map[string]struct {
		ctx   context.Context
		cfg   aws.Config
		isErr bool
	}{
		"success": {
			ctx:   context.Background(),
			cfg:   cfg,
			isErr: false,
		},
	}

	for _, t := range tests {
		result, err := FindInstanceIdsWithConnectedSSM(t.ctx, t.cfg)
		assert.Equal(t.isErr, err != nil)
		fmt.Println(len(result))
	}
}

func TestAskUser(t *testing.T)                {}
func TestAskTeam(t *testing.T)                {}
func TestAskRegion(t *testing.T)              {}
func TestAskTarget(t *testing.T)              {}
func TestAskMultiTarget(t *testing.T)         {}
func TestAskPorts(t *testing.T)               {}
func TestFindInstanceIdByIp(t *testing.T)     {}
func TestFindDomainByInstanceId(t *testing.T) {}
func TestCreateStartSession(t *testing.T)     {}
func TestDeleteStartSession(t *testing.T)     {}
func TestSendCommand(t *testing.T)            {}
func TestPrintCommandInvocation(t *testing.T) {}
func TestGenerateSSHExecCommand(t *testing.T)    {}
func TestPrintReady(t *testing.T)             {}
