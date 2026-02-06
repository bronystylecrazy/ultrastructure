package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Config struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

func LoadAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	opts := make([]func(*config.LoadOptions) error, 0, 3)

	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
			return aws.Config{}, fmt.Errorf("both access key id and secret access key must be set")
		}
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	if cfg.Endpoint != "" {
		opts = append(opts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
				if service != s3.ServiceID {
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				}
				signingRegion := region
				if cfg.Region != "" {
					signingRegion = cfg.Region
				}
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					SigningRegion:     signingRegion,
					HostnameImmutable: true,
				}, nil
			}),
		))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}
