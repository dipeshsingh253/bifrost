package main

import (
	"log"
	"time"

	"github.com/dipesh/bifrost/agent/internal/client"
	"github.com/dipesh/bifrost/agent/internal/collector"
	"github.com/dipesh/bifrost/agent/internal/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	httpClient := client.New(cfg.BackendURL, cfg.APIKey)
	hostCollector := collector.NewHostCollector()
	dockerCollector := collector.NewDockerCollector(cfg)

	log.Printf("bifrost agent started for server=%s backend=%s", cfg.ServerName, cfg.BackendURL)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		if err := runOnce(cfg, httpClient, hostCollector, dockerCollector); err != nil {
			log.Printf("collect/push failed: %v", err)
		}
		<-ticker.C
	}
}

func runOnce(
	cfg config.Config,
	httpClient *client.Client,
	hostCollector *collector.HostCollector,
	dockerCollector *collector.DockerCollector,
) error {
	serverSnapshot, metrics, err := hostCollector.Collect(cfg)
	if err != nil {
		return err
	}

	serviceSnapshots, logPayloads := dockerCollector.Collect()
	serverSnapshot.Services = serviceSnapshots

	return httpClient.PushSnapshot(serverSnapshot, metrics, logPayloads)
}
