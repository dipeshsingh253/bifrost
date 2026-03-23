package main

import (
	"log"
	"os"
	"time"

	"github.com/dipesh/bifrost/agent/internal/client"
	"github.com/dipesh/bifrost/agent/internal/collector"
	"github.com/dipesh/bifrost/agent/internal/config"
)

func main() {
	configPath := os.Getenv("BIFROST_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	if err := ensureEnrollment(configPath, &cfg); err != nil {
		log.Fatalf("enroll agent: %v", err)
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

	return httpClient.PushSnapshot(cfg.AgentID, serverSnapshot, metrics, logPayloads)
}

func ensureEnrollment(configPath string, cfg *config.Config) error {
	if cfg.APIKey != "" {
		return nil
	}
	if cfg.EnrollmentToken == "" {
		return nil
	}

	bootstrapClient := client.New(cfg.BackendURL, cfg.EnrollmentToken)
	apiKey, err := bootstrapClient.Enroll(cfg.AgentID, cfg.ServerID)
	if err != nil {
		return err
	}

	cfg.APIKey = apiKey
	cfg.EnrollmentToken = ""
	return config.Save(configPath, *cfg)
}
