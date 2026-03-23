import type { GetServerSidePropsContext, GetServerSidePropsResult } from "next";

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
import {
  serverApiPath,
  serverContainerApiPath,
  serverProjectApiPath,
  serverProjectsApiPath,
  serverStandaloneContainersApiPath,
} from "@/lib/monitoring-routes";

type ApiEnvelope<T> = {
  success: boolean;
  data: T;
};

type ApiErrorEnvelope = {
  success: false;
  error?: {
    message?: string;
    code?: string;
    details?: unknown;
  };
};

export type AuthUser = {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: string;
};

export type BootstrapStatus = {
  needs_bootstrap: boolean;
};

export type ViewerInvite = {
  id: string;
  email: string;
  role: string;
  status: string;
  expires_at: string;
  invite_token?: string;
};

export type ViewerAccount = {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: string;
  status: string;
  disabled_at?: string | null;
};

export type ViewerAccess = {
  viewers: ViewerAccount[];
  invites: ViewerInvite[];
};

export type TenantSummary = {
  tenant_id: string;
  tenant_name: string;
  admin_count: number;
  viewer_count: number;
};

export type AdminSummaryResponse = {
  tenant: TenantSummary;
  user: AuthUser;
};

type ApiFetchOptions = {
  body?: unknown;
  context?: GetServerSidePropsContext;
  headers?: HeadersInit;
  method?: string;
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

export class ApiRequestError extends Error {
  code?: string;
  details?: unknown;
  status: number;

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

export type AuthenticatedPageProps = {
  currentUser: AuthUser;
};

export function getApiBaseUrl(): string {
  return process.env.BIFROST_API_BASE_URL || defaultApiBaseUrl;
}

export function hasAdminAccess(user: Pick<AuthUser, "role"> | null | undefined): boolean {
  return user?.role === "admin" || user?.role === "owner";
}

export function isApiRequestError(error: unknown): error is ApiRequestError {
  return error instanceof ApiRequestError;
}

export function isApiErrorStatus(error: unknown, status: number): boolean {
  return isApiRequestError(error) && error.status === status;
}

export function isApiErrorCode(error: unknown, ...codes: string[]): boolean {
  return isApiRequestError(error) && typeof error.code === "string" && codes.includes(error.code);
}

export function getApiErrorMessage(error: unknown, fallback = "Request failed"): string {
  if (isApiRequestError(error)) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  return fallback;
}

function isUnauthorizedError(error: unknown): boolean {
  return isApiErrorStatus(error, 401);
}

function isForbiddenError(error: unknown): boolean {
  return isApiErrorStatus(error, 403);
}

function isJsonResponse(contentType: string | null): boolean {
  return Boolean(contentType && contentType.toLowerCase().includes("application/json"));
}

function requestCookieHeader(context?: GetServerSidePropsContext): string {
  return context?.req.headers.cookie ?? "";
}

async function readApiPayload<T>(response: Response): Promise<ApiEnvelope<T>> {
  const contentType = response.headers.get("content-type");
  const text = await response.text();

  if (!isJsonResponse(contentType) || text.trim() === "") {
    throw new ApiRequestError("API response was not JSON", response.status);
  }

  const payload = JSON.parse(text) as ApiEnvelope<T> | ApiErrorEnvelope;
  if (!response.ok) {
    const error = "error" in payload ? payload.error : undefined;
    throw new ApiRequestError(
      error?.message || `API request failed: ${response.status}`,
      response.status,
      error?.code,
      error?.details
    );
  }

  if (!payload.success) {
    const error = "error" in payload ? payload.error : undefined;
    throw new ApiRequestError(
      error?.message || "API response was not successful",
      response.status,
      error?.code,
      error?.details
    );
  }

  return payload;
}

async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const cookie = requestCookieHeader(options.context);

  if (cookie !== "") {
    headers.set("Cookie", cookie);
  }

  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${getApiBaseUrl()}${path}`, {
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    headers,
    method: options.method || "GET",
  });

  const payload = await readApiPayload<T>(response);
  return payload.data;
}

export async function fetchBootstrapStatus(context?: GetServerSidePropsContext): Promise<BootstrapStatus> {
  return apiFetch<BootstrapStatus>("/api/v1/auth/bootstrap/status", { context });
}

export async function fetchSession(context: GetServerSidePropsContext): Promise<AuthUser> {
  return apiFetch<AuthUser>("/api/v1/auth/session", { context });
}

export async function fetchOptionalSession(context: GetServerSidePropsContext): Promise<AuthUser | null> {
  try {
    return await fetchSession(context);
  } catch (error) {
    if (isUnauthorizedError(error)) {
      return null;
    }
    throw error;
  }
}

async function unauthenticatedRedirect<T extends Record<string, unknown>>(
  context: GetServerSidePropsContext
): Promise<GetServerSidePropsResult<T & AuthenticatedPageProps>> {
  try {
    const bootstrap = await fetchBootstrapStatus(context);
    return {
      redirect: {
        destination: bootstrap.needs_bootstrap ? "/setup" : "/login",
        permanent: false,
      },
    };
  } catch {
    return {
      redirect: {
        destination: "/login",
        permanent: false,
      },
    };
  }
}

export async function requireAuthenticatedPage<T extends Record<string, unknown>>(
  context: GetServerSidePropsContext,
  loader: (currentUser: AuthUser) => Promise<T>
): Promise<GetServerSidePropsResult<T & AuthenticatedPageProps>> {
  let currentUser: AuthUser;
  try {
    currentUser = await fetchSession(context);
  } catch (error) {
    if (isUnauthorizedError(error)) {
      return unauthenticatedRedirect<T>(context);
    }
    throw error;
  }

  try {
    const props = await loader(currentUser);
    return {
      props: {
        ...props,
        currentUser,
      },
    };
  } catch (error) {
    if (isUnauthorizedError(error)) {
      return unauthenticatedRedirect<T>(context);
    }
    throw error;
  }
}

export async function requireAdminPage<T extends Record<string, unknown>>(
  context: GetServerSidePropsContext,
  loader: (currentUser: AuthUser) => Promise<T>
): Promise<GetServerSidePropsResult<T & AuthenticatedPageProps>> {
  return requireAuthenticatedPage(context, async (currentUser) => {
    if (!hasAdminAccess(currentUser)) {
      throw new ApiRequestError("admin access required", 403, "FORBIDDEN");
    }
    return loader(currentUser);
  }).catch((error) => {
    if (isForbiddenError(error)) {
      return {
        redirect: {
          destination: "/",
          permanent: false,
        },
      };
    }
    throw error;
  });
}

export async function fetchInviteDetail(
  token: string,
  context?: GetServerSidePropsContext
): Promise<ViewerInvite> {
  return apiFetch<ViewerInvite>(`/api/v1/auth/invites/${token}`, { context });
}

export async function fetchAdminSummary(context?: GetServerSidePropsContext): Promise<AdminSummaryResponse> {
  return apiFetch<AdminSummaryResponse>("/api/v1/admin/summary", { context });
}

export async function fetchViewerAccess(context?: GetServerSidePropsContext): Promise<ViewerAccess> {
  return apiFetch<ViewerAccess>("/api/v1/admin/access", { context });
}

export async function fetchServerList(context?: GetServerSidePropsContext): Promise<ServerListItem[]> {
  return apiFetch<ServerListItem[]>("/api/v1/servers", { context });
}

export async function fetchServerBundle(
  serverRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ServerBundle> {
  return apiFetch<ServerBundle>(serverApiPath(serverRouteID), { context });
}

export async function fetchProjects(
  serverRouteID: string,
  context?: GetServerSidePropsContext
): Promise<{ server: Server; projects: Service[] }> {
  return apiFetch<{ server: Server; projects: Service[] }>(serverProjectsApiPath(serverRouteID), {
    context,
  });
}

export async function fetchProjectDetail(
  serverRouteID: string,
  projectRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ProjectDetailResponse> {
  return apiFetch<ProjectDetailResponse>(serverProjectApiPath(serverRouteID, projectRouteID), {
    context,
  });
}

export async function fetchProjectMetrics(
  serverRouteID: string,
  projectRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ProjectMetricsResponse> {
  return apiFetch<ProjectMetricsResponse>(`${serverProjectApiPath(serverRouteID, projectRouteID)}/metrics`, {
    context,
  });
}

export async function fetchProjectLogs(
  serverRouteID: string,
  projectRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ProjectLogsResponse> {
  return apiFetch<ProjectLogsResponse>(`${serverProjectApiPath(serverRouteID, projectRouteID)}/logs`, {
    context,
  });
}

export async function fetchProjectEvents(
  serverRouteID: string,
  projectRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ProjectEventsResponse> {
  return apiFetch<ProjectEventsResponse>(`${serverProjectApiPath(serverRouteID, projectRouteID)}/events`, {
    context,
  });
}

export async function fetchStandaloneContainers(
  serverRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainersResponse> {
  return apiFetch<ContainersResponse>(serverStandaloneContainersApiPath(serverRouteID), {
    context,
  });
}

export async function fetchContainerDetail(
  serverRouteID: string,
  containerRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainerDetailResponse> {
  return apiFetch<ContainerDetailResponse>(serverContainerApiPath(serverRouteID, containerRouteID), {
    context,
  });
}

export async function fetchContainerMetrics(
  serverRouteID: string,
  containerRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainerMetricsResponse> {
  return apiFetch<ContainerMetricsResponse>(`${serverContainerApiPath(serverRouteID, containerRouteID)}/metrics`, {
    context,
  });
}

export async function fetchContainerLogs(
  serverRouteID: string,
  containerRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainerLogsResponse> {
  return apiFetch<ContainerLogsResponse>(`${serverContainerApiPath(serverRouteID, containerRouteID)}/logs`, {
    context,
  });
}

export async function fetchContainerEvents(
  serverRouteID: string,
  containerRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainerEventsResponse> {
  return apiFetch<ContainerEventsResponse>(`${serverContainerApiPath(serverRouteID, containerRouteID)}/events`, {
    context,
  });
}

export async function fetchContainerEnv(
  serverRouteID: string,
  containerRouteID: string,
  context?: GetServerSidePropsContext
): Promise<ContainerEnvResponse> {
  return apiFetch<ContainerEnvResponse>(`${serverContainerApiPath(serverRouteID, containerRouteID)}/env`, {
    context,
  });
}

export function metricHistoryToSeries(key: string, unit: string, points: MetricPoint[]): MetricSeries {
  return { key, unit, points };
}
