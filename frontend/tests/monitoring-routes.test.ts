import assert from "node:assert/strict";
import test from "node:test";

import {
  serverApiPath,
  serverContainerApiPath,
  serverContainerPath,
  serverPath,
  serverProjectApiPath,
  serverProjectPath,
  serverProjectsApiPath,
  serverProjectsPath,
  serverStandaloneContainersApiPath,
} from "../lib/monitoring-routes.ts";

test("monitoring route builders keep browser paths on UUID route ids", () => {
  const serverRouteID = "5d5c6874-7587-4f72-9298-122e13fa56a4";
  const projectRouteID = "adf7a88d-a8c7-4d66-a447-c8a1e69dbfc4";
  const containerRouteID = "64b6bfaf-f76b-4705-9202-a56b8ce73b6f";

  assert.equal(serverPath(serverRouteID), `/servers/${serverRouteID}`);
  assert.equal(serverProjectsPath(serverRouteID), `/servers/${serverRouteID}/projects`);
  assert.equal(serverProjectPath(serverRouteID, projectRouteID), `/servers/${serverRouteID}/projects/${projectRouteID}`);
  assert.equal(serverContainerPath(serverRouteID, containerRouteID), `/servers/${serverRouteID}/containers/${containerRouteID}`);
});

test("monitoring api route builders encode routed identifiers safely", () => {
  const serverRouteID = "server/with/slash";
  const projectRouteID = "project value";
  const containerRouteID = "container?value";

  assert.equal(serverApiPath(serverRouteID), "/api/v1/servers/server%2Fwith%2Fslash");
  assert.equal(serverProjectsApiPath(serverRouteID), "/api/v1/servers/server%2Fwith%2Fslash/projects");
  assert.equal(
    serverProjectApiPath(serverRouteID, projectRouteID),
    "/api/v1/servers/server%2Fwith%2Fslash/projects/project%20value"
  );
  assert.equal(
    serverContainerApiPath(serverRouteID, containerRouteID),
    "/api/v1/servers/server%2Fwith%2Fslash/containers/container%3Fvalue"
  );
  assert.equal(
    serverStandaloneContainersApiPath(serverRouteID),
    "/api/v1/servers/server%2Fwith%2Fslash/containers?standalone=true"
  );
});
