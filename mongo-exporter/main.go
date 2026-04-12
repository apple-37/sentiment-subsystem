package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"mongo-exporter/internal/config"
	"mongo-exporter/internal/exporter"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	// configPath :="./config.yaml"
	runOnce := flag.Bool("once", false, "run export once and exit")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	exp, err := exporter.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create exporter", "error", err)
		os.Exit(1)
	}
	defer exp.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *runOnce {
		if err := exp.Export(ctx); err != nil {
			logger.Error("export failed", "error", err)
			os.Exit(1)
		}
		logger.Info("export completed")
		return
	}

	logger.Info("starting scheduler", "interval", cfg.Scheduler.Interval)
	exp.RunScheduled(ctx)
	logger.Info("shutdown complete")
}
