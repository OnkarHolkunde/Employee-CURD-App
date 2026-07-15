// Command server is the entry point for the Excel-Import CRUD API.
// It wires together configuration, MySQL, Redis, services, handlers, and
// the HTTP router, then serves with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"excel-crud-app/internal/config"
	"excel-crud-app/internal/database"
	"excel-crud-app/internal/handlers"
	"excel-crud-app/internal/routes"
	"excel-crud-app/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {

	// Logger instance created
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	// MySQL and Redis connections
	if err := database.ConnectMySQL(cfg); err != nil {
		slog.Error("MySQL connection failed", "error", err)
		os.Exit(1)
	}

	if err := database.RunMigrations(); err != nil {
	slog.Error("Migration failed", "error", err)
	os.Exit(1)
	}

	if err := database.ConnectRedis(cfg); err != nil {
		slog.Error("Redis connection failed", "error", err)
		os.Exit(1)
	}

	// Services
	employeeSvc := services.NewEmployeeService(cfg.CacheTTLSeconds)
	jobStore := services.NewJobStore()

	uploadHandler := handlers.NewUploadHandler(employeeSvc, jobStore, cfg.UploadDir)
	employeeHandler := handlers.NewEmployeeHandler(employeeSvc)

	router := routes.NewRouter(cfg.AllowedOrigin, uploadHandler, employeeHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run the server in a goroutine so the main goroutine is free to wait
	// for an OS signal and drive graceful shutdown.
	go func() {
		slog.Info("server starting", "addr", srv.Addr, "gin_mode", cfg.GinMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received, draining in-flight requests")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed, forcing exit", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped cleanly")
}
