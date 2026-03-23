import Head from "next/head";
import type { GetServerSideProps } from "next";
import Link from "next/link";
import { ArrowLeft, Box, Activity, Clock, RefreshCw, Hash, Calendar, Shield, Server, List } from "lucide-react";

import { Layout } from "@/components/Layout";
import {
  fetchContainerDetail,
  fetchContainerEnv,
  getApiErrorMessage,
  isApiErrorCode,
  isApiErrorStatus,
  fetchContainerLogs,
  fetchContainerMetrics,
  requireAuthenticatedPage,
  type AuthenticatedPageProps,
} from "@/lib/api";
import { MonitoringUnavailableState } from "@/components/MonitoringUnavailableState";
import { ChartCard } from "@/components/charts/ChartCard";
import { BifrostAreaChart } from "@/components/charts/AreaChart";
import { LogViewer } from "@/components/server/LogViewer";
import type { Container, LogLine, MetricPoint, Server as ServerType, Service } from "@/lib/types";
import { getContainerDetailNotFoundKind, type DetailNotFoundKind } from "@/lib/monitoring-not-found";
import { serverPath, serverProjectPath } from "@/lib/monitoring-routes";

type ContainerDetailProps = {
  server: ServerType | null;
  project: Service | null;
  container: Container | null;
  metrics: {
    cpu: MetricPoint[];
    memory: MetricPoint[];
    network: MetricPoint[];
  };
  logs: LogLine[];
  env: Record<string, string>;
  loadError: string | null;
  notFoundKind: DetailNotFoundKind;
} & AuthenticatedPageProps;

