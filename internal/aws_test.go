package internal

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		ctx     context.Context
		key     string
		secret  string
		token   string
		region  string
		roleArn string
		isErr   bool
	}{
		"fail":    {isErr: true},
		"success": {ctx: context.Background(), key: mockAwsKey, secret: mockAwsSecret, region: mockRegion, isErr: false},
	}

	for _, t := range tests {
		_, err := NewConfig(t.ctx, t.key, t.secret, t.token, t.region, t.roleArn)
		assert.Equal(t.isErr, err != nil)
	}
}

func TestNewSharedConfig(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		ctx               context.Context
		profile           string
		sharedCredentials []string
		sharedConfigs     []string
		isErr             bool
	}{
		"fail": {isErr: true},
		"success": {
			ctx:               context.Background(),
			profile:           mockProfile,
			sharedConfigs:     []string{config.DefaultSharedConfigFilename()},
			sharedCredentials: []string{config.DefaultSharedCredentialsFilename()},
			isErr:             false},
	}

	for _, t := range tests {
		_, err := NewSharedConfig(t.ctx, t.profile, t.sharedConfigs, t.sharedCredentials)
		assert.Equal(t.isErr, err != nil)
	}
}
