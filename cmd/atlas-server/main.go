package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/atlas-engine/atlas/internal/ai"
	"github.com/atlas-engine/atlas/internal/api"
	"github.com/atlas-engine/atlas/internal/anomaly"
	"github.com/atlas-engine/atlas/internal/benchmarks"
	"github.com/atlas-engine/atlas/internal/chaos"
	"github.com/atlas-engine/atlas/internal/config"
	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/incident"
	"github.com/atlas-engine/atlas/internal/jobqueue"
	"github.com/atlas-engine/atlas/internal/logger"
	"github.com/atlas-engine/atlas/internal/metrics"
	"github.com/atlas-engine/atlas/internal/topology"
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
	log.Info("starting atlas", slog.String("version", "0.1.0"), slog.String("addr", cfg.Server.Addr()))

	bus := event.NewBus(1000)
	bus.Start()

	topologyEngine := topology.NewEngine(bus)
	queue := jobqueue.NewQueue(jobqueue.Options{
		Logger:   log,
		EventBus: bus,
	})
	chaosEngine := chaos.NewEngine(topologyEngine, bus, log)
	anomalyEngine := anomaly.NewDetector(anomaly.Options{
		Bus:    bus,
		Logger: log,
	})
	anomalyEngine.AddRule(anomaly.Rule{Metric: "cpu", CriticalHigh: 90})
	anomalyEngine.AddRule(anomaly.Rule{Metric: "memory_usage", CriticalHigh: 85})

	incidentManager := incident.NewManager(incident.Options{Bus: bus, Logger: log})
	incidentManager.Start()
	advisor := ai.NewAdvisor(3)

	metricsReg := metrics.NewRegistry()
	metrics.NewBusRecorder(bus, metricsReg)
	stopRuntime := make(chan struct{})
	go metrics.RunRuntimeCollector(metricsReg, 15*time.Second, stopRuntime)
	defer close(stopRuntime)

	benchRunner := benchmarks.NewRunner()

	srv := api.NewWithDeps(cfg, log, api.Dependencies{
		Topology: topologyEngine,
		Queue:    queue,
		Chaos:    chaosEngine,
		Anomaly:  anomalyEngine,
		Incident: incidentManager,
		AI:       advisor,
		Metrics:  metricsReg,
		Bench:    benchRunner,
	})
	httpServer := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      srv.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		log.Info("shutdown signal received")
	case err := <-errCh:
		log.Error("server error", slog.Any("error", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("shutdown error", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("atlas stopped")
}
