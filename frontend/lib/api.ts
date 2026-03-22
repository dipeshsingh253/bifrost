import type { GetServerSidePropsContext } from "next";

import type {
  Container,
  EventLog,
  LogLine,
  MetricPoint,
  MetricSeries,
  Server,
  ServerBundle,
  Service,
} from "@/lib/types";

type ApiEnvelope<T> = {
  success: boolean;
  data: T;
};

type ServerListItem = {
  server: Server;
  services: Service[];
};

type ProjectDetailResponse = {
  server: Server;
  project: Service;
};

type ProjectMetricsResponse = {
  project: Service;
  metrics: ServerBundle["containerMetrics"];
};

type ProjectLogsResponse = {
  project: Service;
  logs: LogLine[];
};

type ProjectEventsResponse = {
  project: Service;
  events: EventLog[];
};

type ContainersResponse = {
  server: Server;
  containers: Container[];
};

type ContainerDetailResponse = {
  server?: Server;
  project: Service;
  container: Container;
};

type ContainerMetricsResponse = {
  project: Service;
  container: Container;
  metrics: {
    cpu: MetricPoint[];
    memory: MetricPoint[];
    network: MetricPoint[];
  };
};

type ContainerLogsResponse = {
  project: Service;
  container: Container;
  logs: LogLine[];
};

type ContainerEventsResponse = {
  project: Service;
  container: Container;
  events: EventLog[];
};

type ContainerEnvResponse = {
  project: Service;
  container: Container;
  env: Record<string, string>;
};

const defaultApiBaseUrl = "http://127.0.0.1:8080";
const defaultBearerToken = "demo-owner-token";

export function getApiBaseUrl(): string {
  return process.env.BIFROST_API_BASE_URL || defaultApiBaseUrl;
}

function getBearerToken(): string {
  return process.env.BIFROST_BEARER_TOKEN || defaultBearerToken;
}

async function apiFetch<T>(path: string): Promise<T> {
  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    headers: {
      Authorization: `Bearer ${getBearerToken()}`,
    },
  });

  if (!response.ok) {
    throw new Error(`API request failed: ${response.status}`);
  }

  const payload = (await response.json()) as ApiEnvelope<T>;
  if (!payload.success) {
    throw new Error("API response was not successful");
  }

  return payload.data;
}

export async function fetchServerList(): Promise<ServerListItem[]> {
  return apiFetch<ServerListItem[]>("/api/v1/servers");
}

export async function fetchServerBundle(serverID: string): Promise<ServerBundle> {
  return apiFetch<ServerBundle>(`/api/v1/servers/${serverID}`);
}

export async function fetchProjects(serverID: string): Promise<{ server: Server; projects: Service[] }> {
  return apiFetch<{ server: Server; projects: Service[] }>(`/api/v1/servers/${serverID}/projects`);
}

export async function fetchProjectDetail(serverID: string, projectID: string): Promise<ProjectDetailResponse> {
  return apiFetch<ProjectDetailResponse>(`/api/v1/servers/${serverID}/projects/${projectID}`);
}

export async function fetchProjectMetrics(serverID: string, projectID: string): Promise<ProjectMetricsResponse> {
  return apiFetch<ProjectMetricsResponse>(`/api/v1/servers/${serverID}/projects/${projectID}/metrics`);
}

export async function fetchProjectLogs(serverID: string, projectID: string): Promise<ProjectLogsResponse> {
  return apiFetch<ProjectLogsResponse>(`/api/v1/servers/${serverID}/projects/${projectID}/logs`);
}

export async function fetchProjectEvents(serverID: string, projectID: string): Promise<ProjectEventsResponse> {
  return apiFetch<ProjectEventsResponse>(`/api/v1/servers/${serverID}/projects/${projectID}/events`);
}

export async function fetchStandaloneContainers(serverID: string): Promise<ContainersResponse> {
  return apiFetch<ContainersResponse>(`/api/v1/servers/${serverID}/containers?standalone=true`);
}

export async function fetchContainerDetail(serverID: string, containerID: string): Promise<ContainerDetailResponse> {
  return apiFetch<ContainerDetailResponse>(`/api/v1/servers/${serverID}/containers/${containerID}`);
}

export async function fetchContainerMetrics(serverID: string, containerID: string): Promise<ContainerMetricsResponse> {
  return apiFetch<ContainerMetricsResponse>(`/api/v1/servers/${serverID}/containers/${containerID}/metrics`);
}

export async function fetchContainerLogs(serverID: string, containerID: string): Promise<ContainerLogsResponse> {
  return apiFetch<ContainerLogsResponse>(`/api/v1/servers/${serverID}/containers/${containerID}/logs`);
}

export async function fetchContainerEvents(serverID: string, containerID: string): Promise<ContainerEventsResponse> {
  return apiFetch<ContainerEventsResponse>(`/api/v1/servers/${serverID}/containers/${containerID}/events`);
}

export async function fetchContainerEnv(serverID: string, containerID: string): Promise<ContainerEnvResponse> {
  return apiFetch<ContainerEnvResponse>(`/api/v1/servers/${serverID}/containers/${containerID}/env`);
}

export async function safeServerSideProps<T>(
  _context: GetServerSidePropsContext,
  loader: () => Promise<T>
): Promise<{ props: T }> {
  try {
    return { props: await loader() };
  } catch {
    return { props: nullProps<T>() };
  }
}

function nullProps<T>(): T {
  return null as T;
}

export function metricHistoryToSeries(key: string, unit: string, points: MetricPoint[]): MetricSeries {
  return { key, unit, points };
}
