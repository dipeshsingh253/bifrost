export type InstallMode = "docker" | "systemd";

export type FlowStep = "details" | "config" | "install" | "waiting";

export type SystemOnboardingRecord = {
  id: string;
  server_id: string;
  agent_id: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  connected_at?: string | null;
  api_key?: string;
  enrollment_token?: string;
  backend_url?: string;
  install_script_url?: string;
  docker_run_command?: string;
  systemd_install_command?: string;
  config_yaml?: string;
};

export function filterPendingSystems(systems: SystemOnboardingRecord[]): SystemOnboardingRecord[] {
  return systems.filter((system) => system.status === "awaiting_connection");
}

export function getResumeFlowStep(system: Pick<SystemOnboardingRecord, "status">): FlowStep {
  return system.status === "connected" ? "details" : "waiting";
}

export function shouldPreserveAddSystemProgress(
  step: FlowStep,
  activeSystem: Pick<SystemOnboardingRecord, "status"> | null
): boolean {
  return step === "waiting" && activeSystem?.status === "awaiting_connection";
}

export function getConnectedSystemRedirectPath(
  system: Pick<SystemOnboardingRecord, "status" | "server_id">
): string | null {
  if (system.status !== "connected" || !system.server_id) {
    return null;
  }

  return `/servers/${encodeURIComponent(system.server_id)}`;
}
