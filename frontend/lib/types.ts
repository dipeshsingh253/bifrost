export type Container = {
  id: string;
  name: string;
  image: string;
  status: string;
  health: string;
  cpu_usage_pct: number;
  memory_mb: number;
  restart_count: number;
  uptime: string;
  ports: string[];
  command: string;
  last_seen_at: string;
};

export type Service = {
  id: string;
  tenant_id: string;
  server_id: string;
  name: string;
  compose_project: string;
  status: string;
  container_count: number;
  restart_count: number;
  published_ports: string[];
  containers: Container[];
  last_log_timestamp: string;
};

export type Server = {
  id: string;
  tenant_id: string;
  name: string;
  hostname: string;
  public_ip: string;
  agent_version: string;
  status: string;
  last_seen_at: string;
  uptime_seconds: number;
  cpu_usage_pct: number;
  memory_usage_pct: number;
  disk_usage_pct: number;
  network_rx_mb: number;
  network_tx_mb: number;
  load_average: string;
  os: string;
  kernel: string;
  cpu_model: string;
  cpu_cores: number;
  cpu_threads: number;
  total_memory_gb: number;
  total_disk_gb: number;
};

export type MetricPoint = {
  timestamp: string;
  value: number;
};

export type MetricSeries = {
  key: string;
  unit: string;
  points: MetricPoint[];
};

export type ContainerMetricPoint = {
  timestamp: string;
  [containerKey: string]: number | string;
};

export type ContainerMetricSeries = {
  key: string;
  label: string;
};

export type ServerBundle = {
  server: Server;
  services: Service[];
  metrics: MetricSeries[];
  containerMetrics: {
    cpu: ContainerMetricPoint[];
    memory: ContainerMetricPoint[];
    network: ContainerMetricPoint[];
  };
};

export type LogLine = {
  id: string;
  timestamp: string;
  containerId: string;
  containerName: string;
  serviceTag: string;
  message: string;
};

export type EventLog = {
  id: string;
  timestamp: string;
  type: "restart" | "crash" | "health_change" | "start" | "stop";
  message: string;
  entityName: string;
};
