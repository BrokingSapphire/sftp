package logger

// LoggingConfig configures the logger and its sinks.
type LoggingConfig struct {
	Level            string       `mapstructure:"level"              validate:"required,oneof=debug info warn error fatal" default:"info"`
	Format           string       `mapstructure:"format"             validate:"required,oneof=json console"                default:"json"`
	EnableStackTrace bool         `mapstructure:"enable_stack_trace" default:"false"`
	EnableCaller     bool         `mapstructure:"enable_caller"      default:"false"`
	Sinks            []SinkConfig `mapstructure:"sinks"`
}

// SinkConfig configures a single log output.
type SinkConfig struct {
	Type  string `mapstructure:"type"  validate:"required,oneof=stdout file"`
	Level string `mapstructure:"level" validate:"omitempty,oneof=debug info warn error fatal"`

	// File sink fields.
	Path       string `mapstructure:"path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"  default:"100"`
	MaxBackups int    `mapstructure:"max_backups"  default:"7"`
	MaxAgeDays int    `mapstructure:"max_age_days" default:"90"`
	Compress   bool   `mapstructure:"compress"     default:"true"`
}
