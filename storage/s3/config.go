package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type Config struct {
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UsePathStyle    bool   `mapstructure:"use_path_style"`
	Bucket          string `mapstructure:"bucket"`
	OtelEnabled     bool   `mapstructure:"otel_enabled"`
}

func LoadAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	return LoadAWSConfigWithOptions(ctx, cfg)
}

type LoadOption func(*loadOptions)

type loadOptions struct {
	enableOtel bool
}

func WithOtel() LoadOption {
	return func(o *loadOptions) {
		o.enableOtel = true
	}
}

func LoadAWSConfigWithOptions(ctx context.Context, cfg Config, opts ...LoadOption) (aws.Config, error) {
	options := loadOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	awsOpts := make([]func(*config.LoadOptions) error, 0, 3)

	if cfg.Region != "" {
		awsOpts = append(awsOpts, config.WithRegion(cfg.Region))
	}

	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
			return aws.Config{}, fmt.Errorf("both access key id and secret access key must be set")
		}
		awsOpts = append(awsOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	if cfg.Endpoint != "" {
		awsOpts = append(awsOpts, config.WithBaseEndpoint(cfg.Endpoint))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, awsOpts...)
	if err != nil {
		return aws.Config{}, err
	}

	return awsCfg, nil
}

func NewAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	return LoadAWSConfig(ctx, cfg)
}

func NewAWSConfigWithOptions(ctx context.Context, cfg Config, opts ...LoadOption) (aws.Config, error) {
	return LoadAWSConfigWithOptions(ctx, cfg, opts...)
}
