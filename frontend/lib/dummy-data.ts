import type { ServerBundle, MetricPoint, ContainerMetricPoint, LogLine, EventLog, Service } from "./types";

// ── Seeded PRNG (deterministic to avoid hydration mismatches) ──

let _seed = 42;
function seededRandom(): number {
  _seed = (_seed * 16807 + 0) % 2147483647;
  return (_seed - 1) / 2147483646;
}

function randomBetween(min: number, max: number): number {
  return Math.round((seededRandom() * (max - min) + min) * 100) / 100;
}

function generateTimeSeries(
  count: number,
  baseFn: (i: number) => number,
  intervalMinutes = 1
): MetricPoint[] {
  const now = Date.now();
  return Array.from({ length: count }, (_, i) => ({
    timestamp: new Date(now - (count - 1 - i) * intervalMinutes * 60_000).toISOString(),
    value: baseFn(i),
  }));
}

function generateContainerTimeSeries(
  count: number,
  containerNames: string[],
  baseFn: (name: string, i: number) => number,
  intervalMinutes = 1
): ContainerMetricPoint[] {
  const now = Date.now();
  return Array.from({ length: count }, (_, i) => {
    const point: ContainerMetricPoint = {
      timestamp: new Date(now - (count - 1 - i) * intervalMinutes * 60_000).toISOString(),
    };
    containerNames.forEach((name) => {
      point[name] = baseFn(name, i);
    });
    return point;
  });
}

// ── Server definitions ──

const SERVER_DEFS = [
  { name: "bang-bang-bar", hostname: "host.docker.internal", os: "ubuntu-4gb-ash-2", kernel: "5.15.0-70-generic", cpuModel: "AMD EPYC Processor", cores: 3, threads: 3, memGb: 4, diskGb: 78 },
  { name: "black-lodge", hostname: "vps-fra-01.bifrost.dev", os: "debian-12", kernel: "6.1.0-18-amd64", cpuModel: "AMD EPYC 7B13", cores: 4, threads: 8, memGb: 8, diskGb: 160 },
  { name: "double-r", hostname: "vps-nyc-02.bifrost.dev", os: "ubuntu-22.04", kernel: "5.15.0-94-generic", cpuModel: "Intel Xeon E-2388G", cores: 2, threads: 4, memGb: 4, diskGb: 80 },
  { name: "great-northern", hostname: "prod-web-01.internal", os: "rocky-linux-9", kernel: "5.14.0-362.el9", cpuModel: "AMD EPYC 9654", cores: 8, threads: 16, memGb: 32, diskGb: 500 },
  { name: "jack-rabbits-palace", hostname: "staging-01.internal", os: "ubuntu-24.04", kernel: "6.8.0-31-generic", cpuModel: "Intel Xeon Gold 6248", cores: 4, threads: 8, memGb: 16, diskGb: 200 },
  { name: "laura-palmer", hostname: "db-primary.internal", os: "debian-12", kernel: "6.1.0-21-amd64", cpuModel: "AMD Ryzen 9 7950X", cores: 4, threads: 8, memGb: 16, diskGb: 256 },
  { name: "localhost", hostname: "dev-machine.local", os: "fedora-39", kernel: "6.7.9-200.fc39", cpuModel: "AMD Ryzen 7 5800X", cores: 8, threads: 16, memGb: 32, diskGb: 512 },
  { name: "log-lady", hostname: "log-aggregator.internal", os: "ubuntu-22.04", kernel: "5.15.0-100-generic", cpuModel: "AMD EPYC 7R13", cores: 2, threads: 4, memGb: 8, diskGb: 120 },
  { name: "one-eyed-jacks", hostname: "edge-proxy-01.bifrost.dev", os: "alpine-3.19", kernel: "6.6.16-0-lts", cpuModel: "Intel Xeon E-2276G", cores: 4, threads: 8, memGb: 8, diskGb: 100 },
  { name: "owl-cave", hostname: "monitoring-01.internal", os: "ubuntu-22.04", kernel: "5.15.0-92-generic", cpuModel: "Intel Xeon E5-2680", cores: 2, threads: 4, memGb: 8, diskGb: 160 },
  { name: "red-room", hostname: "cache-01.internal", os: "debian-11", kernel: "5.10.0-28-amd64", cpuModel: "AMD EPYC 7402P", cores: 4, threads: 8, memGb: 16, diskGb: 200 },
  { name: "white-lodge", hostname: "api-gateway.bifrost.dev", os: "ubuntu-22.04", kernel: "5.15.0-97-generic", cpuModel: "AMD EPYC 7B12", cores: 4, threads: 8, memGb: 8, diskGb: 160 },
  { name: "glastonbury", hostname: "worker-01.internal", os: "rocky-linux-9", kernel: "5.14.0-362.el9", cpuModel: "Intel Xeon Gold 6348", cores: 8, threads: 16, memGb: 32, diskGb: 500 },
  { name: "twin-peaks", hostname: "analytics-01.internal", os: "ubuntu-24.04", kernel: "6.8.0-35-generic", cpuModel: "AMD EPYC 9554", cores: 16, threads: 32, memGb: 64, diskGb: 1000 },
];

