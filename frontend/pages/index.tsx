import Head from "next/head";
import type { GetServerSideProps } from "next";
import Link from "next/link";
import { useState, useMemo } from "react";
import { Layout } from "@/components/Layout";
import { fetchServerList } from "@/lib/api";
import type { Server, Service } from "@/lib/types";
import { Bell, MoreHorizontal, RefreshCw, ChevronUp, ChevronDown, Monitor, Cpu, MemoryStick, HardDrive, Wifi, Radio } from "lucide-react";

// ── Meter component ──

function progressColor(value: number) {
  if (value >= 75) return "bg-warning";
  if (value >= 55) return "bg-[hsl(80_60%_45%)]"; // lime
  return "bg-success";
}

function Meter({ value }: { value: number }) {
  return (
    <div className="flex items-center gap-2.5">
      <span className="w-11 text-sm tabular-nums text-foreground">{value.toFixed(1)}%</span>
      <div className="h-2 flex-1 rounded-full bg-muted">
        <div
          className={`h-2 rounded-full transition-all ${progressColor(value)}`}
          style={{ width: `${Math.min(value, 100)}%` }}
        />
      </div>
    </div>
  );
}

// ── Sort helpers ──

type SortKey = "name" | "cpu" | "memory" | "disk" | "net";
type SortDir = "asc" | "desc";

function SortHeader({
  label,
  icon: Icon,
  sortKey,
  currentSort,
  currentDir,
  onSort,
}: {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  sortKey: SortKey;
  currentSort: SortKey;
  currentDir: SortDir;
  onSort: (key: SortKey) => void;
}) {
  const active = currentSort === sortKey;
  return (
    <th
      className="px-4 py-3 cursor-pointer select-none hover:text-foreground transition-colors"
      onClick={() => onSort(sortKey)}
    >
      <div className="flex items-center gap-1.5">
        <Icon className="h-3.5 w-3.5" />
        <span>{label}</span>
        {active && (
          currentDir === "asc" ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />
        )}
      </div>
    </th>
  );
}

// ── Main page ──

type ServerListItem = {
  server: Server;
  services: Service[];
};

type HomeProps = {
  bundles: ServerListItem[];
};

export default function Home({ bundles }: HomeProps) {
  const [filter, setFilter] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const filtered = useMemo(() => {
    let items = [...bundles];

    // Filter
    if (filter) {
      const q = filter.toLowerCase();
      items = items.filter((b) => b.server.name.toLowerCase().includes(q));
    }

    // Sort
    items.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case "name":
          cmp = a.server.name.localeCompare(b.server.name);
          break;
        case "cpu":
          cmp = a.server.cpu_usage_pct - b.server.cpu_usage_pct;
          break;
        case "memory":
          cmp = a.server.memory_usage_pct - b.server.memory_usage_pct;
          break;
        case "disk":
          cmp = a.server.disk_usage_pct - b.server.disk_usage_pct;
          break;
        case "net":
          cmp =
            a.server.network_rx_mb + a.server.network_tx_mb -
            (b.server.network_rx_mb + b.server.network_tx_mb);
          break;
      }
      return sortDir === "asc" ? cmp : -cmp;
    });

    return items;
  }, [bundles, filter, sortKey, sortDir]);

  return (
    <>
      <Head>
        <title>Bifrost</title>
        <meta name="description" content="Bifrost server monitoring dashboard" />
      </Head>
      <Layout>
        <div className="rounded-lg border border-border bg-card">
          {/* Header */}
          <div className="flex flex-col gap-4 px-5 py-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h1 className="text-xl font-semibold text-foreground">All Systems</h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Updated in real time. Click on a system to view information.
              </p>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="text"
                placeholder="Filter..."
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring min-w-[140px]"
              />
              <button className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground hover:bg-accent transition-colors">
                Columns
              </button>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="min-w-full text-left">
              <thead className="border-t border-b border-border bg-table-header text-xs uppercase tracking-wider text-muted-foreground">
                <tr>
                  <SortHeader label="System" icon={Monitor} sortKey="name" currentSort={sortKey} currentDir={sortDir} onSort={handleSort} />
                  <SortHeader label="CPU" icon={Cpu} sortKey="cpu" currentSort={sortKey} currentDir={sortDir} onSort={handleSort} />
                  <SortHeader label="Memory" icon={MemoryStick} sortKey="memory" currentSort={sortKey} currentDir={sortDir} onSort={handleSort} />
                  <SortHeader label="Disk" icon={HardDrive} sortKey="disk" currentSort={sortKey} currentDir={sortDir} onSort={handleSort} />
                  <SortHeader label="Net" icon={Wifi} sortKey="net" currentSort={sortKey} currentDir={sortDir} onSort={handleSort} />
                  <th className="px-4 py-3">
                    <div className="flex items-center gap-1.5">
                      <Radio className="h-3.5 w-3.5" />
                      <span>Agent</span>
                    </div>
                  </th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filtered.map((bundle) => (
                  <ServerRow key={bundle.server.id} bundle={bundle} />
                ))}
                {filtered.length === 0 && (
                  <tr>
                    <td colSpan={7} className="px-5 py-12 text-center text-muted-foreground">
                      No systems found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </Layout>
    </>
  );
}

function ServerRow({ bundle }: { bundle: ServerListItem }) {
  const { server } = bundle;
  const isUp = server.status === "up";

  return (
    <tr className="group cursor-pointer transition-colors hover:bg-accent/50">
      <td className="px-4 py-3.5">
        <Link href={`/servers/${server.id}`} className="flex items-center gap-2.5">
          <span
            className={`h-2.5 w-2.5 rounded-full shrink-0 ${isUp ? "bg-success" : "bg-destructive"}`}
          />
          <span className="font-medium text-foreground group-hover:text-[hsl(140_50%_48%)] transition-colors">
            {server.name}
          </span>
          <RefreshCw className="h-3 w-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
        </Link>
      </td>
      <td className="px-4 py-3.5 min-w-[160px]">
        <Meter value={server.cpu_usage_pct} />
      </td>
      <td className="px-4 py-3.5 min-w-[160px]">
        <Meter value={server.memory_usage_pct} />
      </td>
      <td className="px-4 py-3.5 min-w-[160px]">
        <Meter value={server.disk_usage_pct} />
      </td>
      <td className="px-4 py-3.5 text-sm tabular-nums text-foreground">
        {(server.network_rx_mb + server.network_tx_mb).toFixed(2)} MB/s
      </td>
      <td className="px-4 py-3.5 text-sm text-foreground">
        <span className="inline-flex items-center gap-1.5">
          <span
            className={`h-2 w-2 rounded-full ${isUp ? "bg-success" : "bg-muted-foreground"}`}
          />
          {server.agent_version}
        </span>
      </td>
      <td className="px-4 py-3.5">
        <div className="flex items-center gap-1">
          <button className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <Bell className="h-4 w-4" />
          </button>
          <button className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors">
            <MoreHorizontal className="h-4 w-4" />
          </button>
        </div>
      </td>
    </tr>
  );
}

export const getServerSideProps: GetServerSideProps<HomeProps> = async () => {
  try {
    const bundles = await fetchServerList();
    return { props: { bundles } };
  } catch {
    return { props: { bundles: [] } };
  }
};
