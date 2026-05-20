package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"ai-transfer-tg-to-vk/internal/config"
	"ai-transfer-tg-to-vk/internal/logger"
	"ai-transfer-tg-to-vk/internal/transfer"
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

	// Create transfer orchestrator
	transfer, err := transfer.NewTransfer(cfg, log)
	if err != nil {
		log.Error("Failed to create transfer orchestrator", zap.Error(err))
		os.Exit(1)
	}
	defer transfer.Close()

	// Run transfer with context
	ctx := context.Background()
	if err := transfer.Run(ctx); err != nil {
		log.Error("Transfer failed", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Application finished successfully")
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