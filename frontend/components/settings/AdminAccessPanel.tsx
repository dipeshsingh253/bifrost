import { useState, type FormEvent, type ReactNode } from "react";
import { Copy, MailPlus, Shield, Trash2, UserRoundX, Users } from "lucide-react";

import {
  getApiErrorMessage,
  readBrowserApiPayload,
  type TenantSummary,
  type ViewerAccount,
  type ViewerInvite,
} from "@/lib/api";

type AdminAccessPanelProps = {
  invites: ViewerInvite[];
  tenant: TenantSummary;
  viewers: ViewerAccount[];
};

const inputClassName =
  "h-11 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring";

export function AdminAccessPanel({
  invites: initialInvites,
  tenant: initialTenant,
  viewers: initialViewers,
}: AdminAccessPanelProps) {
  const [tenant, setTenant] = useState(initialTenant);
  const [invites, setInvites] = useState(initialInvites);
  const [viewers, setViewers] = useState(initialViewers);
  const [inviteEmail, setInviteEmail] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [formSuccess, setFormSuccess] = useState<string | null>(null);
  const [freshInviteLink, setFreshInviteLink] = useState<string | null>(null);
  const [workingKey, setWorkingKey] = useState<string | null>(null);

  async function copyInviteLink() {
    if (!freshInviteLink || typeof navigator === "undefined" || !navigator.clipboard) {
      return;
    }

    await navigator.clipboard.writeText(freshInviteLink);
    setFormSuccess("Invite created and copied to clipboard.");
  }

  async function handleInviteSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    setFormSuccess(null);
    setFreshInviteLink(null);
    setWorkingKey("invite");

    try {
      const response = await fetch("/api/admin/invites", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email: inviteEmail,
        }),
      });

      const invite = await readBrowserApiPayload<ViewerInvite>(response, "Failed to create invite");
      setInvites((current) => [invite, ...current.filter((item) => item.id !== invite.id)]);
      setTenant((current) => ({
        ...current,
      }));
      setInviteEmail("");

      if (invite.invite_token && typeof window !== "undefined") {
        const link = `${window.location.origin}/invite/${invite.invite_token}`;
        setFreshInviteLink(link);
        setFormSuccess("Viewer invite created. Share the link below.");
      } else {
        setFormSuccess("Viewer invite created.");
      }
    } catch (error) {
      setFormError(getApiErrorMessage(error, "Failed to create invite"));
    } finally {
      setWorkingKey(null);
    }
  }

  async function revokeInvite(inviteID: string) {
    setFormError(null);
    setFormSuccess(null);
    setWorkingKey(`revoke:${inviteID}`);

    try {
      const response = await fetch(`/api/admin/invites/${inviteID}/revoke`, {
        method: "POST",
      });
      await readBrowserApiPayload<{ status: string }>(response, "Failed to revoke invite");
      setInvites((current) => current.filter((invite) => invite.id !== inviteID));
      setFormSuccess("Invite revoked.");
    } catch (error) {
      setFormError(getApiErrorMessage(error, "Failed to revoke invite"));
    } finally {
      setWorkingKey(null);
    }
  }

  async function disableViewer(userID: string) {
    setFormError(null);
    setFormSuccess(null);
    setWorkingKey(`disable:${userID}`);

    try {
      const response = await fetch(`/api/admin/viewers/${userID}/disable`, {
        method: "POST",
      });
      await readBrowserApiPayload<{ status: string }>(response, "Failed to disable viewer");
      setViewers((current) =>
        current.map((viewer) =>
          viewer.id === userID
            ? {
                ...viewer,
                status: "disabled",
                disabled_at: new Date().toISOString(),
              }
            : viewer
        )
      );
      setFormSuccess("Viewer disabled.");
    } catch (error) {
      setFormError(getApiErrorMessage(error, "Failed to disable viewer"));
    } finally {
      setWorkingKey(null);
    }
  }

  async function deleteViewer(userID: string) {
    if (typeof window !== "undefined" && !window.confirm("Delete this viewer account from the tenant?")) {
      return;
    }

    setFormError(null);
    setFormSuccess(null);
    setWorkingKey(`delete:${userID}`);

    try {
      const response = await fetch(`/api/admin/viewers/${userID}`, {
        method: "DELETE",
      });
      await readBrowserApiPayload<{ status: string }>(response, "Failed to delete viewer");
      setViewers((current) => current.filter((viewer) => viewer.id !== userID));
      setTenant((current) => ({
        ...current,
        viewer_count: Math.max(0, current.viewer_count - 1),
      }));
      setFormSuccess("Viewer deleted.");
    } catch (error) {
      setFormError(getApiErrorMessage(error, "Failed to delete viewer"));
    } finally {
      setWorkingKey(null);
    }
  }

  return (
    <div className="p-0">
      <div className="border-b border-border p-6">
        <h2 className="text-lg font-medium text-foreground">Admin</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Manage viewer access for <span className="text-foreground">{tenant.tenant_name}</span>.
        </p>
      </div>

      <div className="grid gap-4 p-6 md:grid-cols-3">
        <StatCard
          description="Admins with full tenant access"
          icon={<Shield className="h-4 w-4" />}
          title="Admins"
          value={tenant.admin_count.toString()}
        />
        <StatCard
          description="Current viewer accounts in this tenant"
          icon={<Users className="h-4 w-4" />}
          title="Viewers"
          value={tenant.viewer_count.toString()}
        />
        <StatCard
          description="Pending viewer invites awaiting acceptance"
          icon={<MailPlus className="h-4 w-4" />}
          title="Pending Invites"
          value={invites.length.toString()}
        />
      </div>

      <div className="grid gap-6 px-6 pb-6 xl:grid-cols-[1fr_1.1fr]">
        <section className="rounded-xl border border-border bg-background/60 p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-foreground">Invite Viewer</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Viewer access is read-only. Invite links expire after seven days.
            </p>
          </div>

          <form className="grid gap-3" onSubmit={handleInviteSubmit}>
            <label className="grid gap-2">
              <span className="text-sm font-medium text-foreground">Viewer Email</span>
              <input
                autoComplete="email"
                className={inputClassName}
                onChange={(event) => setInviteEmail(event.target.value)}
                placeholder="viewer@example.com"
                type="email"
                value={inviteEmail}
              />
            </label>

            <button
              className="inline-flex h-11 items-center justify-center rounded-md border border-border bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={workingKey === "invite"}
              type="submit"
            >
              {workingKey === "invite" ? "Creating..." : "Create Invite"}
            </button>
          </form>

          {formError ? (
            <div className="mt-4 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {formError}
            </div>
          ) : null}
          {formSuccess ? (
            <div className="mt-4 rounded-md border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 px-3 py-2 text-sm text-[hsl(140_50%_48%)]">
              {formSuccess}
            </div>
          ) : null}

          {freshInviteLink ? (
            <div className="mt-4 rounded-lg border border-border bg-background/70 p-4">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                    Latest Invite Link
                  </div>
                  <div className="mt-2 truncate font-mono text-xs text-foreground">{freshInviteLink}</div>
                </div>
                <button
                  className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-accent"
                  onClick={copyInviteLink}
                  type="button"
                >
                  <Copy className="h-3.5 w-3.5" />
                  Copy
                </button>
              </div>
            </div>
          ) : null}
        </section>

        <section className="rounded-xl border border-border bg-background/60 p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-foreground">Pending Invites</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Revoke invite links that should no longer grant viewer access.
            </p>
          </div>

          <div className="grid gap-3">
            {invites.length === 0 ? (
              <EmptyState message="No pending viewer invites." />
            ) : (
              invites.map((invite) => (
                <div key={invite.id} className="flex flex-col gap-3 rounded-lg border border-border bg-card p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <div className="truncate font-medium text-foreground">{invite.email}</div>
                    <div className="mt-1 text-sm text-muted-foreground">
                      Expires {formatDateTime(invite.expires_at)}
                    </div>
                  </div>
                  <button
                    className="inline-flex h-9 items-center justify-center rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={workingKey === `revoke:${invite.id}`}
                    onClick={() => revokeInvite(invite.id)}
                    type="button"
                  >
                    {workingKey === `revoke:${invite.id}` ? "Revoking..." : "Revoke"}
                  </button>
                </div>
              ))
            )}
          </div>
        </section>
      </div>

      <section className="px-6 pb-6">
        <div className="rounded-xl border border-border bg-background/60 p-5">
          <div className="mb-4">
            <h3 className="text-lg font-semibold text-foreground">Viewer Accounts</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Disable access immediately or delete viewer memberships that are no longer needed.
            </p>
          </div>

          <div className="grid gap-3">
            {viewers.length === 0 ? (
              <EmptyState message="No viewer accounts have joined this tenant yet." />
            ) : (
              viewers.map((viewer) => (
                <div key={viewer.id} className="flex flex-col gap-4 rounded-lg border border-border bg-card p-4 lg:flex-row lg:items-center lg:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate font-medium text-foreground">{viewer.name || viewer.email}</span>
                      <span
                        className={`rounded-full px-2 py-0.5 text-[11px] font-medium uppercase tracking-[0.16em] ${
                          viewer.status === "disabled"
                            ? "bg-destructive/10 text-destructive"
                            : "bg-[hsl(140_50%_48%)]/10 text-[hsl(140_50%_48%)]"
                        }`}
                      >
                        {viewer.status}
                      </span>
                    </div>
                    <div className="mt-1 truncate text-sm text-muted-foreground">{viewer.email}</div>
                    {viewer.disabled_at ? (
                      <div className="mt-1 text-xs text-muted-foreground">
                        Disabled {formatDateTime(viewer.disabled_at)}
                      </div>
                    ) : null}
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    <button
                      className="inline-flex h-9 items-center gap-1 rounded-md border border-border px-3 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                      disabled={viewer.status === "disabled" || workingKey === `disable:${viewer.id}`}
                      onClick={() => disableViewer(viewer.id)}
                      type="button"
                    >
                      <UserRoundX className="h-3.5 w-3.5" />
                      {workingKey === `disable:${viewer.id}` ? "Disabling..." : "Disable"}
                    </button>
                    <button
                      className="inline-flex h-9 items-center gap-1 rounded-md border border-destructive/30 px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-60"
                      disabled={workingKey === `delete:${viewer.id}`}
                      onClick={() => deleteViewer(viewer.id)}
                      type="button"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      {workingKey === `delete:${viewer.id}` ? "Deleting..." : "Delete"}
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </section>
    </div>
  );
}

function StatCard({
  description,
  icon,
  title,
  value,
}: {
  description: string;
  icon: ReactNode;
  title: string;
  value: string;
}) {
  return (
    <div className="rounded-xl border border-border bg-background/60 p-5">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        {icon}
        <span>{title}</span>
      </div>
      <div className="mt-3 text-3xl font-semibold text-foreground">{value}</div>
      <div className="mt-2 text-sm text-muted-foreground">{description}</div>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-dashed border-border bg-card px-4 py-10 text-center text-sm text-muted-foreground">
      {message}
    </div>
  );
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}
