import Link from "next/link";
import type { PropsWithChildren, ReactNode } from "react";

type AuthShellProps = PropsWithChildren<{
  eyebrow?: string;
  footer?: ReactNode;
  subtitle: string;
  title: string;
}>;

export function AuthShell({ children, eyebrow, footer, subtitle, title }: AuthShellProps) {
  return (
    <div className="min-h-screen bg-background px-4 py-10 text-foreground">
      <div className="mx-auto flex min-h-[calc(100vh-5rem)] max-w-5xl items-center justify-center">
        <div className="grid w-full gap-6 lg:grid-cols-[1.1fr_0.9fr]">
          <section className="rounded-2xl border border-border bg-card/95 p-8 shadow-[0_20px_80px_rgba(0,0,0,0.35)]">
            <Link
              href="/"
              className="inline-flex items-center text-xl font-bold tracking-tight text-foreground transition-opacity hover:opacity-90"
            >
              Bifrost
            </Link>
            {eyebrow ? (
              <div className="mt-10 text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                {eyebrow}
              </div>
            ) : null}
            <h1 className="mt-4 text-3xl font-semibold tracking-tight text-foreground">{title}</h1>
            <p className="mt-3 max-w-xl text-sm leading-6 text-muted-foreground">{subtitle}</p>
            <div className="mt-8">{children}</div>
            {footer ? <div className="mt-6 text-sm text-muted-foreground">{footer}</div> : null}
          </section>

          <aside className="hidden rounded-2xl border border-border bg-[radial-gradient(circle_at_top,_rgba(49,196,141,0.18),_transparent_45%),linear-gradient(180deg,rgba(22,22,24,0.95),rgba(11,11,13,0.98))] p-8 lg:flex lg:flex-col lg:justify-between">
            <div>
              <div className="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">
                Monitoring
              </div>
              <h2 className="mt-4 text-2xl font-semibold tracking-tight text-foreground">
                Keep server access simple and tenant-scoped.
              </h2>
              <p className="mt-4 text-sm leading-6 text-muted-foreground">
                Bifrost keeps admin access narrow, viewer access read-only, and onboarding simple enough
                for a single self-hosted install today without blocking multi-tenant cloud later.
              </p>
            </div>

            <div className="grid gap-4">
              <div className="rounded-xl border border-border/80 bg-background/60 p-4">
                <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  Admin
                </div>
                <p className="mt-2 text-sm text-foreground">
                  Bootstrap the install, invite viewers, and manage tenant access from the dashboard.
                </p>
              </div>
              <div className="rounded-xl border border-border/80 bg-background/60 p-4">
                <div className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  Viewer
                </div>
                <p className="mt-2 text-sm text-foreground">
                  Read-only access to systems, projects, containers, logs, and metrics within the tenant.
                </p>
              </div>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}
