package database

type Config struct {
	Dialect    string `mapstructure:"dialect"`
	Migrate    bool   `mapstructure:"migrate"`
	Datasource string `mapstructure:"datasource"`
}
