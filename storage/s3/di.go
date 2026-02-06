package s3

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/otel"
	"github.com/bronystylecrazy/ultrastructure/otelaws"
)

var AppendersGroupName = "us.s3.middlewares.appenders"

func Module(extends ...di.Node) di.Node {
	return di.Options(
		di.AutoGroup[Appender](AppendersGroupName),
		di.Module(
			"us/storage/s3",
			di.Config[Config]("storage.s3"),
			di.ConfigFile("config.toml", di.ConfigType("toml"), di.ConfigEnvOverride(), di.ConfigOptional()),
			di.Provide(NewAWSConfig),
			di.Provide(NewS3Client, di.Params(``, ``, di.Group(AppendersGroupName))),
			di.Options(di.ConvertAnys(extends)...),
		),
	)
}

func UseOtel() di.Node {
	return di.Provide(otelaws.NewMiddlewares, otel.Layer(otelaws.ScopeName))
}

func UseInterfaces() di.Node {
	return di.Options(
		di.Provide(
			func(c *Client) *s3.Client {
				return c.S3
			},
			AsS3ClientInterfaces()...,
		),
		di.Provide(
			func(c *Client) *s3.PresignClient {
				return c.Presign
			},
			di.AsSelf[Presigner](),
		),
	)
}

func AsS3ClientInterfaces() []any {
	return []any{
		di.AsSelf[S3Manager](),
		di.AsSelf[Uploader](),
		di.AsSelf[Downloader](),
		di.AsSelf[MetadataGetter](),
		di.AsSelf[Deleter](),
		di.AsSelf[BatchDeleter](),
		di.AsSelf[Copier](),
		di.AsSelf[Lister](),
		di.AsSelf[BucketLister](),
		di.AsSelf[MultipartUploader](),
		di.AsSelf[MultipartLister](),
		di.AsSelf[BucketManager](),
		di.AsSelf[AccessController](),
		di.AsSelf[Tagger](),
		di.AsSelf[PolicyManager](),
		di.AsSelf[PublicAccessBlocker](),
		di.AsSelf[Encryptor](),
		di.AsSelf[LifecycleManager](),
		di.AsSelf[Versioner](),
		di.AsSelf[WebsiteManager](),
		di.AsSelf[LoggingManager](),
		di.AsSelf[NotificationManager](),
		di.AsSelf[ReplicationManager](),
		di.AsSelf[CORSManager](),
		di.AsSelf[Accelerator](),
		di.AsSelf[MetricsManager](),
		di.AsSelf[AnalyticsManager](),
		di.AsSelf[InventoryManager](),
		di.AsSelf[IntelligentTieringManager](),
		di.AsSelf[OwnershipManager](),
		di.AsSelf[RequestPaymentManager](),
		di.AsSelf[Locator](),
		di.AsSelf[ObjectLockManager](),
		di.AsSelf[LegalHoldManager](),
		di.AsSelf[RetentionManager](),
		di.AsSelf[Restorer](),
		di.AsSelf[Selector](),
		di.AsSelf[ObjectAttributesGetter](),
		di.AsSelf[TorrentGetter](),
		di.AsSelf[ObjectLambdaWriter](),
	}
}