const CONTAINER_IMAGES: Record<string, string[]> = {
  "service-a": ["service-a/backend:latest", "service-a/frontend:latest", "service-a/worker:latest"],
  "service-b": ["service-b/api:latest", "service-b/web:latest", "service-b/worker:latest", "qdrant/qdrant:v1.8"],
  infra: ["postgres:16", "redis:7-alpine", "nginx:1.25-alpine"],
  monitoring: ["grafana/grafana:10.3", "prom/prometheus:v2.50"],
  "llm-tools": ["ghcr.io/ggerganov/whisper.cpp:latest", "ollama/ollama:latest"],
};

function buildContainers(projectName: string, images: string[]) {
  return images.map((image, idx) => {
    const name = `${projectName}-${image.split("/").pop()?.split(":")[0] ?? "svc"}-${idx + 1}`;
    return {
      id: `ctr-${projectName}-${idx}`,
      name,
      image,
      status: seededRandom() > 0.05 ? "running" : "exited",
      health: seededRandom() > 0.1 ? "healthy" : "unhealthy",
      cpu_usage_pct: randomBetween(0, 12),
      memory_mb: randomBetween(30, 600),
      restart_count: Math.floor(seededRandom() * 3),
      uptime: `${Math.floor(seededRandom() * 60) + 1}d`,
      ports: idx === 0 ? [`${3000 + idx}:${3000 + idx}`] : [],
      command: image.includes("worker") ? "python worker.py" : "docker-entrypoint.sh",
      last_seen_at: new Date().toISOString(),
    };
  });
}

function buildServices(serverId: string) {
  const projectKeys = Object.keys(CONTAINER_IMAGES);
  const count = Math.floor(seededRandom() * 3) + 2;
  const selected = projectKeys.sort(() => seededRandom() - 0.5).slice(0, count);

  const projects = selected.map((project) => {
    const containers = buildContainers(project, CONTAINER_IMAGES[project]);
    return {
      id: `svc-${serverId}-${project}`,
      tenant_id: "tenant-demo",
      server_id: serverId,
      name: project,
      compose_project: project,
      status: containers.every((c) => c.status === "running") ? "running" : "degraded",
      container_count: containers.length,
      restart_count: containers.reduce((a, c) => a + c.restart_count, 0),
      published_ports: containers.flatMap((c) => c.ports),
      containers,
      last_log_timestamp: new Date().toISOString(),
    };
  });

  const standaloneCount = Math.floor(seededRandom() * 3) + 1;
  const standalones = Array.from({ length: standaloneCount }).map((_, i) => {
    const name = `standalone-${i}`;
    const containers = buildContainers(name, ["nginx:alpine"]);
    return {
      id: `svc-${serverId}-standalone-${i}`,
      tenant_id: "tenant-demo",
      server_id: serverId,
      name: containers[0].name,
      compose_project: "",
      status: containers[0].status,
      container_count: 1,
      restart_count: containers[0].restart_count,
      published_ports: containers[0].ports,
      containers,
      last_log_timestamp: new Date().toISOString(),
    };
  });

  return [...projects, ...standalones];
}

function buildMetrics(baseCpu: number, baseMem: number, baseDisk: number) {
  const POINTS = 60;
  return [
    {
      key: "cpu_usage_pct",
      unit: "%",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, Math.min(100, baseCpu + Math.sin(i / 5) * 2 + (seededRandom() - 0.5) * 3))
      ),
    },
    {
      key: "memory_usage_pct",
      unit: "%",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, Math.min(100, baseMem + Math.cos(i / 8) * 4 + (seededRandom() - 0.5) * 2))
      ),
    },
    {
      key: "disk_usage_pct",
      unit: "%",
      points: generateTimeSeries(POINTS, () =>
        Math.max(0, Math.min(100, baseDisk + (seededRandom() - 0.5) * 0.5))
      ),
    },
    {
      key: "network_rx_mb",
      unit: " MB/s",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, 0.02 + Math.sin(i / 6) * 0.015 + seededRandom() * 0.01)
      ),
    },
    {
      key: "network_tx_mb",
      unit: " MB/s",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, 0.01 + Math.cos(i / 7) * 0.008 + seededRandom() * 0.005)
      ),
    },
    {
      key: "disk_read_mb",
      unit: " MB/s",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, 0.03 + Math.sin(i / 4) * 0.02 + seededRandom() * 0.015)
      ),
    },
    {
      key: "disk_write_mb",
      unit: " MB/s",
      points: generateTimeSeries(POINTS, (i) =>
        Math.max(0, 0.02 + Math.cos(i / 5) * 0.015 + seededRandom() * 0.01)
      ),
    },
  ];
}

