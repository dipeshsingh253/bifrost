import Link from "next/link";
import { ArrowRight, AlertTriangle, Boxes, Layers3 } from "lucide-react";
import { useMemo } from "react";

import {
  serverAllContainersPath,
  serverContainersPath,
  serverProjectPath,
  serverProjectsPath,
} from "@/lib/monitoring-routes";
import type { Container, Service } from "@/lib/types";

type ServerOverviewProps = {
  serverId: string;
  services: Service[];
};

function isContainerUnhealthy(container: Container) {
  return container.status !== "running" || container.health === "unhealthy";
}

function projectStatus(project: Service) {
  return project.status === "running" && project.containers.every((container) => !isContainerUnhealthy(container))
    ? "running"
    : "unhealthy";
}

function statusBadgeClass(status: string) {
  return status === "running"
    ? "border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 text-[hsl(140_50%_48%)]"
    : "border-[hsl(38_92%_50%)]/30 bg-[hsl(38_92%_50%)]/10 text-[hsl(38_92%_50%)]";
}

function healthTextClass(isHealthy: boolean) {
  return isHealthy ? "text-[hsl(140_50%_48%)]" : "text-[hsl(38_92%_50%)]";
}

function unhealthySummary(
  containers: Container[],
  healthyMessage: string,
  multiMessage: (count: number) => string
) {
  const unhealthy = containers.filter(isContainerUnhealthy);
  if (unhealthy.length === 0) {
    return { text: healthyMessage, healthy: true };
  }
  if (unhealthy.length === 1) {
    return { text: `${unhealthy[0].name} unhealthy`, healthy: false };
  }
  if (unhealthy.length === 2) {
    return { text: `${unhealthy[0].name}, ${unhealthy[1].name} unhealthy`, healthy: false };
  }
  return { text: multiMessage(unhealthy.length), healthy: false };
}

function OverviewCard({
  href,
  label,
  value,
  subtext,
  subtextHealthy,
}: {
  href: string;
  label: string;
  value: number;
  subtext: string;
  subtextHealthy: boolean;
}) {
  return (
    <Link
      href={href}
      className="group rounded-lg border border-border bg-card px-4 py-3.5 transition-all duration-200 hover:-translate-y-0.5 hover:border-ring/50 hover:shadow-[0_10px_24px_rgba(0,0,0,0.14)]"
      >
        <div className="flex items-center justify-between gap-3">
          <div className="text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">{label}</div>
          <ArrowRight className="h-4 w-4 text-muted-foreground transition-transform duration-200 group-hover:translate-x-0.5 group-hover:text-foreground" />
        </div>
      <div className="mt-2 text-2xl font-semibold tracking-tight text-foreground">{value}</div>
      <div className={`mt-0.5 text-xs leading-5 ${healthTextClass(subtextHealthy)}`}>{subtext}</div>
    </Link>
  );
}

