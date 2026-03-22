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
	dataStore := store.NewMemoryStore(seed.Data())

	router := httpapi.NewRouter(cfg, dataStore)

	log.Printf("bifrost backend listening on %s", cfg.ListenAddr)
	if err := router.Run(cfg.ListenAddr); err != nil {
		log.Fatal(err)
	}
}
