package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/librespeed/speedtest/config"
	"github.com/librespeed/speedtest/database"
	"github.com/librespeed/speedtest/results"
	"github.com/librespeed/speedtest/web"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"

	_ "golang.org/x/crypto/x509roots/fallback"
)

var (
	optConfig = flag.String("c", "", "config file to be used, defaults to settings.toml in the same directory")
)

func init() {
	zerologLogger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).
		With().
		Caller().
		Logger()

	logger := slog.New(
		slogzerolog.Option{
			Level:  slog.LevelInfo,
			Logger: &zerologLogger,
		}.
			NewZerologHandler(),
	)
	slog.SetDefault(logger)
}

func main() {
	flag.Parse()
	conf := config.Load(*optConfig)
	web.SetServerLocation(&conf)
	results.Initialize(&conf)
	err := database.SetDBInfo(&conf)
	if err != nil {
		slog.Error("init db", slog.Any("error", err))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	stopWait := make(chan struct{})
	go func() {
		err := web.ListenAndServe(ctx, &conf)
		if err != nil {
			slog.Error("web server", slog.Any("error", err))
		}
		close(stopWait)
	}()
	wait()
	slog.Info("server stopped")
	cancel()
	<-stopWait
}

func wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	slog.Info("signal received", "signal", sig)
}
