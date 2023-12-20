package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/terrastate/internal/store"
	_ "github.com/the-maldridge/terrastate/internal/store/bc"

	"github.com/the-maldridge/terrastate/internal/web"
	"github.com/the-maldridge/terrastate/internal/web/auth"

	_ "github.com/the-maldridge/terrastate/internal/web/auth/file"
	_ "github.com/the-maldridge/terrastate/internal/web/auth/htpasswd"
	_ "github.com/the-maldridge/terrastate/internal/web/auth/ldap"
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
	authsList := strings.Split(ai, ":")
	if len(authsList) == 0 {
		authsList = []string{"file"}
	}
	auths := []web.Option{}
	for _, mech := range authsList {
		a, err := auth.Initialize(mech)
		if err != nil {
			appLogger.Error("Could not initialize auth", "error", err)
			os.Exit(2)
		}
		auths = append(auths, web.WithAuth(a))
	}

	webOpts := append(auths, web.WithLogger(appLogger), web.WithStore(s))

	w, err := web.New(webOpts...)
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
