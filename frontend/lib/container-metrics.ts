import type { Container, ContainerMetricPoint, ContainerMetricSeries } from "@/lib/types";

type MinimalContainer = Pick<Container, "id" | "name">;

export function buildContainerMetricSeries(containers: MinimalContainer[]): ContainerMetricSeries[] {
  const seen = new Set<string>();
  const series: ContainerMetricSeries[] = [];

  for (const container of containers) {
    if (seen.has(container.id)) {
      continue;
    }

    seen.add(container.id);
    series.push({
      key: container.id,
      label: container.name,
    });
  }

  return series;
}

export function filterContainerMetricSeries(
  series: ContainerMetricSeries[],
  query: string
): ContainerMetricSeries[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (normalizedQuery === "") {
    return series;
  }

  return series.filter((item) => item.label.toLowerCase().includes(normalizedQuery));
}

export function listContainerMetricKeys(points: ContainerMetricPoint[]): string[] {
  const keys = new Set<string>();

  for (const point of points) {
    for (const key of Object.keys(point)) {
      if (key !== "timestamp") {
        keys.add(key);
      }
    }
  }

  return Array.from(keys);
}
