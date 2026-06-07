package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/dotslash-flame/flame-chess/internal/config"
	"github.com/dotslash-flame/flame-chess/internal/httpapi"
	"github.com/dotslash-flame/flame-chess/internal/hub"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config making error bro: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	h := hub.New(hub.Options{})
	go h.Run(ctx.Done())

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpapi.NewRouter(h, cfg.SessionHMACSecret, cfg.DevLogin),
	}

	go func() {
		log.Printf("flamechess listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}
