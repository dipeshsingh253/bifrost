import assert from "node:assert/strict";
import test from "node:test";

import {
  buildContainerMetricSeries,
  filterContainerMetricSeries,
  listContainerMetricKeys,
} from "../lib/container-metrics.ts";

test("buildContainerMetricSeries maps stable ids to display labels", () => {
  const series = buildContainerMetricSeries([
    { id: "ctr-api-1", name: "service-a-backend-1" },
    { id: "ctr-worker-1", name: "service-a-worker-1" },
    { id: "ctr-api-1", name: "service-a-backend-1" },
  ]);

  assert.deepEqual(series, [
    { key: "ctr-api-1", label: "service-a-backend-1" },
    { key: "ctr-worker-1", label: "service-a-worker-1" },
  ]);
});

test("filterContainerMetricSeries matches against display labels", () => {
  const series = [
    { key: "ctr-api-1", label: "service-a-backend-1" },
    { key: "ctr-worker-1", label: "service-a-worker-1" },
  ];

  assert.deepEqual(filterContainerMetricSeries(series, "backend"), [
    { key: "ctr-api-1", label: "service-a-backend-1" },
  ]);
});

test("listContainerMetricKeys ignores timestamps and returns all container ids present", () => {
  const keys = listContainerMetricKeys([
    {
      timestamp: "2026-03-23T00:00:00Z",
      "ctr-api-1": 6.8,
    },
    {
      timestamp: "2026-03-23T00:01:00Z",
      "ctr-api-1": 7.1,
      "ctr-worker-1": 2.3,
    },
  ]);

  assert.deepEqual(keys, ["ctr-api-1", "ctr-worker-1"]);
});
