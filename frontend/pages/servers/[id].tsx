import Head from "next/head";
import type { GetServerSideProps } from "next";
import { useId } from "react";
import { useState } from "react";
import { Layout } from "@/components/Layout";
import { fetchServerBundle } from "@/lib/api";
import { InfoBar } from "@/components/server/InfoBar";
import { TimeRangeSelect } from "@/components/server/TimeRangeSelect";
import { ChartCard } from "@/components/charts/ChartCard";
import { BifrostAreaChart } from "@/components/charts/AreaChart";
import { StackedAreaChart } from "@/components/charts/StackedAreaChart";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { ServicesSection } from "@/components/server/ServicesSection";
import type { ServerBundle } from "@/lib/types";

type ServerDetailProps = {
  bundle: ServerBundle | null;
};

export default function ServerDetail({ bundle }: ServerDetailProps) {
  const [timeRange, setTimeRange] = useState("1h");
  const [grid, setGrid] = useState(true);
  const [containerFilter, setContainerFilter] = useState("");

  if (!bundle) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          System not found.
        </div>
      </Layout>
    );
  }

  const { server, metrics, containerMetrics, services } = bundle;

  const metricByKey = new Map(metrics.map((m) => [m.key, m]));
  const cpuData = metricByKey.get("cpu_usage_pct");
  const memData = metricByKey.get("memory_usage_pct");
  const diskData = metricByKey.get("disk_usage_pct");
  const netRxData = metricByKey.get("network_rx_mb");
  const netTxData = metricByKey.get("network_tx_mb");
  const diskReadData = metricByKey.get("disk_read_mb");
  const diskWriteData = metricByKey.get("disk_write_mb");

  // Container names for stacked charts
  const allContainerNames = Object.keys(containerMetrics.cpu[0] ?? {}).filter(
    (k) => k !== "timestamp"
  );
  const filteredContainerNames = containerFilter
    ? allContainerNames.filter((n) =>
        n.toLowerCase().includes(containerFilter.toLowerCase())
      )
    : allContainerNames;

  const containerFilterInput = allContainerNames.length > 0 ? (
    <input
      type="text"
      placeholder="Filter..."
      value={containerFilter}
      onChange={(e) => setContainerFilter(e.target.value)}
      className="rounded-md border border-border bg-background px-2.5 py-1 text-xs text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-ring w-24"
    />
  ) : null;

  const gridCols = grid ? "xl:grid-cols-2" : "";

  return (
    <>
      <Head>
        <title>{server.name} · Bifrost</title>
      </Head>
      <Layout>
        {/* Navigation + Info */}
        <div className="mb-4">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors mb-3"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            All Systems
          </Link>

          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-foreground">{server.name}</h1>
              <div className="mt-2">
                <InfoBar server={server} />
              </div>
            </div>
            <div className="shrink-0">
              <TimeRangeSelect
                selectedTime={timeRange}
                onTimeChange={setTimeRange}
                grid={grid}
                onGridToggle={() => setGrid((g) => !g)}
              />
            </div>
          </div>
        </div>

        {/* ── Chart Grid ── */}
        <div className={`grid gap-4 ${gridCols}`}>
          {/* CPU Usage */}
          {cpuData && (
            <ChartCard title="CPU Usage" description="Average system-wide CPU utilization">
              <BifrostAreaChart
                data={cpuData.points}
                unit="%"
                color="green"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(0)}%`}
              />
            </ChartCard>
          )}

          {/* Docker CPU Usage */}
          {filteredContainerNames.length > 0 && (
            <ChartCard
              title="Docker CPU Usage"
              description="Average CPU utilization of containers"
              cornerEl={containerFilterInput}
            >
              <StackedAreaChart
                data={containerMetrics.cpu}
                containerNames={filteredContainerNames}
                unit="%"
                tickFormatter={(v) => `${v.toFixed(1)}%`}
              />
            </ChartCard>
          )}

          {/* Memory Usage */}
          {memData && (
            <ChartCard title="Memory Usage" description="Precise utilization at the recorded time">
              <BifrostAreaChart
                data={memData.points}
                unit="%"
                color="green"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(0)}%`}
              />
            </ChartCard>
          )}

          {/* Docker Memory */}
          {filteredContainerNames.length > 0 && (
            <ChartCard
              title="Docker Memory Usage"
              description="Memory usage of docker containers"
              cornerEl={containerFilterInput}
            >
              <StackedAreaChart
                data={containerMetrics.memory}
                containerNames={filteredContainerNames}
                unit=" MB"
                tickFormatter={(v) => `${v.toFixed(0)} MB`}
              />
            </ChartCard>
          )}

          {/* Disk Usage */}
          {diskData && (
            <ChartCard title="Disk Usage" description="Usage of root partition">
              <BifrostAreaChart
                data={diskData.points}
                unit="%"
                color="purple"
                domain={[0, "auto"]}
                tickFormatter={(v) => `${v.toFixed(0)}%`}
              />
            </ChartCard>
          )}

          {/* Disk I/O */}
          {diskReadData && diskWriteData && (
            <ChartCard title="Disk I/O" description="Throughput of root filesystem">
              <DualAreaChart
                data1={diskReadData.points}
                data2={diskWriteData.points}
                label1="Read"
                label2="Write"
                unit=" MB/s"
              />
            </ChartCard>
          )}

          {/* Bandwidth */}
          {netRxData && netTxData && (
            <ChartCard title="Bandwidth" description="Network traffic of public interfaces">
              <DualAreaChart
                data1={netTxData.points}
                data2={netRxData.points}
                label1="Sent"
                label2="Received"
                unit=" MB/s"
              />
            </ChartCard>
          )}

          {/* Docker Network I/O */}
          {filteredContainerNames.length > 0 && (
            <ChartCard
              title="Docker Network I/O"
              description="Network traffic of docker containers"
              cornerEl={containerFilterInput}
            >
              <StackedAreaChart
                data={containerMetrics.network}
                containerNames={filteredContainerNames}
                unit=" MB/s"
                tickFormatter={(v) => `${v.toFixed(2)} MB/s`}
              />
            </ChartCard>
          )}
        </div>

        {/* ── Services Section ── */}
        <ServicesSection serverId={server.id as string} services={services} />
      </Layout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<ServerDetailProps> = async (context) => {
  const id = context.params?.id;
  if (typeof id !== "string") {
    return { props: { bundle: null } };
  }

  try {
    const bundle = await fetchServerBundle(id);
    return { props: { bundle } };
  } catch {
    return { props: { bundle: null } };
  }
};

