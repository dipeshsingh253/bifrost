import { useId } from "react";
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
import type { ContainerMetricPoint } from "@/lib/types";

type Props = {
  data: ContainerMetricPoint[];
  containerNames: string[];
  unit?: string;
  height?: number;
  tickFormatter?: (value: number) => string;
};

const PALETTE = [
  "hsl(220 70% 50%)",   // blue
  "hsl(160 60% 45%)",   // teal
  "hsl(30 80% 55%)",    // orange
  "hsl(280 65% 60%)",   // purple
  "hsl(340 75% 55%)",   // pink
  "hsl(190 70% 50%)",   // cyan
  "hsl(100 50% 45%)",   // green
  "hsl(45 90% 55%)",    // yellow
  "hsl(0 65% 55%)",     // red
  "hsl(250 60% 60%)",   // indigo
];

function formatTime(timestamp: string) {
  const d = new Date(timestamp);
  return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

export function StackedAreaChart({
  data,
  containerNames,
  unit = "",
  height = 180,
  tickFormatter,
}: Props) {
  const chartId = useId();

  const defaultTickFormatter = (v: number) => {
    if (unit === "%") return `${v.toFixed(1)}%`;
    if (unit === " MB") return `${v.toFixed(0)} MB`;
    if (unit === " MB/s") return `${v.toFixed(2)} MB/s`;
    return String(Math.round(v * 100) / 100);
  };

  const formatter = tickFormatter ?? defaultTickFormatter;

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
        <defs>
          {containerNames.map((name, i) => (
            <linearGradient
              key={name}
              id={`${chartId}-${i}`}
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop offset="5%" stopColor={PALETTE[i % PALETTE.length]} stopOpacity={0.35} />
              <stop offset="95%" stopColor={PALETTE[i % PALETTE.length]} stopOpacity={0.02} />
            </linearGradient>
          ))}
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
          width={48}
          tickFormatter={formatter}
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
          formatter={(value: number, name: string) => [`${value.toFixed(2)}${unit}`, name]}
        />
        <Legend
          wrapperStyle={{ fontSize: 11, color: "hsl(220 4% 67%)" }}
          iconType="circle"
          iconSize={8}
        />
        {containerNames.map((name, i) => (
          <Area
            key={name}
            type="monotone"
            dataKey={name}
            stackId="1"
            stroke={PALETTE[i % PALETTE.length]}
            strokeWidth={1}
            fill={`url(#${chartId}-${i})`}
            dot={false}
          />
        ))}
      </AreaChart>
    </ResponsiveContainer>
  );
}