function buildContainerMetrics(containerNames: string[]) {
  const POINTS = 60;
  return {
    cpu: generateContainerTimeSeries(POINTS, containerNames, (name, i) => {
      const hash = name.length * 7;
      return Math.max(0, Math.sin((i + hash) / 6) * 0.5 + 0.3 + seededRandom() * 0.2);
    }),
    memory: generateContainerTimeSeries(POINTS, containerNames, (name, i) => {
      const hash = name.length * 13;
      return Math.max(0, 50 + Math.sin((i + hash) / 8) * 30 + seededRandom() * 20);
    }),
    network: generateContainerTimeSeries(POINTS, containerNames, (name, i) => {
      const hash = name.length * 11;
      return Math.max(0, 0.01 + Math.sin((i + hash) / 5) * 0.008 + seededRandom() * 0.005);
    }),
  };
}

// ── Build bundles ──

export const serverBundles: ServerBundle[] = SERVER_DEFS.map((def, idx) => {
  const serverId = `srv-${idx.toString().padStart(3, "0")}`;
  const cpu = randomBetween(0.8, 8);
  const mem = randomBetween(17, 45);
  const disk = randomBetween(19, 72);

  const services = buildServices(serverId);
  const allContainerNames = services.flatMap((s) => s.containers.map((c) => c.name));

  return {
    server: {
      id: serverId,
      tenant_id: "tenant-demo",
      name: def.name,
      hostname: def.hostname,
      public_ip: `${10 + idx}.${Math.floor(seededRandom() * 255)}.${Math.floor(seededRandom() * 255)}.${Math.floor(seededRandom() * 255)}`,
      agent_version: "0.8.0",
      status: seededRandom() > 0.07 ? "up" : "down",
      last_seen_at: new Date().toISOString(),
      uptime_seconds: Math.floor(seededRandom() * 600 + 1) * 86400,
      cpu_usage_pct: cpu,
      memory_usage_pct: mem,
      disk_usage_pct: disk,
      network_rx_mb: randomBetween(0, 0.08),
      network_tx_mb: randomBetween(0, 0.04),
      load_average: `${randomBetween(0, 2)} ${randomBetween(0, 2)} ${randomBetween(0, 1.5)}`,
      os: def.os,
      kernel: def.kernel,
      cpu_model: def.cpuModel,
      cpu_cores: def.cores,
      cpu_threads: def.threads,
      total_memory_gb: def.memGb,
      total_disk_gb: def.diskGb,
    },
    services,
    metrics: buildMetrics(cpu, mem, disk),
    containerMetrics: buildContainerMetrics(allContainerNames.slice(0, 8)),
  };
});

export function getServerBundle(id: string): ServerBundle | undefined {
  return serverBundles.find((b) => b.server.id === id);
}

// ── Mock Logs & Events Generators ──

const LOG_MESSAGES = [
  "Handling GET request to /api/health",
  "User authentication successful",
  "Connected to database successfully",
  "Error: Could not resolve host redis",
  "Sending heartbeat to control plane",
  "Processed 42 background jobs in queue",
  "Updating cache keys",
  "GET /metrics 200 OK - 15ms",
  "WARN: High memory usage detected",
  "Restarting component due to memory limit",
];

export function generateContainerLogs(containerName: string, serviceTag: string, count: number, containerId: string = ""): LogLine[] {
  const now = Date.now();
  const levels: LogLine["level"][] = ["info", "warn", "error"];
  return Array.from({ length: count }, (_, i) => {
    const msg = LOG_MESSAGES[Math.floor(seededRandom() * LOG_MESSAGES.length)];
    return {
      id: `log-${containerName}-${i}`,
      timestamp: new Date(now - (count - 1 - i) * 1000).toISOString(),
      containerId,
      containerName,
      serviceTag,
      level: levels[Math.floor(seededRandom() * levels.length)],
      message: msg,
    };
  });
}

export function generateProjectLogs(project: Service, count: number): LogLine[] {
  let logs: LogLine[] = [];
  project.containers.forEach((c) => {
    const serviceTag = c.name.replace(`${project.name}-`, "").split("-")[0];
    logs = logs.concat(generateContainerLogs(c.name, serviceTag, Math.ceil(count / project.containers.length), c.id));
  });
  return logs.sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
}

export function generateEvents(count: number, entities: { name: string, isHealthy: boolean }[]): EventLog[] {
  const now = Date.now();
  const types: EventLog["type"][] = ["restart", "crash", "health_change", "start", "stop"];
  
  return Array.from({ length: count }, (_, i) => {
    const entity = entities[Math.floor(seededRandom() * entities.length)];
    const type = types[Math.floor(seededRandom() * types.length)];
    
    let message = "";
    switch (type) {
      case "restart": message = `Service restarted gracefully`; break;
      case "crash": message = `Process exited with code 137 (OOM Killed)`; break;
      case "health_change": message = `Health status changed to ${entity.isHealthy ? 'healthy' : 'unhealthy'}`; break;
      case "start": message = `Container started`; break;
      case "stop": message = `Container stopped`; break;
    }

    return {
      id: `event-${i}`,
      timestamp: new Date(now - (count - 1 - i) * 50000).toISOString(),
      type,
      message,
      entityName: entity.name,
    };
  }).reverse(); // newest first
}
