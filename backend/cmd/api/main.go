package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"campus-market/backend/internal/app"
	"campus-market/backend/internal/db"
	"campus-market/backend/internal/httpapi"
)

func main() {
	cfg := app.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()
	if err := store.RunMigrations(ctx, cfg.MigrationsDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(cfg, store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
