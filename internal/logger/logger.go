package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logging configuration
type Config struct {
	Level      string `mapstructure:"level"`
	Encoding   string `mapstructure:"encoding"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	IncludeCaller bool `mapstructure:"include_caller"`
	IncludeStacktrace bool `mapstructure:"include_stacktrace"`
}

// NewLogger creates a new zap logger based on configuration
func NewLogger(config Config) (*zap.Logger, error) {
	// Set log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}
	
	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// Create encoder
	var encoder zapcore.Encoder
	if config.Encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	
	// Create write syncer
	var writeSyncer zapcore.WriteSyncer
	if config.OutputPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
		
		// Create rotating file writer
		rotator := &lumberjack.Logger{
			Filename:   config.OutputPath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   true,
		}
		writeSyncer = zapcore.AddSync(rotator)
		
		// Also write to stdout for development
		writeSyncer = zapcore.NewMultiWriteSyncer(
			writeSyncer,
			zapcore.AddSync(os.Stdout),
		)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	
	// Create core
	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		level,
	)
	
	// Create logger options
	options := []zap.Option{}
	if config.IncludeCaller {
		options = append(options, zap.AddCaller())
	}
	if config.IncludeStacktrace {
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}
	
	// Create logger
	logger := zap.New(core, options...)
	
	return logger, nil
}

// NewDefaultConfig returns default logging configuration
func NewDefaultConfig() Config {
	return Config{
		Level:      "info",
		Encoding:   "json",
		OutputPath: "",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		IncludeCaller: true,
		IncludeStacktrace: true,
	}
}

// ContextLogger is a wrapper around zap.Logger with contextual fields
type ContextLogger struct {
	*zap.Logger
	fields []zap.Field
}

// With creates a new ContextLogger with additional fields
func (cl *ContextLogger) With(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{
		Logger: cl.Logger.With(fields...),
		fields: append(cl.fields[:len(cl.fields):len(cl.fields)], fields...),
	}
}

// Operation creates a new ContextLogger with operation field
func (cl *ContextLogger) Operation(name string) *ContextLogger {
	return cl.With(zap.String("operation", name))
}

// Resource creates a new ContextLogger with resource field
func (cl *ContextLogger) Resource(id string) *ContextLogger {
	return cl.With(zap.String("resource", id))
}

// Component creates a new ContextLogger with component field
func (cl *ContextLogger) Component(name string) *ContextLogger {
	return cl.With(zap.String("component", name))
}

// NewContextLogger creates a new ContextLogger from a zap.Logger
func NewContextLogger(logger *zap.Logger) *ContextLogger {
	return &ContextLogger{
		Logger: logger,
		fields: []zap.Field{},
	}
}