package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"buildestate/backend/internal/access/policy"
	"buildestate/backend/internal/access/sessions"
	"buildestate/backend/internal/contracts"
	"buildestate/backend/internal/httpapi"
	"buildestate/backend/internal/platform/clock"
	"buildestate/backend/internal/platform/config"
	"buildestate/backend/internal/platform/database"
	"buildestate/backend/internal/platform/storage"
)

func run(logger *slog.Logger) error {
	c, err := config.Load()
	if err != nil {
		return errors.New("invalid application configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, c.SQLitePath)
	if err != nil {
		return errors.New("database startup failed")
	}
	defer db.Close()
	if err = database.Migrate(ctx, db); err != nil {
		return errors.New("database migration failed")
	}
	readers, err := database.OpenReaders(ctx, c.SQLitePath)
	if err != nil {
		return errors.New("database reader startup failed")
	}
	defer readers.Close()
	if err = os.MkdirAll(c.AttachmentsDir, 0700); err != nil {
		return errors.New("attachment storage unavailable")
	}
	if err = storage.Check(c.AttachmentsDir); err != nil {
		return errors.New("attachment storage unavailable")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", c.Port))
	if err != nil {
		return errors.New("listener startup failed")
	}
	key, err := sessions.LoadKey(c.DataDir)
	if err != nil {
		listener.Close()
		return errors.New("session key initialization failed")
	}
	server := &http.Server{Handler: httpapi.New(contracts.Dependencies{DB: db, ReadDB: readers, Clock: clock.UTC{}, SessionKey: key, CookieSecure: c.CookieSecure, Policy: policy.Service{DB: db, Clock: clock.UTC{}}}, c.StaticDir, c.AttachmentsDir, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 120 * time.Second, WriteTimeout: 125 * time.Second, IdleTimeout: 60 * time.Second}
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	exited := make(chan error, 1)
	go func() { exited <- server.Serve(listener) }()
	logger.Info("server ready", "address", listener.Addr().String())
	select {
	case err = <-exited:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("HTTP service failed")
		}
	case <-signals.Done():
		shutdown, done := context.WithTimeout(context.Background(), 15*time.Second)
		defer done()
		if err = server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return errors.New("graceful shutdown timed out")
		}
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("startup or service failure", "message", err.Error())
		os.Exit(1)
	}
}
