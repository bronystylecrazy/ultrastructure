package s3

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Option func(*clientOptions)

type clientOptions struct {
	s3Options      []func(*s3.Options)
	presignOptions []func(*s3.PresignOptions)
}

func WithS3Option(opt func(*s3.Options)) Option {
	return func(o *clientOptions) {
		o.s3Options = append(o.s3Options, opt)
	}
}

func WithPresignOption(opt func(*s3.PresignOptions)) Option {
	return func(o *clientOptions) {
		o.presignOptions = append(o.presignOptions, opt)
	}
}

func WithPathStyle(pathStyle bool) Option {
	return WithS3Option(func(o *s3.Options) {
		o.UsePathStyle = pathStyle
	})
}

type Client struct {
	S3      *s3.Client
	Presign *s3.PresignClient
}

func NewClient(cfg aws.Config, opts ...Option) *Client {
	options := clientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	client := s3.NewFromConfig(cfg, options.s3Options...)
	presign := s3.NewPresignClient(client, options.presignOptions...)

	return &Client{
		S3:      client,
		Presign: presign,
	}
}
