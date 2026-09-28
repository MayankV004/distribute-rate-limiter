package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/streamliner/rate-limiter/internal/config"
)

// Run starts the main API gateway server and an internal metrics server.
// It sets up graceful shutdown on SIGINT/SIGTERM and exposes /healthz and /readyz endpoints.
func Run(ctx context.Context, cfg config.ServerConfig, mainHandler, metricsHandler http.Handler) error {
	// Add health probes to the main handler
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// In a real system, you might check Redis connectivity here
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.Handle("/", mainHandler)

	addr := cfg.Addr
	if addr == "" {
		addr = ":8080"
	}
	
	mainSrv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	var metricsSrv *http.Server
	if cfg.MetricsAddr != "" && metricsHandler != nil {
		metricsSrv = &http.Server{
			Addr:    cfg.MetricsAddr,
			Handler: metricsHandler,
		}
	}

	// Channel to listen for errors coming from the listeners
	serverErrors := make(chan error, 2)

	go func() {
		slog.Info("Starting main server", "addr", mainSrv.Addr)
		if err := mainSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	if metricsSrv != nil {
		go func() {
			slog.Info("Starting metrics server", "addr", metricsSrv.Addr)
			if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErrors <- err
			}
		}()
	}

	// Trap SIGINT and SIGTERM
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until a signal is received or an error occurs
	select {
	case err := <-serverErrors:
		return err
	case sig := <-shutdown:
		slog.Info("Shutdown signal received", "signal", sig)
		
		// Create a context with a timeout for the shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // default grace period
		defer cancel()

		var err error
		if metricsSrv != nil {
			if mErr := metricsSrv.Shutdown(shutdownCtx); mErr != nil {
				slog.Error("Metrics server shutdown failed", "error", mErr)
				err = mErr
			}
		}

		if mErr := mainSrv.Shutdown(shutdownCtx); mErr != nil {
			slog.Error("Main server shutdown failed", "error", mErr)
			err = mErr
		}

		slog.Info("Graceful shutdown complete")
		return err
	}
}
