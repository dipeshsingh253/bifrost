import assert from "node:assert/strict";
import test from "node:test";

import {
  getContainerDetailNotFoundKind,
  getProjectDetailNotFoundKind,
} from "../lib/monitoring-not-found.ts";

function buildApiError(code: string) {
  return { code };
}

test("getProjectDetailNotFoundKind preserves server vs project misses", () => {
  assert.equal(getProjectDetailNotFoundKind(buildApiError("SERVER_NOT_FOUND")), "server");
  assert.equal(getProjectDetailNotFoundKind(buildApiError("PROJECT_NOT_FOUND")), "project");
  assert.equal(getProjectDetailNotFoundKind(buildApiError("INTERNAL_ERROR")), null);
});

test("getProjectDetailNotFoundKind keeps stale legacy project routes mapped to project misses", () => {
  const staleLegacyProjectRouteError = buildApiError("PROJECT_NOT_FOUND");

  assert.equal(getProjectDetailNotFoundKind(staleLegacyProjectRouteError), "project");
});

test("getContainerDetailNotFoundKind preserves server vs container misses", () => {
  assert.equal(getContainerDetailNotFoundKind(buildApiError("SERVER_NOT_FOUND")), "server");
  assert.equal(getContainerDetailNotFoundKind(buildApiError("CONTAINER_NOT_FOUND")), "container");
  assert.equal(getContainerDetailNotFoundKind(buildApiError("INTERNAL_ERROR")), null);
});

test("getContainerDetailNotFoundKind keeps stale legacy container routes mapped to container misses", () => {
  const staleLegacyContainerRouteError = buildApiError("CONTAINER_NOT_FOUND");

  assert.equal(getContainerDetailNotFoundKind(staleLegacyContainerRouteError), "container");
});