// ── Dual area chart (for Bandwidth and Disk I/O) ──

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from "recharts";
import type { MetricPoint } from "@/lib/types";

function DualAreaChart({
  data1,
  data2,
  label1,
  label2,
  unit,
}: {
  data1: MetricPoint[];
  data2: MetricPoint[];
  label1: string;
  label2: string;
  unit: string;
}) {
  const chartId = useId();

  // Merge two series by timestamp
  const merged = data1.map((point, i) => ({
    timestamp: point.timestamp,
    [label1]: point.value,
    [label2]: data2[i]?.value ?? 0,
  }));

  function formatTime(timestamp: string) {
    const d = new Date(timestamp);
    return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  }

  return (
    <ResponsiveContainer width="100%" height={180}>
      <AreaChart data={merged} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={`${chartId}-1`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="hsl(280 65% 60%)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="hsl(280 65% 60%)" stopOpacity={0.02} />
          </linearGradient>
          <linearGradient id={`${chartId}-2`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="hsl(160 60% 45%)" stopOpacity={0.3} />
            <stop offset="95%" stopColor="hsl(160 60% 45%)" stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="hsl(220 3% 17%)" vertical={false} />
        <XAxis
          dataKey="timestamp"
          tickFormatter={formatTime}
          tick={{ fontSize: 11, fill: "hsl(220 4% 67%)" }}
          axisLine={false}
          tickLine={false}
          minTickGap={40}
        />
        <YAxis
          tick={{ fontSize: 11, fill: "hsl(220 4% 67%)" }}
          axisLine={false}
          tickLine={false}
          width={52}
          tickFormatter={(v) => `${v.toFixed(2)}${unit}`}
        />
        <Tooltip
          contentStyle={{
            background: "hsl(220 5.5% 10.5%)",
            border: "1px solid hsl(220 3% 17%)",
            borderRadius: "0.5rem",
            fontSize: 12,
            color: "hsl(220 2% 97%)",
          }}
          labelFormatter={formatTime}
          formatter={(value: number, name: string) => [`${value.toFixed(3)}${unit}`, name]}
        />
        <Legend
          wrapperStyle={{ fontSize: 11, color: "hsl(220 4% 67%)" }}
          iconType="circle"
          iconSize={8}
        />
        <Area
          type="monotone"
          dataKey={label1}
          stroke="hsl(280 65% 60%)"
          strokeWidth={1.5}
          fill={`url(#${chartId}-1)`}
          dot={false}
        />
        <Area
          type="monotone"
          dataKey={label2}
          stroke="hsl(160 60% 45%)"
          strokeWidth={1.5}
          fill={`url(#${chartId}-2)`}
          dot={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}