export default function ContainerDetail({
  server,
  project,
  container,
  metrics,
  logs,
  env,
  loadError,
  notFoundKind,
  currentUser,
}: ContainerDetailProps) {

  if (loadError) {
    return (
      <Layout currentUser={currentUser}>
        <MonitoringUnavailableState
          message={loadError}
          title="Container monitoring is temporarily unavailable."
        />
      </Layout>
    );
  }

  if (notFoundKind === "server") {
    return (
      <Layout currentUser={currentUser}>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          Server not found.
        </div>
      </Layout>
    );
  }

  if (notFoundKind === "container") {
    return (
      <Layout currentUser={currentUser}>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          Container not found.
        </div>
      </Layout>
    );
  }

  if (!server || !container || !project) {
    return (
      <Layout currentUser={currentUser}>
        <MonitoringUnavailableState
          message="Container details could not be resolved."
          title="Container monitoring is temporarily unavailable."
        />
      </Layout>
    );
  }

  const isStandalone = project.compose_project === "";
  const serviceTag = isStandalone ? container.name : container.name.replace(`${project.name}-`, "").split("-")[0];

  // Map container metrics to simple points
  const cpuPoints = metrics.cpu;
  const memPoints = metrics.memory;
  const netPoints = metrics.network;

  return (
    <>
      <Head>
        <title>{`${container.name} · ${server.name} · Bifrost`}</title>
      </Head>
      <Layout currentUser={currentUser}>
        {/* Navigation */}
        <div className="mb-6 flex flex-col gap-4">
          <nav className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
            <Link href={serverPath(server.id)} className="hover:text-foreground transition-colors inline-flex items-center gap-1">
              <ArrowLeft className="h-3.5 w-3.5" />
              {server.name}
            </Link>
            {!isStandalone && (
              <>
                <span className="text-border">/</span>
                <Link href={serverProjectPath(server.id, project.id)} className="hover:text-foreground transition-colors">
                  {project.name}
                </Link>
              </>
            )}
            <span className="text-border">/</span>
            <span className="text-foreground font-medium">Container: {container.name}</span>
          </nav>

          {/* Header Dashboard */}
          <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-3">
                <span
                  className={`h-3 w-3 rounded-full ${
                    container.status === "running" ? (container.health === "healthy" ? "bg-[hsl(140_50%_48%)]" : "bg-warning") : "bg-destructive"
                  }`}
                />
                <h1 className="text-2xl font-semibold text-foreground">{container.name}</h1>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <span className="bg-muted px-2 py-0.5 rounded text-xs text-foreground/80 font-mono">
                  {container.image}
                </span>
                {!isStandalone && (
                  <>
                    <span>in project</span>
                    <span className="text-foreground font-medium">{project.name}</span>
                    <span className="bg-[hsl(280_65%_70%)]/10 text-[hsl(280_65%_70%)] px-2 py-0.5 rounded text-xs uppercase font-medium">
                      {serviceTag}
                    </span>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Activity className="h-4 w-4" />
              <span>Status</span>
            </div>
            <div className="text-2xl font-semibold text-foreground flex items-baseline gap-2">
              <span className="capitalize">{container.status}</span>
              {container.status === "running" && (
                <span className={`text-xs font-normal ${container.health === "healthy" ? "text-success" : "text-destructive"}`}>
                  ({container.health})
                </span>
              )}
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Clock className="h-4 w-4" />
              <span>Uptime</span>
            </div>
            <div className="text-2xl font-semibold text-foreground">
              {container.status === "running" ? container.uptime : "-"}
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <RefreshCw className="h-4 w-4" />
              <span>Restarts</span>
            </div>
            <div className="text-2xl font-semibold text-foreground">
              {container.restart_count}
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Server className="h-4 w-4" />
              <span>Live Usage</span>
            </div>
            <div className="text-sm font-medium text-foreground py-0.5">
              {container.cpu_usage_pct.toFixed(1)}% CPU
            </div>
            <div className="text-sm font-medium text-foreground py-0.5">
              {container.memory_mb.toFixed(0)} MB RAM
            </div>
          </div>
        </div>

        {/* ── Resource Usage ── */}
        <div className="mb-8">
          <h2 className="text-lg font-semibold text-foreground mb-4">Resource Usage</h2>
          <div className="grid gap-4 xl:grid-cols-3">
            <ChartCard title="CPU Usage" description="Container CPU utilization">
              <BifrostAreaChart
                data={cpuPoints}
                unit="%"
                color="green"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(1)}%`}
              />
            </ChartCard>
            <ChartCard title="Memory Usage" description="Container memory consumption">
              <BifrostAreaChart
                data={memPoints}
                unit=" MB"
                color="purple"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(0)} MB`}
              />
            </ChartCard>
            <ChartCard title="Network I/O" description="Container traffic throughput">
              <BifrostAreaChart
                data={netPoints}
                unit=" MB/s"
                color="blue"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(2)} MB/s`}
              />
            </ChartCard>
          </div>
        </div>

        <div className="grid gap-8 lg:grid-cols-3 mb-8">
          <div className="lg:col-span-2 flex flex-col gap-8">
            {/* ── Logs ── */}
            <div>
              <LogViewer logs={logs} title="Container Logs" />
            </div>
          </div>

          <div className="flex flex-col gap-8">
            {/* ── Details Grid ── */}
            <div>
              <h2 className="text-lg font-semibold text-foreground mb-4">Container Details</h2>
              <div className="rounded-lg border border-border bg-card flex flex-col overflow-hidden">
                <div className="flex flex-col gap-1 p-4 border-b border-border">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground flex items-center gap-2"><Hash className="h-3.5 w-3.5" /> Container ID</span>
                    <span className="font-mono text-xs max-w-[140px] truncate" title={container.id}>{container.id}</span>
                  </div>
                </div>
                <div className="flex flex-col gap-1 p-4 border-b border-border">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground flex items-center gap-2"><Box className="h-3.5 w-3.5" /> Image</span>
                    <span className="font-mono text-xs max-w-[180px] break-all text-right">{container.image}</span>
                  </div>
                </div>
                <div className="flex flex-col gap-1 p-4 border-b border-border">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground flex items-center gap-2"><Shield className="h-3.5 w-3.5" /> Health</span>
                    <span className="text-foreground capitalize">{container.health}</span>
                  </div>
                </div>
                <div className="flex flex-col gap-1 p-4 border-b border-border">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground flex items-center gap-2"><Calendar className="h-3.5 w-3.5" /> Last Seen</span>
                    <span className="text-foreground text-xs tabular-nums">
                      {new Date(container.last_seen_at).toLocaleString()}
                    </span>
                  </div>
                </div>
                <div className="flex flex-col gap-2 p-4 border-b border-border">
                  <div className="text-sm text-muted-foreground flex items-center gap-2">
                    <List className="h-3.5 w-3.5" /> Command
                  </div>
                  <div className="bg-[#0A0A0B] border border-border rounded px-3 py-2 text-xs font-mono text-foreground break-all">
                    {container.command}
                  </div>
                </div>
                <div className="flex flex-col gap-2 p-4">
                  <div className="text-sm text-muted-foreground flex items-center justify-between">
                    <span>Ports</span>
                    {container.ports.length === 0 && <span className="text-xs">None exposed</span>}
                  </div>
                  {container.ports.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-1">
                      {container.ports.map(port => (
                        <span key={port} className="bg-muted px-2 py-1 rounded-md text-[11px] font-mono border border-border">
                          {port}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* ── Env Vars ── */}
            <div>
              <h2 className="text-lg font-semibold text-foreground mb-4">Environment Variables</h2>
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex flex-col gap-2">
                  {Object.entries(env).map(([k, v]) => (
                    <div key={k} className="flex flex-col sm:flex-row sm:items-baseline sm:justify-between border-b border-border/50 pb-2 last:border-0 last:pb-0 gap-1">
                      <span className="text-sm text-[hsl(280_65%_70%)] font-mono truncate">{k}</span>
                      <span className="text-xs text-foreground font-mono break-all">{v}</span>
                    </div>
                  ))}
                  <div className="text-xs text-muted-foreground mt-2 italic text-center">
                    ({Object.keys(env).length} variables)
                  </div>
                </div>
              </div>
            </div>

          </div>
        </div>
      </Layout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<ContainerDetailProps> = async (context) => {
  const serverRouteID = context.params?.id;
  const containerRouteID = context.params?.containerId;
  if (typeof serverRouteID !== "string" || typeof containerRouteID !== "string") {
    return requireAuthenticatedPage(context, async () => ({
      server: null,
      project: null,
        container: null,
        metrics: { cpu: [], memory: [], network: [] },
        logs: [],
        env: {},
        loadError: null,
        notFoundKind: null,
      }));
  }

  return requireAuthenticatedPage(context, async () => {
    try {
      const [detail, metrics, logs, env] = await Promise.all([
        fetchContainerDetail(serverRouteID, containerRouteID, context),
        fetchContainerMetrics(serverRouteID, containerRouteID, context),
        fetchContainerLogs(serverRouteID, containerRouteID, context),
        fetchContainerEnv(serverRouteID, containerRouteID, context),
      ]);

      return {
        server: detail.server ?? null,
        project: detail.project,
        container: detail.container,
        metrics: metrics.metrics,
        logs: logs.logs,
        env: env.env,
        loadError: null,
        notFoundKind: null,
      };
    } catch (error) {
      if (isApiErrorStatus(error, 401)) {
        throw error;
      }
      const notFoundKind = getContainerDetailNotFoundKind(error);
      if (notFoundKind) {
        return {
          server: null,
          project: null,
          container: null,
          metrics: { cpu: [], memory: [], network: [] },
          logs: [],
          env: {},
          loadError: null,
          notFoundKind,
        };
      }
      return {
        server: null,
        project: null,
        container: null,
        metrics: { cpu: [], memory: [], network: [] },
        logs: [],
        env: {},
        loadError: getApiErrorMessage(error, "Failed to load container details from the backend."),
        notFoundKind: null,
      };
    }
  });
};
