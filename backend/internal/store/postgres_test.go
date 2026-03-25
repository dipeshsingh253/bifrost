package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dipesh/bifrost/backend/internal/domain"
)

func TestPostgresSelfEnrollPendingAgentBeforeFirstServerIngest(t *testing.T) {
	store, rawDB, cleanup := newPostgresTestStore(t)
	defer cleanup()

	admin, err := store.BootstrapAdmin("Bifrost Test", "Admin User", "admin@example.com", "password123")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	onboarding, err := store.CreateSystemOnboarding(admin.TenantID, admin.ID, "Validation VPS", "Pending enrollment")
	if err != nil {
		t.Fatalf("create system onboarding: %v", err)
	}

	var serverCount int
	if err := rawDB.QueryRow(`SELECT COUNT(*) FROM servers WHERE id = $1`, onboarding.ServerID).Scan(&serverCount); err != nil {
		t.Fatalf("query server count: %v", err)
	}
	if serverCount != 0 {
		t.Fatalf("expected no server row before first ingest, found %d", serverCount)
	}

	enrolledAgent, err := store.SelfEnrollPendingAgent(onboarding.AgentID, onboarding.ServerID)
	if err != nil {
		t.Fatalf("self-enroll pending agent: %v", err)
	}
	if enrolledAgent.APIKey == "" || enrolledAgent.APIKey == onboarding.APIKey {
		t.Fatalf("expected self-enrollment to rotate the bootstrap api key")
	}
	if enrolledAgent.Version != "enrolled" {
		t.Fatalf("expected agent version to be enrolled, got %q", enrolledAgent.Version)
	}
	if enrolledAgent.ServerID != onboarding.ServerID {
		t.Fatalf("expected enrolled agent to retain onboarding server id in memory, got %q", enrolledAgent.ServerID)
	}

	lookedUpAgent, err := store.AgentByAPIKey(enrolledAgent.APIKey)
	if err != nil {
		t.Fatalf("look up enrolled agent by rotated key: %v", err)
	}
	if lookedUpAgent.ID != onboarding.AgentID {
		t.Fatalf("expected rotated key to resolve enrolled agent %q, got %q", onboarding.AgentID, lookedUpAgent.ID)
	}
	if lookedUpAgent.Version != "enrolled" {
		t.Fatalf("expected persisted enrolled agent version, got %q", lookedUpAgent.Version)
	}
	if lookedUpAgent.ServerID != "" {
		t.Fatalf("expected persisted agent to remain detached from server_id before first ingest, got %q", lookedUpAgent.ServerID)
	}

	if _, err := store.AgentByAPIKey(onboarding.APIKey); err != ErrNotFound {
		t.Fatalf("expected bootstrap key lookup to fail after enrollment, got %v", err)
	}

	var persistedServerID sql.NullString
	if err := rawDB.QueryRow(`SELECT server_id FROM agents WHERE id = $1`, onboarding.AgentID).Scan(&persistedServerID); err != nil {
		t.Fatalf("query persisted agent server_id: %v", err)
	}
	if persistedServerID.Valid {
		t.Fatalf("expected persisted agent server_id to stay NULL before ingest, got %q", persistedServerID.String)
	}
}

