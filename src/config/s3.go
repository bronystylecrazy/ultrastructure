package config

type S3Config struct {
	PublicEndpoint  string `mapstructure:"public_endpoint" yaml:"public_endpoint"`
	Endpoint        string `mapstructure:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key" yaml:"secret_access_key"`
	BucketName      string `mapstructure:"bucket_name" yaml:"bucket_name"`
	UseSSL          bool   `mapstructure:"use_ssl" yaml:"use_ssl"`
}
