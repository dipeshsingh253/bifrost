import { useId } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import type { MetricPoint } from "@/lib/types";

type Props = {
  data: MetricPoint[];
  unit?: string;
  color?: string;
  gradientId?: string;
  height?: number;
  domain?: [number | string, number | string];
  tickFormatter?: (value: number) => string;
};

const COLORS = {
  green: { stroke: "hsl(140 50% 48%)", fill: "hsl(140 50% 48%)" },
  blue: { stroke: "hsl(220 70% 50%)", fill: "hsl(220 70% 50%)" },
  purple: { stroke: "hsl(280 65% 60%)", fill: "hsl(280 65% 60%)" },
  amber: { stroke: "hsl(38 92% 50%)", fill: "hsl(38 92% 50%)" },
};

function formatTime(timestamp: string) {
  const d = new Date(timestamp);
  return d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

export function BifrostAreaChart({
  data,
  unit = "",
  color = "green",
  gradientId,
  height = 180,
  domain,
  tickFormatter,
}: Props) {
  const reactId = useId();
  const id = gradientId ?? `gradient-${color}-${reactId}`;
  const palette = COLORS[color as keyof typeof COLORS] ?? COLORS.green;

  const defaultTickFormatter = (v: number) => {
    if (unit === "%") return `${v.toFixed(0)}%`;
    if (unit === " MB/s") return `${v.toFixed(2)} MB/s`;
    if (unit === " GB") return `${v.toFixed(1)} GB`;
    return String(v);
  };

  const formatter = tickFormatter ?? defaultTickFormatter;

  return (
    <ResponsiveContainer width="100%" height={height}>
      <AreaChart data={data} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={palette.fill} stopOpacity={0.3} />
            <stop offset="95%" stopColor={palette.fill} stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <CartesianGrid
          strokeDasharray="3 3"
          stroke="hsl(220 3% 17%)"
          vertical={false}
        />
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
          domain={domain ?? ["auto", "auto"]}
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
          formatter={(value: number) => [`${value.toFixed(2)}${unit}`, ""]}
        />
        <Area
          type="monotone"
          dataKey="value"
          stroke={palette.stroke}
          strokeWidth={1.5}
          fill={`url(#${id})`}
          dot={false}
          activeDot={{ r: 3, stroke: palette.stroke, strokeWidth: 2 }}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}
