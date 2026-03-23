import assert from "node:assert/strict";
import test from "node:test";

import {
  filterPendingSystems,
  getConnectedSystemRedirectPath,
  getResumeFlowStep,
  shouldPreserveAddSystemProgress,
  type SystemOnboardingRecord,
} from "../lib/add-system-flow.ts";

function buildSystem(overrides: Partial<SystemOnboardingRecord> = {}): SystemOnboardingRecord {
  return {
    id: "7c3dc5c3-7c7d-44e0-b961-3b127abfa8a0",
    server_id: "f799eae4-e91a-4d91-b050-abd234307776",
    agent_id: "0a2c1560-b363-46be-b42f-74a6ca2e9888",
    name: "Validation VPS",
    description: "Test system",
    status: "awaiting_connection",
    created_at: "2026-03-23T00:00:00Z",
    connected_at: null,
    ...overrides,
  };
}

test("filterPendingSystems keeps only waiting onboardings for the resume list", () => {
  const systems = [
    buildSystem({ id: "onb-pending", status: "awaiting_connection" }),
    buildSystem({ id: "onb-connected", status: "connected", connected_at: "2026-03-23T00:10:00Z" }),
  ];

  assert.deepEqual(filterPendingSystems(systems).map((system) => system.id), ["onb-pending"]);
});

test("getResumeFlowStep sends pending systems back into the waiting state", () => {
  assert.equal(getResumeFlowStep(buildSystem({ status: "awaiting_connection" })), "waiting");
});

test("getResumeFlowStep keeps connected systems out of the waiting state", () => {
  assert.equal(getResumeFlowStep(buildSystem({ status: "connected" })), "details");
});

test("shouldPreserveAddSystemProgress only preserves an awaiting connection wait state", () => {
  assert.equal(
    shouldPreserveAddSystemProgress("waiting", buildSystem({ status: "awaiting_connection" })),
    true
  );
  assert.equal(
    shouldPreserveAddSystemProgress("waiting", buildSystem({ status: "connected" })),
    false
  );
  assert.equal(
    shouldPreserveAddSystemProgress("config", buildSystem({ status: "awaiting_connection" })),
    false
  );
});

test("getConnectedSystemRedirectPath only redirects once the system connects", () => {
  assert.equal(getConnectedSystemRedirectPath(buildSystem({ status: "awaiting_connection" })), null);
  assert.equal(
    getConnectedSystemRedirectPath(
      buildSystem({
        status: "connected",
        server_id: "5d5c6874-7587-4f72-9298-122e13fa56a4",
      })
    ),
    "/servers/5d5c6874-7587-4f72-9298-122e13fa56a4"
  );
});

test("getConnectedSystemRedirectPath stays null when the route-facing server id is missing", () => {
  assert.equal(
    getConnectedSystemRedirectPath(
      buildSystem({
        status: "connected",
        server_id: "",
      })
    ),
    null
  );
});
