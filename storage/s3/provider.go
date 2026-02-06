package s3

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/middleware"
)

type Appender interface {
	Append(apiOptions *[]func(*middleware.Stack) error)
}

func NewS3Client(cfg aws.Config, s3cfg Config, appender ...Appender) *Client {

	fmt.Println("appenders:", len(appender))

	for _, a := range appender {
		a.Append(&cfg.APIOptions)
	}
	return NewClient(cfg, WithPathStyle(s3cfg.UsePathStyle))
}
