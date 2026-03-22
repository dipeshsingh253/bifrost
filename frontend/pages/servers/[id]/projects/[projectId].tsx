import Head from "next/head";
import type { GetServerSideProps } from "next";
import { useState, useMemo } from "react";
import Link from "next/link";
import { ArrowLeft, Activity, Box, RefreshCw, BarChart2 } from "lucide-react";

import { Layout } from "@/components/Layout";
import {
  fetchProjectDetail,
  fetchProjectEvents,
  fetchProjectLogs,
  fetchProjectMetrics,
} from "@/lib/api";
import { ChartCard } from "@/components/charts/ChartCard";
import { StackedAreaChart } from "@/components/charts/StackedAreaChart";
import { LogViewer } from "@/components/server/LogViewer";
import { EventsList } from "@/components/server/EventsList";
import type { EventLog, LogLine, Server, ServerBundle, Service } from "@/lib/types";

type ProjectDetailProps = {
  server: Server | null;
  project: Service | null;
  containerMetrics: ServerBundle["containerMetrics"];
  logs: LogLine[];
  events: EventLog[];
};

export default function ProjectDetail({
  server,
  project,
  containerMetrics,
  logs,
  events,
}: ProjectDetailProps) {
  const [selectedService, setSelectedService] = useState<string | null>(null);

  if (!server) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          Server not found.
        </div>
      </Layout>
    );
  }

  if (!project) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          Project not found.
        </div>
      </Layout>
    );
  }

  // Compute metrics
  const containerNames = project.containers.map(c => c.name);
  
  // Service grouping
  const servicesMap = new Map<string, {
    name: string;
    containers: typeof project.containers;
    running: number;
    unhealthy: number;
    cpu: number;
    mem: number;
  }>();

  project.containers.forEach(c => {
    const serviceTag = c.name.replace(`${project.name}-`, "").split("-")[0];
    if (!servicesMap.has(serviceTag)) {
      servicesMap.set(serviceTag, { name: serviceTag, containers: [], running: 0, unhealthy: 0, cpu: 0, mem: 0 });
    }
    const group = servicesMap.get(serviceTag)!;
    group.containers.push(c);
    group.cpu += c.cpu_usage_pct;
    group.mem += c.memory_mb;
    if (c.status === "running") group.running++;
    if (c.health === "unhealthy") group.unhealthy++;
  });

  const serviceGroups = Array.from(servicesMap.values());
  const unhealthyCount = project.containers.filter(c => c.health === "unhealthy").length;
  
  const totalCpu = project.containers.reduce((acc, c) => acc + c.cpu_usage_pct, 0);
  const totalMem = project.containers.reduce((acc, c) => acc + c.memory_mb, 0);
  
  const totalSystemCpu = server.cpu_cores * 100; // rough approx
  const pctCpu = ((totalCpu / totalSystemCpu) * 100).toFixed(1);
  const pctMem = ((totalMem / (server.total_memory_gb * 1024)) * 100).toFixed(1);

  // Filtered containers
  const displayContainers = selectedService 
    ? project.containers.filter(c => c.name.replace(`${project.name}-`, "").split("-")[0] === selectedService)
    : project.containers;

  return (
    <>
      <Head>
        <title>{project.name} · {server.name} · Bifrost</title>
      </Head>
      <Layout>
        {/* Navigation */}
        <div className="mb-6 flex flex-col gap-4">
          <nav className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <Link href={`/servers/${server.id}`} className="hover:text-foreground transition-colors inline-flex items-center gap-1">
              <ArrowLeft className="h-3.5 w-3.5" />
              {server.name}
            </Link>
            <span className="text-border">/</span>
            <span className="text-foreground font-medium">Project: {project.name}</span>
          </nav>

          {/* Header Dashboard */}
          <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-3">
                <span className={`h-3 w-3 rounded-full ${project.status === "running" ? "bg-success" : "bg-destructive"}`} />
                <h1 className="text-2xl font-semibold text-foreground">{project.name}</h1>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">
                Docker Compose Project on {server.hostname}
              </p>
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
              <span className="capitalize">{project.status}</span>
              {unhealthyCount > 0 && <span className="text-xs text-destructive font-normal">({unhealthyCount} unhealthy)</span>}
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Box className="h-4 w-4" />
              <span>Services / Containers</span>
            </div>
            <div className="text-2xl font-semibold text-foreground">
              {serviceGroups.length} <span className="text-base font-normal text-muted-foreground">/ {project.container_count}</span>
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <RefreshCw className="h-4 w-4" />
              <span>Total Restarts</span>
            </div>
            <div className="text-2xl font-semibold text-foreground">
              {project.restart_count}
            </div>
          </div>
          <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <BarChart2 className="h-4 w-4" />
              <span>Resource Footprint</span>
            </div>
            <div className="text-sm font-medium text-foreground py-0.5">
              {pctCpu}% CPU <span className="text-muted-foreground font-normal">({totalCpu.toFixed(1)}%)</span>
            </div>
            <div className="text-sm font-medium text-foreground py-0.5">
              {pctMem}% RAM <span className="text-muted-foreground font-normal">({totalMem.toFixed(0)} MB)</span>
            </div>
          </div>
        </div>

        {/* ── Resource Usage ── */}
        <div className="mb-8">
          <h2 className="text-lg font-semibold text-foreground mb-4">Resource Usage</h2>
          <div className="grid gap-4 xl:grid-cols-2">
            <ChartCard title="CPU Usage" description="Project containers CPU footprint">
              <StackedAreaChart
                data={containerMetrics.cpu}
                containerNames={containerNames}
                unit="%"
                tickFormatter={(v) => `${v.toFixed(1)}%`}
              />
            </ChartCard>
            <ChartCard title="Memory Usage" description="Project containers memory footprint">
              <StackedAreaChart
                data={containerMetrics.memory}
                containerNames={containerNames}
                unit=" MB"
                tickFormatter={(v) => `${v.toFixed(0)} MB`}
              />
            </ChartCard>
          </div>
        </div>

        <div className="grid gap-8 lg:grid-cols-3">
          <div className="lg:col-span-2 flex flex-col gap-8">
            {/* ── Service Groups ── */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-foreground">Service Groups</h2>
                {selectedService && (
                  <button onClick={() => setSelectedService(null)} className="text-sm text-primary hover:text-primary/80">
                    Clear filter
                  </button>
                )}
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                {serviceGroups.map(group => (
                  <div
                    key={group.name}
                    onClick={() => setSelectedService(selectedService === group.name ? null : group.name)}
                    className={`rounded-md border p-4 cursor-pointer transition-colors ${
                      selectedService === group.name 
                        ? "border-[hsl(140_50%_48%)]/50 bg-[hsl(140_50%_48%)]/5" 
                        : "border-border bg-card hover:bg-accent/50"
                    }`}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="font-medium text-foreground">{group.name}</div>
                      <div className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
                        {group.containers.length} containers
                      </div>
                    </div>
                    <div className="flex flex-col gap-1 text-sm text-muted-foreground">
                      <div className="flex justify-between">
                        <span>Status:</span>
                        <span className={group.running === group.containers.length ? "text-[hsl(140_50%_48%)]" : "text-orange-500"}>
                          {group.running} running
                          {group.unhealthy > 0 && <span className="text-destructive ml-1">({group.unhealthy} err)</span>}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span>Resources:</span>
                        <span>{group.cpu.toFixed(1)}% CPU / {group.mem.toFixed(0)} MB</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* ── Containers List ── */}
            <div>
              <h2 className="text-lg font-semibold text-foreground mb-4">
                Containers {selectedService && <span className="text-muted-foreground font-normal text-sm">(filtered by: {selectedService})</span>}
              </h2>
              <div className="flex flex-col gap-2">
                {displayContainers.map(c => {
                  const sTag = c.name.replace(`${project.name}-`, "").split("-")[0];
                  return (
                    <Link
                      key={c.id}
                      href={`/servers/${server.id}/containers/${c.id}`}
                      className="group flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-md border border-border bg-card px-4 py-3 hover:bg-accent/50 transition-colors cursor-pointer"
                    >
                      <div className="flex items-center gap-3">
                        <span
                          className={`h-2.5 w-2.5 rounded-full shrink-0 ${
                            c.status === "running" ? (c.health === "healthy" ? "bg-[hsl(140_50%_48%)]" : "bg-orange-500") : "bg-destructive"
                          }`}
                          title={`Status: ${c.status}, Health: ${c.health}`}
                        />
                        <div>
                          <div className="font-medium text-foreground flex items-center gap-2 group-hover:text-[hsl(140_50%_48%)] transition-colors">
                            {c.name}
                          </div>
                          <div className="text-xs text-muted-foreground mt-0.5 flex items-center gap-2">
                           <span className="bg-muted px-1.5 py-0.5 rounded text-[10px] uppercase">{sTag}</span>
                           <span>Uptime: {c.uptime}</span>
                           {c.restart_count > 0 && <span className="text-orange-500 text-[11px] bg-orange-500/10 px-1 py-0.5 rounded">Restarts: {c.restart_count}</span>}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-4 text-sm tabular-nums text-muted-foreground">
                        <span className="w-16 text-right">CPU {c.cpu_usage_pct.toFixed(1)}%</span>
                        <span className="w-20 text-right">RAM {c.memory_mb.toFixed(0)} MB</span>
                      </div>
                    </Link>
                  );
                })}
              </div>
            </div>

            {/* ── Logs ── */}
            <div>
              <LogViewer logs={logs} title="Project Combined Logs" />
            </div>
          </div>

          <div className="flex flex-col gap-8">
            {/* ── Events ── */}
            <div>
              <h2 className="text-lg font-semibold text-foreground mb-4">Recent Events</h2>
              <EventsList events={events} />
            </div>
          </div>
        </div>
      </Layout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<ProjectDetailProps> = async (context) => {
  const id = context.params?.id;
  const projectId = context.params?.projectId;
  if (typeof id !== "string" || typeof projectId !== "string") {
    return {
      props: {
        server: null,
        project: null,
        containerMetrics: { cpu: [], memory: [], network: [] },
        logs: [],
        events: [],
      },
    };
  }

  try {
    const [detail, metrics, logs, events] = await Promise.all([
      fetchProjectDetail(id, projectId),
      fetchProjectMetrics(id, projectId),
      fetchProjectLogs(id, projectId),
      fetchProjectEvents(id, projectId),
    ]);

    return {
      props: {
        server: detail.server,
        project: detail.project,
        containerMetrics: metrics.metrics,
        logs: logs.logs,
        events: events.events,
      },
    };
  } catch {
    return {
      props: {
        server: null,
        project: null,
        containerMetrics: { cpu: [], memory: [], network: [] },
        logs: [],
        events: [],
      },
    };
  }
};