func TestPostgresIngestRewritesLegacyRuntimeMonitoringIDsToUUIDs(t *testing.T) {
	store, _, cleanup := newPostgresTestStore(t)
	defer cleanup()

	admin, err := store.BootstrapAdmin("Bifrost Test", "Admin User", "admin@example.com", "password123")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	const serverID = "9329fc4b-4d0d-4d3a-8fcb-53f87d7d9e0d"
	const agentID = "5b8b7684-f4e3-44a9-8fc6-a58a6110b541"
	const legacyServiceID = "svc-zhiro"
	const legacyContainerID = "2263a588fa73c3b8e1efda820b74a0374dd872bd2cca887045b40e6e1b10c35d"

	if _, err := store.EnrollAgent(domain.Agent{
		ID:         agentID,
		TenantID:   admin.TenantID,
		Name:       "uuid-agent",
		APIKey:     "uuid-agent-key",
		Version:    "0.1.0",
		ServerID:   serverID,
		ServerName: "dev-server",
		Hostname:   "ubuntu-bifrost",
	}); err != nil {
		t.Fatalf("enroll agent: %v", err)
	}

	if err := store.Ingest(domain.IngestPayload{
		AgentID: agentID,
		Server: domain.ServerSnapshot{
			ID:             serverID,
			Name:           "dev-server",
			Hostname:       "ubuntu-bifrost",
			PublicIP:       "203.0.113.42",
			AgentVersion:   "0.1.0",
			Status:         "up",
			UptimeSeconds:  100,
			CPUUsagePct:    10,
			MemoryUsagePct: 20,
			DiskUsagePct:   30,
			NetworkRXMB:    1,
			NetworkTXMB:    1,
			LoadAverage:    "0.10 0.10 0.10",
			OS:             "Ubuntu",
			Kernel:         "6.8.0",
			CPUModel:       "AMD EPYC",
			CPUCores:       4,
			CPUThreads:     8,
			TotalMemoryGB:  16,
			TotalDiskGB:    256,
			CollectedAt:    time.Date(2026, time.March, 23, 6, 0, 0, 0, time.UTC),
			Services: []domain.ServiceSnapshot{
				{
					ID:             legacyServiceID,
					Name:           "zhiro",
					ComposeProject: "zhiro",
					Status:         "running",
					PublishedPorts: []string{"8000:8000"},
					Containers: []domain.ContainerSnapshot{
						{
							ID:           legacyContainerID,
							Name:         "zhiro-app",
							Image:        "zhiro-app",
							Status:       "running",
							Health:       "healthy",
							CPUUsagePct:  0.15,
							MemoryMB:     92.46,
							NetworkMB:    0.03,
							RestartCount: 0,
							Uptime:       "1h",
							Ports:        []string{"8000:8000"},
							Command:      "uvicorn src.main:app --host 0.0.0.0 --port 8000 --workers 1",
							LastSeenAt:   time.Date(2026, time.March, 23, 6, 0, 0, 0, time.UTC),
						},
					},
				},
			},
		},
		Logs: []domain.LogPayload{
			{
				ServerID:    serverID,
				ServiceID:   legacyServiceID,
				ContainerID: legacyContainerID,
				Level:       "info",
				Message:     "app boot complete",
				Timestamp:   time.Date(2026, time.March, 23, 6, 0, 0, 0, time.UTC),
			},
		},
	}); err != nil {
		t.Fatalf("ingest snapshot: %v", err)
	}

	bundle, err := store.ServerBundle(admin.TenantID, serverID)
	if err != nil {
		t.Fatalf("server bundle: %v", err)
	}
	if len(bundle.Services) != 1 {
		t.Fatalf("expected one ingested service, got %d", len(bundle.Services))
	}

	service := bundle.Services[0]
	if service.ID == legacyServiceID || !isUUIDString(service.ID) {
		t.Fatalf("expected canonical uuid service id instead of %q", service.ID)
	}
	if len(service.Containers) != 1 {
		t.Fatalf("expected one ingested container, got %d", len(service.Containers))
	}

	container := service.Containers[0]
	if container.ID == legacyContainerID || !isUUIDString(container.ID) {
		t.Fatalf("expected canonical uuid container id instead of %q", container.ID)
	}

	if len(bundle.ContainerMetrics.CPU) == 0 {
		t.Fatalf("expected container metrics to be recorded")
	}
	lastMetricPoint := bundle.ContainerMetrics.CPU[len(bundle.ContainerMetrics.CPU)-1].Values
	if _, ok := lastMetricPoint[container.ID]; !ok {
		t.Fatalf("expected canonical container id %q in metrics", container.ID)
	}
	if _, ok := lastMetricPoint[legacyContainerID]; ok {
		t.Fatalf("expected raw docker container id %q to be absent from metrics", legacyContainerID)
	}

	logs := store.LogsByService(service.ID)
	if len(logs) != 1 {
		t.Fatalf("expected one remapped log line, got %d", len(logs))
	}
	if logs[0].ServiceID != service.ID || logs[0].ContainerID != container.ID {
		t.Fatalf("expected remapped canonical log ids, got %+v", logs[0])
	}
}

func newPostgresTestStore(t *testing.T) (*PostgresStore, *sql.DB, func()) {
	t.Helper()

	adminURL := postgresTestAdminURL()
	adminDB, err := sql.Open("pgx", adminURL)
	if err != nil {
		t.Fatalf("open postgres admin database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := adminDB.PingContext(ctx); err != nil {
		_ = adminDB.Close()
		t.Skipf("skipping postgres regression test: %v", err)
	}

	databaseName := fmt.Sprintf("bifrost_store_test_%d", time.Now().UTC().UnixNano())
	if _, err := adminDB.ExecContext(ctx, `CREATE DATABASE `+quotePostgresIdentifier(databaseName)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create test database: %v", err)
	}

	testURL := postgresDatabaseURL(adminURL, databaseName)
	store, err := NewPostgresStore(testURL)
	if err != nil {
		_, _ = adminDB.ExecContext(context.Background(), `DROP DATABASE IF EXISTS `+quotePostgresIdentifier(databaseName))
		_ = adminDB.Close()
		t.Fatalf("open test store: %v", err)
	}

	if err := applyPostgresTestMigrations(store.db); err != nil {
		_ = store.Close()
		_, _ = adminDB.ExecContext(context.Background(), `DROP DATABASE IF EXISTS `+quotePostgresIdentifier(databaseName))
		_ = adminDB.Close()
		t.Fatalf("apply test migrations: %v", err)
	}

	cleanup := func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close postgres test store: %v", err)
		}
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		if _, err := adminDB.ExecContext(dropCtx, `DROP DATABASE IF EXISTS `+quotePostgresIdentifier(databaseName)); err != nil {
			t.Fatalf("drop postgres test database: %v", err)
		}
		if err := adminDB.Close(); err != nil {
			t.Fatalf("close postgres admin database: %v", err)
		}
	}

	return store, store.db, cleanup
}

func postgresTestAdminURL() string {
	if value := strings.TrimSpace(os.Getenv("BIFROST_TEST_DATABASE_URL")); value != "" {
		return value
	}
	return "postgres://bifrost:bifrost@127.0.0.1:5433/postgres?sslmode=disable"
}

func postgresDatabaseURL(baseURL, databaseName string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}
	parsed.Path = "/" + databaseName
	return parsed.String()
}

func quotePostgresIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func applyPostgresTestMigrations(db *sql.DB) error {
	migrationsDir, err := postgresMigrationsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			return err
		}
		if len(content) == 0 {
			continue
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
	}

	return nil
}

func postgresMigrationsDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "migrations")), nil
}
