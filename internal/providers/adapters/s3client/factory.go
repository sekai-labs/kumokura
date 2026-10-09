package s3client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/platform/transport"
)

type FactoryOptions struct {
	HTTPClient       *http.Client
	TransportOptions *transport.ClientOptions
}

type ClientFactory struct {
	httpClient *http.Client
}

func NewClientFactory(opts *FactoryOptions) (*ClientFactory, error) {
	if opts != nil && opts.HTTPClient != nil {
		return &ClientFactory{httpClient: opts.HTTPClient}, nil
	}

	var trOpts transport.ClientOptions
	if opts != nil && opts.TransportOptions != nil {
		trOpts = *opts.TransportOptions
	} else {
		trOpts = transport.DefaultClientOptions()
	}

	client, err := transport.NewHTTPClient(trOpts)
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &ClientFactory{httpClient: client}, nil
}

func (f *ClientFactory) Build(ctx context.Context, account *accountdomain.Account, creds accountdomain.Credentials) (*s3.Client, error) {
	if account == nil {
		return nil, accountdomain.ErrInvalidAccountID
	}

	region := account.Region
	if region == "" {
		region = "us-east-1"
	}

	credProvider := credentials.NewStaticCredentialsProvider(
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken,
	)

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credProvider),
		awsconfig.WithHTTPClient(f.httpClient),
		awsconfig.WithClientLogMode(0),
		awsconfig.WithLogger(logging.Nop{}),
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws default config: %w", err)
	}

	s3Options := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = account.UsePathStyle
			o.ClientLogMode = 0
			o.Logger = logging.Nop{}
		},
	}

	if account.Endpoint != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(account.Endpoint)
		})
	}

	return s3.NewFromConfig(cfg, s3Options...), nil
}
