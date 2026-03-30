import { useMemo } from "react";
import Link from "next/link";
import { ArrowRight, Box, Boxes, Activity } from "lucide-react";
import type { Service } from "@/lib/types";
import {
  serverContainerPath,
  serverContainersPath,
  serverProjectPath,
  serverProjectsPath,
} from "@/lib/monitoring-routes";

function StatCard({ label, value, icon: Icon, alert }: { label: string; value: number | string; icon: any; alert?: boolean }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4 flex flex-col gap-1">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Icon className="h-4 w-4" />
        <span>{label}</span>
      </div>
      <div className={`text-2xl font-semibold ${alert ? "text-destructive" : "text-foreground"}`}>
        {value}
      </div>
    </div>
  );
}

export function ServicesSection({ serverId, services }: { serverId: string; services: Service[] }) {
  const { projects, standalones, stats } = useMemo(() => {
    const p = services.filter((s) => s.compose_project !== "");
    const st = services.filter((s) => s.compose_project === "");
    const unhealthyProjects = p.filter((s) => s.status !== "running").length;
    const unhealthyStandalones = st.filter((s) => s.status !== "running").length;
    
    const sortedProjects = [...p].sort((a, b) => {
      if (a.status !== "running" && b.status === "running") return -1;
      if (a.status === "running" && b.status !== "running") return 1;
      const cpuA = a.containers.reduce((acc, c) => acc + c.cpu_usage_pct, 0);
      const cpuB = b.containers.reduce((acc, c) => acc + c.cpu_usage_pct, 0);
      if (Math.abs(cpuA - cpuB) > 0.1) return cpuB - cpuA;
      return b.last_log_timestamp.localeCompare(a.last_log_timestamp);
    });

    const sortedStandalones = [...st].sort((a, b) => {
      const statA = a.containers[0]?.status || a.status;
      const statB = b.containers[0]?.status || b.status;
      if (statA !== "running" && statB === "running") return -1;
      if (statA === "running" && statB !== "running") return 1;
      const cpuA = a.containers[0]?.cpu_usage_pct || 0;
      const cpuB = b.containers[0]?.cpu_usage_pct || 0;
      return cpuB - cpuA;
    });

    return {
      projects: sortedProjects,
      standalones: sortedStandalones,
      stats: {
        totalProjects: p.length,
        totalContainers: services.reduce((acc, s) => acc + s.container_count, 0),
        unhealthyCount: unhealthyProjects + unhealthyStandalones,
      }
    };
  }, [services]);

  const pPreview = projects.slice(0, 3);
  const stPreview = standalones.slice(0, 3);

  return (
    <div className="mt-8 flex flex-col gap-8">
      <div>
        <h2 className="text-xl font-semibold text-foreground mb-4">Services</h2>
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <StatCard label="Total Projects" value={stats.totalProjects} icon={Boxes} />
          <StatCard label="Total Containers" value={stats.totalContainers} icon={Box} />
          <StatCard label="Unhealthy" value={stats.unhealthyCount} icon={Activity} alert={stats.unhealthyCount > 0} />
        </div>
      </div>

      {/* Projects Section */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium text-foreground">Projects ({projects.length})</h3>
          {projects.length > 3 && (
            <Link
              href={serverProjectsPath(serverId)}
              className="group flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              View All <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
            </Link>
          )}
        </div>

        {projects.length === 0 ? (
          <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
            No docker-compose projects found
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            {pPreview.map((p) => {
              const totalCpu = p.containers.reduce((acc, c) => acc + c.cpu_usage_pct, 0);
              const totalMem = p.containers.reduce((acc, c) => acc + c.memory_mb, 0);
              return (
                <Link
                  key={p.id}
                  href={serverProjectPath(serverId, p.id)}
                  className="group flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-md border border-border bg-card px-4 py-3 hover:bg-accent/50 transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-3">
                    <span
                      className={`h-2.5 w-2.5 rounded-full shrink-0 ${
                        p.status === "running" ? "bg-success" : "bg-destructive"
                      }`}
                    />
                    <div>
                      <div className="font-medium text-foreground group-hover:text-[hsl(140_50%_48%)] transition-colors">
                        {p.name}
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5">
                        {p.container_count} containers
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4 text-sm tabular-nums text-muted-foreground">
                    <span className="w-20 text-right">CPU {totalCpu.toFixed(1)}%</span>
                    <span className="w-24 text-right">RAM {totalMem.toFixed(0)} MB</span>
                    <span className="w-20 text-right uppercase text-[10px] tracking-wider border border-border bg-muted px-2 py-0.5 rounded text-center">
                      {p.status}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </div>

      {/* Standalone Containers Section */}
      {standalones.length > 0 && (
        <div className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-medium text-foreground">Standalone Containers ({standalones.length})</h3>
            {standalones.length > 3 && (
              <Link
                href={serverContainersPath(serverId)}
                className="group flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                View All <ArrowRight className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5" />
              </Link>
            )}
          </div>

          <div className="flex flex-col gap-2">
            {stPreview.map((s) => {
              const c = s.containers[0];
              if (!c) return null;
              return (
                <Link
                  key={s.id}
                  href={serverContainerPath(serverId, c.id)}
                  className="group flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-md border border-border bg-card px-4 py-3 hover:bg-accent/50 transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-3">
                    <span
                      className={`h-2.5 w-2.5 rounded-full shrink-0 ${
                        c.status === "running" ? "bg-success" : "bg-destructive"
                      }`}
                    />
                    <div>
                      <div className="font-medium text-foreground group-hover:text-[hsl(140_50%_48%)] transition-colors">
                        {c.name}
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5 max-w-[200px] truncate" title={c.ports.join(", ")}>
                        {c.ports.length > 0 ? `Ports: ${c.ports.join(", ")}` : "No exposed ports"}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4 text-sm tabular-nums text-muted-foreground">
                    <span className="w-20 text-right">CPU {c.cpu_usage_pct.toFixed(1)}%</span>
                    <span className="w-24 text-right">RAM {c.memory_mb.toFixed(0)} MB</span>
                    <span className="w-20 text-right uppercase text-[10px] tracking-wider border border-border bg-muted px-2 py-0.5 rounded text-center">
                      {c.status}
                    </span>
                  </div>
                </Link>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
