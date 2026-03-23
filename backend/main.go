package main

import (
	"log"

	"github.com/dipesh/bifrost/backend/internal/config"
	"github.com/dipesh/bifrost/backend/internal/httpapi"
	"github.com/dipesh/bifrost/backend/internal/seed"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func main() {
	cfg := config.Load()
	dataStore, err := store.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer dataStore.Close()

	if cfg.SeedDemoData {
		if err := dataStore.EnsureSeedData(seed.Data()); err != nil {
			log.Fatalf("seed postgres: %v", err)
		}
	}

	router := httpapi.NewRouter(cfg, dataStore)

	log.Printf("bifrost backend listening on %s", cfg.ListenAddr)
	if err := router.Run(cfg.ListenAddr); err != nil {
		log.Fatal(err)
	}
}
