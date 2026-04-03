package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/dipesh/bifrost/backend/config"
	"github.com/dipesh/bifrost/backend/internal/router"
	"github.com/dipesh/bifrost/backend/internal/seed"
	shareddb "github.com/dipesh/bifrost/backend/internal/shared/database"
	"github.com/dipesh/bifrost/backend/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := shareddb.Open(cfg.DatabaseURL, shareddb.Options{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	dataStore := store.NewPostgresStoreFromDB(db)

	if cfg.SeedDemoData {
		if err := dataStore.EnsureSeedData(seed.Data()); err != nil {
			log.Fatalf("seed postgres: %v", err)
		}
	}

	engine := router.New(cfg, dataStore)
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           engine,
		ReadHeaderTimeout: cfg.ServerReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
		MaxHeaderBytes:    cfg.ServerMaxHeaderBytes,
	}

	log.Printf("bifrost backend listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
