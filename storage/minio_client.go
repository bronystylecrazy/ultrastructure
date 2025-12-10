package storage

import (
	"context"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func CreateMinioClient(ctx context.Context, config Config, log *zap.Logger) (*minio.Client, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, config.BucketName)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(ctx, config.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		log.Info("bucket created:", zap.String("bucket", config.BucketName))
	} else {
		log.Info("bucket already exists:", zap.String("bucket", config.BucketName))
	}

	return client, nil
}
