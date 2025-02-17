package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
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
	conf, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		return
	}
	web.SetServerLocation(conf)
	results.Initialize(conf)
	err = database.SetDBInfo(conf)
	if err != nil {
		slog.Error("init db", slog.Any("error", err))
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	stopWait, closeFn := onceChan[struct{}]()
	go func() {
		err := web.ListenAndServe(ctx, conf)
		if err != nil {
			slog.Error("web server", slog.Any("error", err))
		}
		closeFn()
	}()
	go wait(closeFn)
	<-stopWait
	slog.Info("server stopped")
	cancel()
}

func wait(closeFn func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	slog.Info("signal received", "signal", sig.String())
	closeFn()
}
func onceChan[T any]() (chan T, func()) {
	var once sync.Once
	var c = make(chan T)
	return c, func() {
		once.Do(func() {
			close(c)
		})
	}
}
