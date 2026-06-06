package main

import (
	"log"
	"net/http"

	"github.com/dotslash-flame/flame-chess/internal/config"
	"github.com/dotslash-flame/flame-chess/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config making error bro: %v", err)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: httpapi.NewRouter(),
	}

	log.Printf("flamechess listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
