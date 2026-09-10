package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/atlas-engine/atlas/internal/config"
	"github.com/atlas-engine/atlas/internal/logger"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format, cfg.Log.Output)
	log.Info("starting atlas agent", slog.String("node_id", cfg.Raft.NodeID))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("agent shutting down")
}
