package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/middleware"
)

type Appender interface {
	Append(apiOptions *[]func(*middleware.Stack) error)
}

func NewS3Client(cfg aws.Config, s3cfg Config, appender ...Appender) *Client {
	for _, a := range appender {
		a.Append(&cfg.APIOptions)
	}
	return NewClient(cfg, WithPathStyle(s3cfg.UsePathStyle))
}
