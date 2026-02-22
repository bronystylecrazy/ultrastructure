package xgorm

import "time"

type LogConfig struct {
	Enabled                   *bool         `mapstructure:"enabled"`
	LogLevel                  string        `mapstructure:"level" default:"warn"`
	SlowThreshold             time.Duration `mapstructure:"slow_threshold" default:"100ms"`
	SkipCallerLookup          bool          `mapstructure:"skip_caller_lookup" default:"false"`
	IgnoreRecordNotFoundError bool          `mapstructure:"ignore_record_not_found_error" default:"false"`
	ParameterizedQueries      bool          `mapstructure:"parameterized_queries" default:"true"`
}

type TraceConfig struct {
	Enabled               *bool             `mapstructure:"enabled"`
	DBName                string            `mapstructure:"db_name"`
	Attributes            map[string]string `mapstructure:"attributes"`
	WithoutQueryVariables bool              `mapstructure:"without_query_variables" default:"true"`
	WithoutMetrics        bool              `mapstructure:"without_metrics" default:"false"`
	IncludeDryRunSpans    bool              `mapstructure:"include_dry_run_spans" default:"false"`
}
