package s3

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type PutInputOption func(*s3.PutObjectInput)

func WithPutContentType(contentType string) PutInputOption {
	return func(in *s3.PutObjectInput) {
		in.ContentType = aws.String(contentType)
	}
}

func WithPutACL(acl string) PutInputOption {
	return func(in *s3.PutObjectInput) {
		in.ACL = s3types.ObjectCannedACL(acl)
	}
}

// UploadFileHeader uploads a multipart file using PutObject.
func UploadFileHeader(ctx context.Context, uploader Uploader, bucket, key string, fileHeader *multipart.FileHeader, opts ...PutInputOption) (*s3.PutObjectOutput, error) {
	if fileHeader == nil {
		return nil, fmt.Errorf("file header cannot be nil")
	}
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key must be set")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          file,
		ContentLength: aws.Int64(fileHeader.Size),
	}

	if ct := fileHeader.Header.Get("Content-Type"); ct != "" {
		input.ContentType = aws.String(ct)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(input)
		}
	}

	return uploader.PutObject(ctx, input)
}
