package internal

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func newAWSHTTPClient() aws.HTTPClient {
	return awshttp.NewBuildableClient().
		WithDialerOptions(func(d *net.Dialer) {
			d.Timeout = defaultDialTimeout
			d.KeepAlive = defaultDialKeepAlive
		}).
		WithTransportOptions(func(tr *http.Transport) {
			tr.Proxy = http.ProxyFromEnvironment
			tr.ForceAttemptHTTP2 = false
			tr.MaxIdleConns = defaultMaxIdleConns
			tr.IdleConnTimeout = defaultIdleConnTimeout
			tr.TLSHandshakeTimeout = defaultTLSHandshakeTimeout
			tr.ExpectContinueTimeout = defaultExpectContinueTimeout
			tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		})
}

const (
	defaultDialTimeout           = 30 * time.Second
	defaultDialKeepAlive         = 30 * time.Second
	defaultMaxIdleConns          = 100
	defaultIdleConnTimeout       = 90 * time.Second
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
)

// NewConfig creates a config for accessing AWS with passing credential parameters.
func NewConfig(ctx context.Context, key, secret, session, region, roleArn string) (aws.Config, error) {
	var (
		opts []func(*config.LoadOptions) error
		cfg  aws.Config
		err  error
	)
	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParams)
	}

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	opts = append(opts, config.WithHTTPClient(newAWSHTTPClient()))

	// if parameters for credentials doesn't pass it.
	if key == "" || secret == "" {
		cfg, err = config.LoadDefaultConfig(ctx, opts...)
	} else {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(key, secret, session)))
		cfg, err = config.LoadDefaultConfig(ctx, opts...)
	}
	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds
	if roleArn != "" {
		sts := sts.NewFromConfig(cfg)
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts, roleArn))
	}

	return cfg, nil
}

// NewSharedConfig creates a config for accessing AWS that is based on shared files, such as credentials file.
func NewSharedConfig(ctx context.Context, profile string, sharedConfigFiles, sharedCredentialsFiles []string) (aws.Config, error) {
	if ctx == nil {
		return aws.Config{}, WrapError(ErrInvalidParams)
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile),
		config.WithSharedConfigFiles(sharedConfigFiles),
		config.WithSharedCredentialsFiles(sharedCredentialsFiles),
		config.WithHTTPClient(newAWSHTTPClient()),
	)
	if err != nil {
		return aws.Config{}, WrapError(err)
	}

	return cfg, nil
}
