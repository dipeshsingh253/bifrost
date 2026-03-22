package collector

import (
	"testing"
	"time"
)

func TestBuildDockerRuntimeParsesInspectAndStats(t *testing.T) {
	row := dockerPSRow{
		ID:      "abc123",
		Image:   "ghcr.io/acme/service-a:latest",
		Names:   "service-a-backend-1",
		Status:  "Up 5 minutes",
		Command: "bin/api",
		Ports:   "0.0.0.0:8000->8000/tcp",
		Labels:  "com.docker.compose.project=service-a",
	}

	inspect := dockerInspectEntry{ID: "abc123", Name: "/service-a-backend-1", RestartCount: 2}
	inspect.Config.Image = "ghcr.io/acme/service-a@sha256:abc"
	inspect.Config.Labels = map[string]string{"com.docker.compose.project": "service-a"}
	inspect.State.Status = "running"
	inspect.State.StartedAt = time.Now().Add(-26 * time.Hour).UTC().Format(time.RFC3339Nano)
	inspect.State.Health = &struct {
		Status string `json:"Status"`
	}{Status: "healthy"}

	stats := dockerStatsRow{
		ID:       "abc123",
		CPUPerc:  "12.5%",
		MemUsage: "256MiB / 1GiB",
		NetIO:    "1.5MB / 512kB",
	}

	runtime := buildDockerRuntime(row, inspect, stats)
	if runtime.project != "service-a" {
		t.Fatalf("expected compose project service-a, got %q", runtime.project)
	}
	if runtime.status != "running" || runtime.health != "healthy" {
		t.Fatalf("expected running healthy runtime, got status=%q health=%q", runtime.status, runtime.health)
	}
	if runtime.cpuUsagePct != 12.5 {
		t.Fatalf("expected cpu usage 12.5, got %f", runtime.cpuUsagePct)
	}
	if runtime.memoryMB < 255 || runtime.memoryMB > 257 {
		t.Fatalf("expected memory around 256 MiB, got %f", runtime.memoryMB)
	}
	if runtime.networkMB < 1.99 || runtime.networkMB > 2.01 {
		t.Fatalf("expected total network around 2.0 MB, got %f", runtime.networkMB)
	}
	if runtime.restartCount != 2 {
		t.Fatalf("expected restart count 2, got %d", runtime.restartCount)
	}
}

func TestBuildServiceSnapshotsSeparatesProjectsAndStandalones(t *testing.T) {
	runtimes := []dockerRuntime{
		{id: "1", name: "service-a-backend-1", project: "service-a", status: "running", health: "healthy", ports: []string{"8000:8000"}},
		{id: "2", name: "service-a-worker-1", project: "service-a", status: "running", health: "healthy"},
		{id: "3", name: "edge-proxy-1", project: "", status: "running", health: "healthy", ports: []string{"8080:80"}},
	}

	services := buildServiceSnapshots(runtimes)
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0].ComposeProject != "" || services[0].Name != "edge-proxy-1" {
		t.Fatalf("expected standalone service first after sorting, got name=%q compose_project=%q", services[0].Name, services[0].ComposeProject)
	}
	if services[1].ComposeProject != "service-a" || len(services[1].Containers) != 2 {
		t.Fatalf("expected compose project service-a with 2 containers")
	}
}

func TestParseDockerLogOutputAndLevelInference(t *testing.T) {
	output := []byte("2026-03-22T10:00:00.000000000Z INFO started server\n2026-03-22T10:00:01.000000000Z WARN high memory usage\n")
	logs, latest := parseDockerLogOutput("srv-1", "svc-1", "ctr-1", output)

	if len(logs) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(logs))
	}
	if logs[0].Level != "info" || logs[1].Level != "warn" {
		t.Fatalf("expected inferred log levels info/warn, got %q/%q", logs[0].Level, logs[1].Level)
	}
	if latest.IsZero() {
		t.Fatalf("expected latest timestamp to be populated")
	}
}

func TestParseHelpers(t *testing.T) {
	if project := composeProject(parseLabelMap("com.docker.compose.project=service-a,foo=bar"), "service-a-backend-1"); project != "service-a" {
		t.Fatalf("expected compose project from labels, got %q", project)
	}
	if parsePercent("17.5%") != 17.5 {
		t.Fatalf("expected percent parsing to work")
	}
	if parseMemoryUsage("512MiB / 2GiB") < 511 || parseMemoryUsage("512MiB / 2GiB") > 513 {
		t.Fatalf("expected memory parsing around 512 MiB")
	}
	if parseNetIO("1.5MB / 512kB") < 1.99 || parseNetIO("1.5MB / 512kB") > 2.01 {
		t.Fatalf("expected net io parsing around 2.0 MB")
	}
	if inferLogLevel("ERROR failed to connect") != "error" {
		t.Fatalf("expected error log level inference")
	}
}