export function ServerOverview({ serverId, services }: ServerOverviewProps) {
  const {
    projects,
    standaloneContainerCount,
    totalContainers,
    unhealthyContainers,
    projectCardSubtext,
    containersCardSubtext,
    standaloneCardSubtext,
  } = useMemo(() => {
    const projects = services
      .filter((service) => service.compose_project !== "")
      .sort((left, right) => {
        const leftStatus = projectStatus(left);
        const rightStatus = projectStatus(right);
        if (leftStatus !== rightStatus) {
          return leftStatus === "unhealthy" ? -1 : 1;
        }

        if (left.container_count !== right.container_count) {
          return right.container_count - left.container_count;
        }

        return left.name.localeCompare(right.name);
      });

    const standaloneServices = services.filter((service) => service.compose_project === "");
    const projectContainers = projects.flatMap((project) => project.containers);
    const allContainers = services.flatMap((service) => service.containers);
    const standaloneContainers = standaloneServices.flatMap((service) => service.containers);
    const projectCardSubtext = unhealthySummary(
      projectContainers,
      "All projects healthy",
      (count) => `${count} project containers unhealthy`
    );
    const containersCardSubtext = unhealthySummary(
      allContainers,
      "All containers healthy",
      (count) => `${count} containers unhealthy`
    );
    const standaloneCardSubtext =
      standaloneContainers.length === 0
        ? { text: "No standalone containers", healthy: true }
        : unhealthySummary(
            standaloneContainers,
            "All standalone healthy",
            (count) => `${count} standalone unhealthy`
          );

    return {
      projects,
      standaloneContainerCount: standaloneServices.reduce((total, service) => total + service.container_count, 0),
      totalContainers: services.reduce((total, service) => total + service.container_count, 0),
      unhealthyContainers: allContainers.filter(isContainerUnhealthy).length,
      projectCardSubtext,
      containersCardSubtext,
      standaloneCardSubtext,
    };
  }, [services]);

  const summaryCards = [
    {
      href: serverProjectsPath(serverId),
      label: "Projects",
      value: projects.length,
      subtext: projectCardSubtext.text,
      subtextHealthy: projectCardSubtext.healthy,
    },
    {
      href: serverAllContainersPath(serverId),
      label: "Containers",
      value: totalContainers,
      subtext: containersCardSubtext.text,
      subtextHealthy: containersCardSubtext.healthy,
    },
    {
      href: serverContainersPath(serverId),
      label: "Standalone",
      value: standaloneContainerCount,
      subtext: standaloneCardSubtext.text,
      subtextHealthy: standaloneCardSubtext.healthy,
    },
    ...(unhealthyContainers > 0
      ? [
          {
            href: serverAllContainersPath(serverId),
            label: "Unhealthy",
            value: unhealthyContainers,
            subtext: "need attention",
            subtextHealthy: false,
          },
        ]
      : []),
  ];

  return (
    <section className="mb-5">
      <div className="space-y-1.5">
        <h2 className="text-lg font-semibold text-foreground">Overview</h2>
        <p className="text-sm leading-5 text-muted-foreground">
          Quick server health and shortcuts into projects and containers.
        </p>
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
        <div className={`grid gap-3 ${summaryCards.length > 3 ? "sm:grid-cols-2 2xl:grid-cols-4" : "sm:grid-cols-3"}`}>
          {summaryCards.map((card) => (
            <OverviewCard key={card.label} {...card} />
          ))}
        </div>

        <div className="rounded-lg border border-border bg-card p-3.5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h3 className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                Project Breakdown
              </h3>
              <p className="mt-1 text-xs leading-5 text-muted-foreground">
                Open a project to inspect service groups, logs, and container metrics.
              </p>
            </div>
            {projects.length > 0 ? (
              <Link
                href={serverProjectsPath(serverId)}
                className="inline-flex shrink-0 items-center gap-1 text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                View all
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            ) : null}
          </div>

          {projects.length === 0 ? (
            <div className="mt-3 rounded-md border border-dashed border-border bg-background/40 px-4 py-5 text-sm text-muted-foreground">
              No docker-compose projects are reporting on this server yet.
            </div>
          ) : (
            <div className="mt-3 grid gap-2">
              {projects.map((project) => {
                const status = projectStatus(project);
                const unhealthyCount = project.containers.filter(isContainerUnhealthy).length;
                return (
                  <Link
                    key={project.id}
                    href={serverProjectPath(serverId, project.id)}
                    className="group grid min-h-12 grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-3 rounded-md border border-border bg-background/30 px-3 py-2.5 transition-colors duration-200 hover:border-ring/50 hover:bg-accent/35"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border bg-card">
                        <Boxes className="h-4 w-4 text-muted-foreground" />
                      </div>
                      <div className="truncate text-sm font-medium text-foreground transition-colors group-hover:text-[hsl(140_50%_48%)]">
                        {project.name}
                      </div>
                    </div>

                    <div className="text-sm text-muted-foreground">
                      {project.container_count} {project.container_count === 1 ? "container" : "containers"}
                      {unhealthyCount > 0 ? (
                        <span className="text-[hsl(38_92%_50%)]">
                          {" "}
                          • {unhealthyCount} unhealthy
                        </span>
                      ) : null}
                    </div>

                    <div className="flex items-center justify-end gap-2">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.18em] ${statusBadgeClass(status)}`}
                      >
                        {status === "running" ? <Layers3 className="h-3 w-3" /> : <AlertTriangle className="h-3 w-3" />}
                        {status}
                      </span>
                      <ArrowRight className="h-4 w-4 text-muted-foreground transition-transform duration-200 group-hover:translate-x-0.5 group-hover:text-foreground" />
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
