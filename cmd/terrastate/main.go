package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/terrastate/internal/store"
	_ "github.com/the-maldridge/terrastate/internal/store/bc"

	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"

	_ "github.com/the-maldridge/terrastate/internal/web/auth/file"
	_ "github.com/the-maldridge/terrastate/internal/web/auth/netauth"
)

func main() {
	ll := os.Getenv("TS_LOG_LEVEL")
	if ll == "" {
		ll = "INFO"
	}
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "terrastate",
		Level: hclog.LevelFromString(ll),
	})

	store.SetLogger(appLogger.Named("store"))
	store.DoCallbacks()

	auth.SetLogger(appLogger.Named("web").Named("auth"))
	auth.DoCallbacks()

	si := os.Getenv("TS_STORE")
	if si == "" {
		si = "bitcask"
	}
	s, err := store.Initialize(si)
	if err != nil {
		appLogger.Error("Could not initialize store", "error", err)
		os.Exit(2)
	}

	ai := os.Getenv("TS_AUTH")
	if ai == "" {
		ai = "file"
	}
	a, err := auth.Initialize(ai)
	if err != nil {
		appLogger.Error("Could not initialize auth", "error", err)
		os.Exit(2)
	}

	w, err := web.New(
		web.WithLogger(appLogger),
		web.WithStore(s),
		web.WithAuth(a),
	)
	if err != nil {
		appLogger.Error("Error initializing webserver", "error", err)
	}

	bind := os.Getenv("TS_BIND")
	if bind == "" {
		bind = ":3030"
	}

	go func() {
		w.Serve(bind)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.Shutdown(ctx); err != nil {
		appLogger.Error("Error during shutdown", "error", err)
		os.Exit(2)
	}
	if err := s.Close(); err != nil {
		appLogger.Error("Error releasing store connection", "error", err)
		os.Exit(2)
	}
	appLogger.Info("Goodbye!")
}
