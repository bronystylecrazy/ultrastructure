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
		opts = append(opts, config.WithBaseEndpoint(cfg.Endpoint))
	}

	return config.LoadDefaultConfig(ctx, opts...)
}

func NewAWSConfig(ctx context.Context, cfg Config) (aws.Config, error) {
	return LoadAWSConfig(ctx, cfg)
}
