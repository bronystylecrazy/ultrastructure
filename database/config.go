package database

type Config struct {
	Driver     string `mapstructure:"driver"`
	Migrate    bool   `mapstructure:"migrate"`
	Datasource string `mapstructure:"datasource"`
}
