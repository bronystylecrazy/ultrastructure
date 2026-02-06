package s3

import "github.com/aws/aws-sdk-go-v2/aws"

func NewS3Client(cfg aws.Config, s3cfg Config) *Client {
	return NewClient(cfg, WithPathStyle(s3cfg.UsePathStyle))
}
