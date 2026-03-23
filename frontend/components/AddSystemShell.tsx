import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/router";
import { ArrowLeft, Check, Copy, FileCode2, LoaderCircle, RefreshCw, Server, TerminalSquare, X } from "lucide-react";

import { getApiErrorMessage } from "@/lib/api";
import {
  filterPendingSystems,
  getConnectedSystemRedirectPath,
  getResumeFlowStep,
  shouldPreserveAddSystemProgress,
  type FlowStep,
  type InstallMode,
  type SystemOnboardingRecord,
} from "@/lib/add-system-flow";

type AddSystemShellProps = {
  isOpen: boolean;
  onClose: () => void;
};

type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: {
    message?: string;
  };
};

const shellSteps = [
  { key: "details", title: "System Details", icon: Server },
  { key: "config", title: "Quick Install", icon: TerminalSquare },
  { key: "install", title: "Advanced", icon: FileCode2 },
] as const;

const inputClassName =
  "h-11 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring";

async function readPayload<T>(response: Response): Promise<T> {
  const payload = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || !payload.success || payload.data === undefined) {
    throw new Error(payload.error?.message || "Request failed");
  }
  return payload.data;
}

export function AddSystemShell({ isOpen, onClose }: AddSystemShellProps) {
  const router = useRouter();
  const [step, setStep] = useState<FlowStep>("details");
  const [systemName, setSystemName] = useState("");
  const [description, setDescription] = useState("");
  const [preferredInstallMode, setPreferredInstallMode] = useState<InstallMode>("systemd");
  const [selectedInstallTab, setSelectedInstallTab] = useState<InstallMode>("systemd");
  const [activeSystem, setActiveSystem] = useState<SystemOnboardingRecord | null>(null);
  const [pendingSystems, setPendingSystems] = useState<SystemOnboardingRecord[]>([]);
  const [working, setWorking] = useState(false);
  const [isLoadingPending, setIsLoadingPending] = useState(false);
  const [isPolling, setIsPolling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copyFeedback, setCopyFeedback] = useState<string | null>(null);
  const [lastCheckedAt, setLastCheckedAt] = useState<string | null>(null);
  const pollTimerRef = useRef<number | null>(null);
  const preserveProgress = shouldPreserveAddSystemProgress(step, activeSystem);

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current !== null) {
      window.clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
    setIsPolling(false);
  }, []);

  const resetState = useCallback(() => {
    setStep("details");
    setSystemName("");
    setDescription("");
    setPreferredInstallMode("systemd");
    setSelectedInstallTab("systemd");
    setActiveSystem(null);
    setPendingSystems([]);
    setWorking(false);
    setIsLoadingPending(false);
    setIsPolling(false);
    setError(null);
    setCopyFeedback(null);
    setLastCheckedAt(null);
  }, []);

  const loadPendingSystems = useCallback(async () => {
    setIsLoadingPending(true);

    try {
      const response = await fetch("/api/admin/systems");
      const payload = await readPayload<SystemOnboardingRecord[]>(response);
      setPendingSystems(filterPendingSystems(payload));
    } catch (loadError) {
      setError(getApiErrorMessage(loadError, "Failed to load pending systems"));
    } finally {
      setIsLoadingPending(false);
    }
  }, []);

  const pollSystemStatus = useCallback(
    async (systemID: string, showLoader: boolean) => {
      if (showLoader) {
        setIsPolling(true);
      }

      try {
        const response = await fetch(`/api/admin/systems/${systemID}`);
        const payload = await readPayload<SystemOnboardingRecord>(response);

        setActiveSystem((current) => (current ? { ...current, ...payload } : payload));
        setPendingSystems((current) => current.filter((item) => item.id !== payload.id));
        setLastCheckedAt(new Date().toISOString());

        const redirectPath = getConnectedSystemRedirectPath(payload);
        if (redirectPath) {
          stopPolling();
          onClose();
          resetState();
          await router.push(redirectPath);
        }
      } catch (pollError) {
        setError(getApiErrorMessage(pollError, "Failed to refresh system status"));
      } finally {
        if (showLoader) {
          setIsPolling(false);
        }
      }
    },
    [onClose, resetState, router, stopPolling]
  );

  const dismiss = useCallback(() => {
    stopPolling();
    if (!preserveProgress) {
      resetState();
    }
    onClose();
  }, [onClose, preserveProgress, resetState, stopPolling]);

  useEffect(() => {
    if (!isOpen) {
      stopPolling();
      if (!preserveProgress) {
        resetState();
      }
      return;
    }

    void loadPendingSystems();
  }, [isOpen, loadPendingSystems, preserveProgress, resetState, stopPolling]);

  useEffect(() => {
    if (!isOpen || step !== "waiting" || !activeSystem || activeSystem.status === "connected") {
      stopPolling();
      return;
    }

    void pollSystemStatus(activeSystem.id, true);
    pollTimerRef.current = window.setInterval(() => {
      void pollSystemStatus(activeSystem.id, false);
    }, 3000);

    return () => stopPolling();
  }, [activeSystem, isOpen, pollSystemStatus, step, stopPolling]);

  useEffect(() => {
    if (!copyFeedback) {
      return;
    }

    const timeout = window.setTimeout(() => {
      setCopyFeedback(null);
    }, 1800);

    return () => window.clearTimeout(timeout);
  }, [copyFeedback]);

  async function handleCreateSystem() {
    setError(null);
    setCopyFeedback(null);
    setWorking(true);

    try {
      const response = await fetch("/api/admin/systems", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name: systemName,
          description,
        }),
      });

      const payload = await readPayload<SystemOnboardingRecord>(response);
      setActiveSystem(payload);
      setSelectedInstallTab(preferredInstallMode);
      setPendingSystems((current) => [
        {
          ...payload,
          api_key: undefined,
          enrollment_token: undefined,
          backend_url: undefined,
          install_script_url: undefined,
          docker_run_command: undefined,
          systemd_install_command: undefined,
          config_yaml: undefined,
        },
        ...current.filter((item) => item.id !== payload.id),
      ]);
      setStep("config");
    } catch (submitError) {
      setError(getApiErrorMessage(submitError, "Failed to create system"));
    } finally {
      setWorking(false);
    }
  }

  async function resumePendingSystem(systemID: string) {
    setError(null);
    setWorking(true);

    try {
      const response = await fetch(`/api/admin/systems/${systemID}`);
      const payload = await readPayload<SystemOnboardingRecord>(response);
      setActiveSystem(payload);
      setStep(getResumeFlowStep(payload));
    } catch (resumeError) {
      setError(getApiErrorMessage(resumeError, "Failed to load pending system"));
    } finally {
      setWorking(false);
    }
  }

  async function reissuePendingSystem(systemID: string) {
    setError(null);
    setCopyFeedback(null);
    stopPolling();
    setWorking(true);

    try {
      const response = await fetch(`/api/admin/systems/${systemID}/reissue`, {
        method: "POST",
      });
      const payload = await readPayload<SystemOnboardingRecord>(response);
      setActiveSystem(payload);
      setPendingSystems((current) => [
        {
          ...payload,
          api_key: undefined,
          enrollment_token: undefined,
          backend_url: undefined,
          install_script_url: undefined,
          docker_run_command: undefined,
          systemd_install_command: undefined,
          config_yaml: undefined,
        },
        ...current.filter((item) => item.id !== payload.id),
      ]);
      setLastCheckedAt(null);
      setStep("config");
    } catch (reissueError) {
      setError(getApiErrorMessage(reissueError, "Failed to reissue system credentials"));
    } finally {
      setWorking(false);
    }
  }

  async function cancelPendingSystem(systemID: string) {
    if (typeof window !== "undefined") {
      const confirmed = window.confirm("Cancel this pending system onboarding and invalidate its current credentials?");
      if (!confirmed) {
        return;
      }
    }

    setError(null);
    setCopyFeedback(null);
    stopPolling();
    setWorking(true);

    try {
      const response = await fetch(`/api/admin/systems/${systemID}/cancel`, {
        method: "POST",
      });
      await readPayload<{ status: string }>(response);
      setPendingSystems((current) => current.filter((item) => item.id !== systemID));
      setLastCheckedAt(null);

      if (activeSystem?.id === systemID) {
        setActiveSystem(null);
        setStep("details");
      }
    } catch (cancelError) {
      setError(getApiErrorMessage(cancelError, "Failed to cancel pending system"));
    } finally {
      setWorking(false);
    }
  }

  async function copyText(label: string, value: string) {
    if (typeof navigator === "undefined" || !navigator.clipboard) {
      setError("Clipboard access is not available in this browser.");
      return;
    }

    try {
      await navigator.clipboard.writeText(value);
      setCopyFeedback(`${label} copied.`);
      setError(null);
    } catch {
      setError(`Failed to copy ${label.toLowerCase()}.`);
    }
  }

  if (!isOpen) {
    return null;
  }

  const bootstrapToken = activeSystem?.enrollment_token || activeSystem?.api_key || "";
  const dockerCommand = activeSystem?.docker_run_command || (activeSystem?.config_yaml ? buildDockerCommand(activeSystem) : "");
  const systemdCommand =
    activeSystem?.systemd_install_command || (activeSystem?.config_yaml ? buildSystemdCommand(activeSystem) : "");
  const canSubmit = systemName.trim() !== "" && !working;
  const canRenderConfig = Boolean((dockerCommand || systemdCommand) && bootstrapToken && activeSystem?.backend_url);

  return (
    <div className="fixed inset-0 z-[60]">
      <button
        aria-label="Close add system panel"
        className="absolute inset-0 bg-black/50"
        onClick={dismiss}
        type="button"
      />

      <aside
        aria-labelledby="add-system-title"
        aria-modal="true"
        className="absolute right-0 top-0 flex h-full w-full max-w-[560px] flex-col border-l border-border bg-card shadow-2xl"
        role="dialog"
      >
        <div className="border-b border-border px-5 py-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <div className="text-[11px] font-medium uppercase tracking-[0.24em] text-muted-foreground">
                Add System
              </div>
              <h2 id="add-system-title" className="mt-2 text-xl font-semibold text-foreground">
                {activeSystem ? activeSystem.name : "Start A New Agent Onboarding"}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Create the system record, copy one install command, then wait for the first snapshot.
              </p>
            </div>

            <button
              className="rounded-md p-2 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              onClick={dismiss}
              type="button"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="mt-5 grid grid-cols-3 gap-2">
            {shellSteps.map(({ key, title, icon: Icon }, index) => {
              const active = step === key;
              const complete =
                (key === "details" && step !== "details") ||
                (key === "config" && (step === "install" || step === "waiting")) ||
                (key === "install" && step === "waiting");

              return (
                <div
                  key={key}
                  className={`rounded-lg border px-3 py-3 text-left transition-colors ${
                    active ? "border-foreground bg-background" : "border-border bg-background/50"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <div
                      className={`flex h-8 w-8 items-center justify-center rounded-md border ${
                        complete
                          ? "border-[hsl(140_50%_48%)] bg-[hsl(140_50%_48%)]/10 text-[hsl(140_50%_48%)]"
                          : "border-border"
                      }`}
                    >
                      {complete ? <Check className="h-4 w-4" /> : <Icon className="h-4 w-4" />}
                    </div>
                    <div className="min-w-0">
                      <div className="text-[11px] uppercase tracking-[0.2em] text-muted-foreground">
                        Step {index + 1}
                      </div>
                      <div className="truncate text-sm font-medium text-foreground">{title}</div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-5">
          {step === "details" ? (
            <div className="space-y-5">
              {pendingSystems.length > 0 ? (
                <section className="rounded-xl border border-border bg-background/40 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium text-foreground">Pending Connections</div>
                      <p className="mt-1 text-sm text-muted-foreground">
                        Reopen an onboarding wait state instead of starting over.
                      </p>
                    </div>
                    <button
                      className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-accent"
                      disabled={isLoadingPending}
                      onClick={() => void loadPendingSystems()}
                      type="button"
                    >
                      <RefreshCw className={`h-3.5 w-3.5 ${isLoadingPending ? "animate-spin" : ""}`} />
                      Refresh
                    </button>
                  </div>

                  <div className="mt-4 space-y-3">
                    {pendingSystems.map((system) => (
                      <div key={system.id} className="rounded-lg border border-border bg-card/60 p-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <div className="text-sm font-medium text-foreground">{system.name}</div>
                            <div className="mt-1 text-xs text-muted-foreground">
                              Server ID {system.server_id}
                            </div>
                          </div>
                          <div className="flex flex-wrap items-center justify-end gap-2">
                            <button
                              className="inline-flex h-9 items-center justify-center rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                              disabled={working}
                              onClick={() => void resumePendingSystem(system.id)}
                              type="button"
                            >
                              Resume
                            </button>
                            <button
                              className="inline-flex h-9 items-center justify-center rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                              disabled={working}
                              onClick={() => void reissuePendingSystem(system.id)}
                              type="button"
                            >
                              Reissue
                            </button>
                            <button
                              className="inline-flex h-9 items-center justify-center rounded-md border border-destructive/30 px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-60"
                              disabled={working}
                              onClick={() => void cancelPendingSystem(system.id)}
                              type="button"
                            >
                              Cancel
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </section>
              ) : null}

              <form
                className="space-y-5"
                onSubmit={(event) => {
                  event.preventDefault();
                  void handleCreateSystem();
                }}
              >
                <div className="rounded-xl border border-border bg-background/60 p-4">
                  <div className="text-sm font-medium text-foreground">System Details</div>
                  <p className="mt-1 text-sm text-muted-foreground">
                    This creates a pending system record and a fresh agent credential set for one VPS.
                  </p>
                </div>

                <label className="grid gap-2">
                  <span className="text-sm font-medium text-foreground">System Name</span>
                  <input
                    className={inputClassName}
                    onChange={(event) => setSystemName(event.target.value)}
                    placeholder="Production VPS"
                    type="text"
                    value={systemName}
                  />
                </label>

                <label className="grid gap-2">
                  <span className="text-sm font-medium text-foreground">Description</span>
                  <textarea
                    className="min-h-[110px] w-full rounded-md border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring"
                    onChange={(event) => setDescription(event.target.value)}
                    placeholder="Primary app node in ap-south-1"
                    value={description}
                  />
                </label>

                <div className="grid gap-2">
                  <span className="text-sm font-medium text-foreground">Install Mode</span>
                  <div className="grid grid-cols-2 gap-2">
                    <InstallModeButton
                      active={preferredInstallMode === "docker"}
                      description="Run the agent as a container."
                      label="Docker"
                      onClick={() => setPreferredInstallMode("docker")}
                    />
                    <InstallModeButton
                      active={preferredInstallMode === "systemd"}
                      description="Run the agent as a host service."
                      label="systemd"
                      onClick={() => setPreferredInstallMode("systemd")}
                    />
                  </div>
                </div>

                {error ? (
                  <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {error}
                  </div>
                ) : null}

                <button
                  className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-md border border-border bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={!canSubmit}
                  type="submit"
                >
                  {working ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                  {working ? "Creating System..." : "Create Install Instructions"}
                </button>
              </form>
            </div>
          ) : null}

          {step === "config" && activeSystem && canRenderConfig ? (
            <div className="space-y-5">
              <div className="rounded-xl border border-border bg-background/60 p-4">
                <div className="text-sm font-medium text-foreground">Quick Install</div>
                <p className="mt-1 text-sm text-muted-foreground">
                  Copy one command and run it on the target host. The first agent startup will exchange the bootstrap token for a long-lived API key automatically.
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <InfoCard label="Server ID" value={activeSystem.server_id} />
                <InfoCard label="Agent ID" value={activeSystem.agent_id} />
                <InfoCard label="Backend URL" value={activeSystem.backend_url as string} />
                <InfoCard label="Status" value={activeSystem.status} />
              </div>

              <div className="grid grid-cols-2 gap-2">
                <InstallModeButton
                  active={selectedInstallTab === "docker"}
                  description="Containerized one-command install."
                  label="Docker"
                  onClick={() => setSelectedInstallTab("docker")}
                />
                <InstallModeButton
                  active={selectedInstallTab === "systemd"}
                  description="Host service script. Recommended."
                  label="systemd"
                  onClick={() => setSelectedInstallTab("systemd")}
                />
              </div>

              <ConfigSection
                description={
                  selectedInstallTab === "docker"
                    ? "Runs the agent directly from the container image using environment variables instead of a handwritten config file."
                    : "Fetches the installer script, extracts the agent binary from the Docker image, installs the systemd unit, and starts the service."
                }
                label={selectedInstallTab === "docker" ? "Docker Command" : "systemd Install Command"}
                onCopy={() => void copyText("Install commands", selectedInstallTab === "docker" ? dockerCommand : systemdCommand)}
                value={selectedInstallTab === "docker" ? dockerCommand : systemdCommand}
              />
            </div>
          ) : null}

          {step === "install" && activeSystem && canRenderConfig ? (
            <div className="space-y-5">
              <div className="rounded-xl border border-border bg-background/60 p-4">
                <div className="text-sm font-medium text-foreground">Advanced Manual Setup</div>
                <p className="mt-1 text-sm text-muted-foreground">
                  Use these raw values only if you need to customize the install yourself instead of using the quick commands.
                </p>
              </div>

              <ConfigSection
                description="Legacy/manual fallback. The quick install commands above already include these values."
                label="Agent Config"
                onCopy={() => void copyText("Config YAML", activeSystem.config_yaml as string)}
                value={activeSystem.config_yaml as string}
              />

              <ConfigSection
                description="Shown once here for bootstrap only. The agent rewrites its persisted config with the real API key after enrollment."
                label="Enrollment Token"
                onCopy={() => void copyText("Enrollment token", bootstrapToken)}
                value={bootstrapToken}
              />
            </div>
          ) : null}

          {step === "waiting" && activeSystem ? (
            <div className="space-y-5">
              <div className="rounded-xl border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-[hsl(140_50%_48%)]">
                  {isPolling ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                  Waiting For First Snapshot
                </div>
                <p className="mt-1 text-sm text-foreground">
                  Keep this open after you start the agent. The drawer will redirect automatically as soon as the system connects.
                </p>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <InfoCard label="System Name" value={activeSystem.name} />
                <InfoCard label="System ID" value={activeSystem.id} />
                <InfoCard label="Server ID" value={activeSystem.server_id} />
                <InfoCard label="Current Status" value={activeSystem.status} />
              </div>

              <div className="rounded-xl border border-border bg-background/40 p-4">
                <div className="text-sm font-medium text-foreground">Polling Status</div>
                <p className="mt-1 text-sm text-muted-foreground">
                  Status is refreshed every few seconds through the admin onboarding detail API.
                </p>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <button
                    className="inline-flex h-9 items-center gap-1 rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={isPolling}
                    onClick={() => void pollSystemStatus(activeSystem.id, true)}
                    type="button"
                  >
                    <RefreshCw className={`h-3.5 w-3.5 ${isPolling ? "animate-spin" : ""}`} />
                    Refresh Now
                  </button>
                  {lastCheckedAt ? (
                    <span className="text-xs text-muted-foreground">
                      Last checked {formatDateTime(lastCheckedAt)}
                    </span>
                  ) : null}
                </div>
                <div className="mt-4 flex flex-wrap items-center gap-2">
                  <button
                    className="inline-flex h-9 items-center justify-center rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={working}
                    onClick={() => void reissuePendingSystem(activeSystem.id)}
                    type="button"
                  >
                    Reissue Credentials
                  </button>
                  <button
                    className="inline-flex h-9 items-center justify-center rounded-md border border-destructive/30 px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={working}
                    onClick={() => void cancelPendingSystem(activeSystem.id)}
                    type="button"
                  >
                    Cancel Pending System
                  </button>
                </div>
              </div>
            </div>
          ) : null}
        </div>

        <div className="border-t border-border px-5 py-4">
          {copyFeedback ? (
            <div className="mb-3 rounded-md border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 px-3 py-2 text-sm text-[hsl(140_50%_48%)]">
              {copyFeedback}
            </div>
          ) : null}
          {error && step !== "details" ? (
            <div className="mb-3 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          ) : null}

          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {step !== "details" && step !== "waiting" ? (
                <button
                  className="inline-flex h-11 items-center gap-1 rounded-md border border-border px-4 text-sm font-medium text-foreground transition-colors hover:bg-accent"
                  onClick={() => setStep(step === "config" ? "details" : "config")}
                  type="button"
                >
                  <ArrowLeft className="h-4 w-4" />
                  Back
                </button>
              ) : null}
            </div>

            <button
              className="inline-flex h-11 items-center justify-center rounded-md border border-border px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              onClick={dismiss}
              type="button"
            >
              {step === "waiting" ? "Close And Resume Later" : "Close"}
            </button>
          </div>

          {step === "config" ? (
            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              <button
                className="inline-flex h-11 items-center justify-center rounded-md border border-border bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90"
                onClick={() => setStep("waiting")}
                type="button"
              >
                Continue To Waiting State
              </button>
              <button
                className="inline-flex h-11 items-center justify-center rounded-md border border-border px-4 text-sm font-medium text-foreground transition-colors hover:bg-accent"
                onClick={() => setStep("install")}
                type="button"
              >
                View Advanced Manual Setup
              </button>
            </div>
          ) : null}

          {step === "install" ? (
            <button
              className="mt-3 inline-flex h-11 w-full items-center justify-center rounded-md border border-border bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90"
              onClick={() => setStep("waiting")}
              type="button"
            >
              Continue To Waiting State
            </button>
          ) : null}
        </div>
      </aside>
    </div>
  );
}

function InstallModeButton({
  active,
  description,
  label,
  onClick,
}: {
  active: boolean;
  description: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className={`rounded-lg border px-4 py-3 text-left transition-colors ${
        active ? "border-foreground bg-background" : "border-border bg-background/40 hover:bg-background/70"
      }`}
      onClick={onClick}
      type="button"
    >
      <div className="text-sm font-medium text-foreground">{label}</div>
      <div className="mt-1 text-xs text-muted-foreground">{description}</div>
    </button>
  );
}

function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border bg-background/40 p-4">
      <div className="text-[11px] uppercase tracking-[0.2em] text-muted-foreground">{label}</div>
      <div className="mt-2 break-all font-mono text-xs text-foreground">{value}</div>
    </div>
  );
}

function ConfigSection({
  description,
  label,
  onCopy,
  value,
}: {
  description: string;
  label: string;
  onCopy: () => void;
  value: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-background/40">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <div className="text-sm font-medium text-foreground">{label}</div>
          <div className="text-xs text-muted-foreground">{description}</div>
        </div>
        <button
          className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-accent"
          onClick={onCopy}
          type="button"
        >
          <Copy className="h-3.5 w-3.5" />
          Copy
        </button>
      </div>
      <CodeBlock value={value} />
    </div>
  );
}

function CodeBlock({ value }: { value: string }) {
  return (
    <pre className="max-h-[280px] overflow-x-auto overflow-y-auto p-4 text-xs leading-6 text-foreground">
      <code>{value}</code>
    </pre>
  );
}

function buildDockerCommand(system: SystemOnboardingRecord): string {
  return [
    "docker run -d \\",
    "  --name bifrost-agent \\",
    "  --restart unless-stopped \\",
    "  --network host \\",
    "  -v bifrost-agent-data:/var/lib/bifrost-agent \\",
    "  -v /var/run/docker.sock:/var/run/docker.sock:ro \\",
    "  -e BIFROST_CONFIG_PATH=/var/lib/bifrost-agent/config.yaml \\",
    `  -e BIFROST_AGENT_ID=${shellEnvQuote(system.agent_id)} \\`,
    `  -e BIFROST_SERVER_ID=${shellEnvQuote(system.server_id)} \\`,
    `  -e BIFROST_SERVER_NAME=${shellEnvQuote(system.name)} \\`,
    `  -e BIFROST_BACKEND_URL=${shellEnvQuote(system.backend_url || "")} \\`,
    `  -e BIFROST_ENROLLMENT_TOKEN=${shellEnvQuote(system.enrollment_token || system.api_key || "")} \\`,
    "  bifrost-agent:latest",
  ].join("\n");
}

function buildSystemdCommand(system: SystemOnboardingRecord): string {
  return [
    `curl -fsSL ${shellEnvQuote((system.install_script_url || `${system.backend_url || ""}/api/v1/install/agent.sh`).replace(/\/+$/, ""))} | sudo env \\`,
    `  BIFROST_AGENT_ID=${shellEnvQuote(system.agent_id)} \\`,
    `  BIFROST_SERVER_ID=${shellEnvQuote(system.server_id)} \\`,
    `  BIFROST_SERVER_NAME=${shellEnvQuote(system.name)} \\`,
    `  BIFROST_BACKEND_URL=${shellEnvQuote(system.backend_url || "")} \\`,
    `  BIFROST_ENROLLMENT_TOKEN=${shellEnvQuote(system.enrollment_token || system.api_key || "")} \\`,
    "  sh",
  ].join("\n");
}

function shellEnvQuote(value: string): string {
  return `'${value.replace(/'/g, `'\"'\"'`)}'`;
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleTimeString();
}
