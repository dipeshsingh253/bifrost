import Head from "next/head";
import type { GetServerSideProps } from "next";
import Link from "next/link";
import { useState } from "react";
import { Layout } from "@/components/Layout";
import { ArrowLeft, Search, Box } from "lucide-react";
import { fetchStandaloneContainers } from "@/lib/api";
import type { Container, Server } from "@/lib/types";

type ContainersListProps = {
  server: Server | null;
  containers: Container[];
};

export default function ContainersList({ server, containers }: ContainersListProps) {
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
  
  const filteredContainers = containers.filter(c => c.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <>
      <Head>
        <title>Standalone Containers · {server.name} · Bifrost</title>
      </Head>
      <Layout>
        <div className="mb-6 flex flex-col gap-4">
          <nav className="flex flex-wrap items-center gap-1.5 text-sm text-muted-foreground">
            <Link href={`/servers/${server.id}`} className="hover:text-foreground transition-colors inline-flex items-center gap-1">
              <ArrowLeft className="h-3.5 w-3.5" />
              {server.name}
            </Link>
            <span className="text-border">/</span>
            <span className="text-foreground font-medium">Standalone Containers</span>
          </nav>

          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <h1 className="text-2xl font-semibold text-foreground flex items-center gap-2">
              <Box className="h-6 w-6 text-muted-foreground" />
              Standalone Containers ({containers.length})
            </h1>
            <div className="relative">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                placeholder="Search containers..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="h-9 w-full sm:w-64 rounded-md border border-border bg-background pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        <div className="grid gap-3">
          {filteredContainers.length === 0 ? (
            <div className="rounded-lg border border-border bg-card p-12 text-center text-muted-foreground">
              No standalone containers found matching &quot;{search}&quot;.
            </div>
          ) : (
            filteredContainers.map((c) => {
              return (
                <Link
                  key={c.id}
                  href={`/servers/${server.id}/containers/${c.id}`}
                  className="group flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-lg border border-border bg-card p-4 hover:bg-accent/50 transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-4">
                    <span
                      className={`h-3 w-3 rounded-full shrink-0 ${
                        c.status === "running" ? (c.health === "healthy" ? "bg-success" : "bg-warning") : "bg-destructive"
                      }`}
                      title={`Status: ${c.status}`}
                    />
                    <div>
                      <div className="font-semibold text-foreground text-base group-hover:text-[hsl(140_50%_48%)] transition-colors flex items-center gap-2">
                        {c.name}
                      </div>
                      <div className="text-sm text-muted-foreground mt-1 flex flex-wrap items-center gap-3">
                        <span className="max-w-[180px] sm:max-w-md truncate" title={c.image}>{c.image}</span>
                        {c.ports.length > 0 && <span className="bg-muted px-1.5 py-0.5 rounded text-[10px] font-mono border border-border">{c.ports.join(", ")}</span>}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center justify-between sm:justify-end sm:gap-6 text-sm tabular-nums text-muted-foreground sm:bg-background sm:px-4 sm:py-2 sm:rounded-md sm:border border-border/50">
                    <div className="flex flex-col items-start sm:items-end">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">CPU</span>
                      <span className="font-medium text-foreground">{c.cpu_usage_pct.toFixed(1)}%</span>
                    </div>
                    <div className="flex flex-col items-start sm:items-end w-20 sm:w-auto">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">RAM</span>
                      <span className="font-medium text-foreground">{c.memory_mb.toFixed(0)} MB</span>
                    </div>
                    <div className="flex flex-col items-end w-24">
                      <span className="text-[10px] text-muted-foreground/70 uppercase">Status</span>
                      <span className="uppercase text-[11px] tracking-wider text-foreground">{c.status}</span>
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

export const getServerSideProps: GetServerSideProps<ContainersListProps> = async (context) => {
  const id = context.params?.id;
  if (typeof id !== "string") {
    return { props: { server: null, containers: [] } };
  }

  try {
    const data = await fetchStandaloneContainers(id);
    return { props: { server: data.server, containers: data.containers } };
  } catch {
    return { props: { server: null, containers: [] } };
  }
};
