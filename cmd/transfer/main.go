package main

import (
	"flag"
	"fmt"
	"os"

	"ai-transfer-tg-to-vk/internal/config"
	"ai-transfer-tg-to-vk/internal/logger"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file (default: search in default locations)")
	generateConfig := flag.String("generate-config", "", "Generate example configuration file at the specified path")
	flag.Parse()

	// Generate config if requested
	if *generateConfig != "" {
		if err := generateExampleConfig(*generateConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate example config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Example configuration generated at: %s\n", *generateConfig)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Convert logging config
	logConfig := logger.Config{
		Level:             cfg.Logging.Level,
		Encoding:          cfg.Logging.Encoding,
		OutputPath:        cfg.Logging.OutputPath,
		MaxSize:           cfg.Logging.MaxSize,
		MaxBackups:        cfg.Logging.MaxBackups,
		MaxAge:            cfg.Logging.MaxAge,
		IncludeCaller:     cfg.Logging.IncludeCaller,
		IncludeStacktrace: cfg.Logging.IncludeStacktrace,
	}

	// Initialize logger
	log, err := logger.NewLogger(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Replace global logger
	zap.ReplaceGlobals(log)

	log.Info("Application started",
		zap.String("config", *configPath),
		zap.Bool("dry_run", cfg.Transfer.DryRun),
	)

	// TODO: Implement actual transfer logic
	log.Info("Transfer logic not yet implemented")

	log.Info("Application finished")
}

func loadConfig(configPath string) (*config.Config, error) {
	loader := config.NewLoader()
	cfg, err := loader.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func generateExampleConfig(outputPath string) error {
	loader := config.NewLoader()
	return loader.GenerateExampleConfig(outputPath)
}