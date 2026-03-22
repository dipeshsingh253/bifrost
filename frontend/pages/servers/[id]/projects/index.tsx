import Head from "next/head";
import type { GetServerSideProps } from "next";
import Link from "next/link";
import { useState } from "react";
import { Layout } from "@/components/Layout";
import { ArrowLeft, Search, Boxes } from "lucide-react";
import { fetchProjects } from "@/lib/api";
import type { Server, Service } from "@/lib/types";

type ProjectsListProps = {
  server: Server | null;
  projects: Service[];
};

export default function ProjectsList({ server, projects }: ProjectsListProps) {
  const [search, setSearch] = useState("");

  if (!server) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          Server not found.
        </div>
      </Layout>
    );
  }
  
  const filteredProjects = projects.filter(p => p.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <>
      <Head>
        <title>Projects · {server.name} · Bifrost</title>
      </Head>
      <Layout>
        <div className="mb-6 flex flex-col gap-4">
          <nav className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
            <Link href={`/servers/${server.id}`} className="hover:text-foreground transition-colors inline-flex items-center gap-1">
              <ArrowLeft className="h-3.5 w-3.5" />
              {server.name}
            </Link>
            <span className="text-border">/</span>
            <span className="text-foreground font-medium">All Projects</span>
          </nav>

          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <h1 className="text-2xl font-semibold text-foreground flex items-center gap-2">
              <Boxes className="h-6 w-6 text-muted-foreground" />
              Projects ({projects.length})
            </h1>
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search projects..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="h-9 w-full sm:w-64 rounded-md border border-border bg-background pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        <div className="grid gap-3">
          {filteredProjects.length === 0 ? (
            <div className="rounded-lg border border-border bg-card p-12 text-center text-muted-foreground">
              No projects found matching &quot;{search}&quot;.
            </div>
          ) : (
            filteredProjects.map((p) => {
              const totalCpu = p.containers.reduce((acc, c) => acc + c.cpu_usage_pct, 0);
              const totalMem = p.containers.reduce((acc, c) => acc + c.memory_mb, 0);
              const unhealthy = p.containers.filter(c => c.health === "unhealthy").length;
              return (
                <Link
                  key={p.id}
                  href={`/servers/${server.id}/projects/${p.id}`}
                  className="group flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-lg border border-border bg-card p-4 hover:bg-accent/50 transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-4">
                    <span
                      className={`h-3 w-3 rounded-full shrink-0 ${
                        p.status === "running" ? "bg-success" : "bg-destructive"
                      }`}
                    />
                    <div>
                      <div className="font-semibold text-foreground text-base group-hover:text-[hsl(140_50%_48%)] transition-colors flex items-center gap-2">
                        {p.name}
                      </div>
                      <div className="text-sm text-muted-foreground mt-1 flex items-center gap-3">
                        <span>{p.container_count} containers</span>
                        <span>Restarts: {p.restart_count}</span>
                        {unhealthy > 0 && <span className="text-xs text-destructive font-normal bg-destructive/10 px-1.5 py-0.5 rounded">({unhealthy} unhealthy)</span>}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center justify-between sm:justify-end sm:gap-6 text-sm tabular-nums text-muted-foreground sm:bg-background sm:px-4 sm:py-2 sm:rounded-md sm:border border-border/50">
                    <div className="flex flex-col items-start sm:items-end">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">CPU</span>
                      <span className="font-medium text-foreground">{totalCpu.toFixed(1)}%</span>
                    </div>
                    <div className="flex flex-col items-start sm:items-end w-20 sm:w-auto">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">RAM</span>
                      <span className="font-medium text-foreground">{totalMem.toFixed(0)} MB</span>
                    </div>
                    <div className="flex flex-col items-end w-24">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">Status</span>
                      <span className="uppercase text-[11px] tracking-wider text-foreground">{p.status}</span>
                    </div>
                  </div>
                </Link>
              );
            })
          )}
        </div>
      </Layout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<ProjectsListProps> = async (context) => {
  const id = context.params?.id;
  if (typeof id !== "string") {
    return { props: { server: null, projects: [] } };
  }

  try {
    const data = await fetchProjects(id);
    return { props: { server: data.server, projects: data.projects } };
  } catch {
    return { props: { server: null, projects: [] } };
  }
};
