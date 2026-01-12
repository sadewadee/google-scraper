package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sadewadee/google-scraper/internal/proxygate"
	"github.com/sadewadee/google-scraper/runner"
	"github.com/sadewadee/google-scraper/runner/databaserunner"
	"github.com/sadewadee/google-scraper/runner/filerunner"
	"github.com/sadewadee/google-scraper/runner/installplaywright"
	"github.com/sadewadee/google-scraper/runner/lambdaaws"
	"github.com/sadewadee/google-scraper/runner/managerrunner"
	"github.com/sadewadee/google-scraper/runner/webrunner"
	"github.com/sadewadee/google-scraper/runner/workerrunner"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner.Banner()

	log.Println("Starting application...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan

		log.Println("Received signal, shutting down...")

		cancel()
	}()

	cfg := runner.ParseConfig()

	log.Printf("RunMode: %d (Manager=%v, Worker=%v, Web=%v)", cfg.RunMode, cfg.ManagerMode, cfg.WorkerMode, cfg.WebRunner)

	runnerInstance, err := runnerFactory(cfg)
	if err != nil {
		cancel()
		os.Stderr.WriteString(err.Error() + "\n")

		runner.Telemetry().Close()

		os.Exit(1)
	}

	egroup, ctx := errgroup.WithContext(ctx)

	// Start ProxyGate if enabled
	if cfg.ProxyGateEnabled {
		pgCfg := &proxygate.Config{
			Enabled:              true,
			ListenAddr:           cfg.ProxyGateAddr,
			SourceURLs:           cfg.ProxyGateSources,
			RefreshInterval:      cfg.ProxyGateRefreshInterval,
			ValidatorConcurrency: 50,
		}

		if len(pgCfg.SourceURLs) == 0 {
			pgCfg = proxygate.DefaultConfig()
			pgCfg.ListenAddr = cfg.ProxyGateAddr
		}

		pg := proxygate.New(pgCfg)

		egroup.Go(func() error {
			return pg.Run(ctx)
		})
	}

	egroup.Go(func() error {
		if err := runnerInstance.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	})

	if err := egroup.Wait(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		_ = runnerInstance.Close(ctx)
		runner.Telemetry().Close()
		os.Exit(1)
	}

	_ = runnerInstance.Close(ctx)
	runner.Telemetry().Close()

	os.Exit(0)
}

func runnerFactory(cfg *runner.Config) (runner.Runner, error) {
	switch cfg.RunMode {
	case runner.RunModeFile:
		return filerunner.New(cfg)
	case runner.RunModeDatabase, runner.RunModeDatabaseProduce:
		return databaserunner.New(cfg)
	case runner.RunModeInstallPlaywright:
		return installplaywright.New(cfg)
	case runner.RunModeWeb:
		return webrunner.New(cfg)
	case runner.RunModeAwsLambda:
		return lambdaaws.New(cfg)
	case runner.RunModeAwsLambdaInvoker:
		return lambdaaws.NewInvoker(cfg)
	case runner.RunModeManager:
		return managerrunner.New(&managerrunner.Config{
			DatabaseURL:  cfg.Dsn,
			Address:      cfg.Addr,
			DataFolder:   cfg.DataFolder,
			StaticFolder: cfg.StaticFolder,
		})
	case runner.RunModeWorker:
		return workerrunner.New(&workerrunner.Config{
			ManagerURL:   cfg.ManagerURL,
			WorkerID:     cfg.WorkerID,
			RunnerConfig: cfg,
		})
	default:
		return nil, fmt.Errorf("%w: %d", runner.ErrInvalidRunMode, cfg.RunMode)
	}
}
