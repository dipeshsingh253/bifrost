import Link from "next/link";
import { useRouter } from "next/router";
import { useState, type PropsWithChildren } from "react";
import { Search, Sun, Settings, User, Plus, Shield } from "lucide-react";

import { AddSystemShell } from "@/components/AddSystemShell";
import { hasAdminAccess, type AuthUser } from "@/lib/api";

type LayoutProps = PropsWithChildren<{
  currentUser?: AuthUser | null;
}>;

export function Layout({ children, currentUser }: LayoutProps) {
  const router = useRouter();
  const isHome = router.pathname === "/";
  const [loggingOut, setLoggingOut] = useState(false);
  const [isAddSystemOpen, setIsAddSystemOpen] = useState(false);
  const canManageSystems = hasAdminAccess(currentUser);

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await fetch("/api/auth/logout", {
        method: "POST",
      });
    } finally {
      await router.push("/login");
      setLoggingOut(false);
    }
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <AddSystemShell isOpen={canManageSystems && isAddSystemOpen} onClose={() => setIsAddSystemOpen(false)} />

      {/* ── Top navigation bar ── */}
      <header className="sticky top-0 z-50 border-b border-border bg-card">
        <div className="mx-auto flex max-w-[1500px] items-center justify-between px-4 py-2.5">
          {/* Left: logo + search */}
          <div className="flex items-center gap-3">
            <Link
              href="/"
              className="text-xl font-bold tracking-tight text-foreground hover:opacity-90 transition-opacity"
            >
              Bifrost
            </Link>

            <div className="hidden md:flex items-center gap-2 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-muted-foreground min-w-[180px]">
              <Search className="h-3.5 w-3.5 opacity-60" />
              <span>Search</span>
              <span className="ml-auto rounded border border-border px-1.5 py-0.5 text-[10px] font-medium tracking-wider">
                Ctrl K
              </span>
            </div>
          </div>

          {/* Right: actions */}
          <div className="flex items-center gap-1">
            <button
              className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="Toggle theme"
            >
              <Sun className="h-4 w-4" />
            </button>
            <button
              className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="Settings"
            >
              <Settings className="h-4 w-4" />
            </button>
            <button
              className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              title="User"
            >
              <User className="h-4 w-4" />
            </button>
            {canManageSystems ? (
              <Link
                className="ml-2 inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-accent"
                href="/admin"
              >
                <Shield className="h-3.5 w-3.5" />
                Admin
              </Link>
            ) : null}
            {currentUser ? (
              <div className="ml-2 hidden items-center gap-2 rounded-md border border-border bg-background px-3 py-1.5 sm:flex">
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-foreground">{currentUser.name}</div>
                  <div className="truncate text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
                    {currentUser.role}
                  </div>
                </div>
                <button
                  className="rounded-md border border-border px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={loggingOut}
                  onClick={handleLogout}
                  type="button"
                >
                  {loggingOut ? "..." : "Logout"}
                </button>
              </div>
            ) : null}
            {canManageSystems ? (
              <button
                className="ml-2 flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-accent"
                onClick={() => setIsAddSystemOpen(true)}
                type="button"
              >
                <Plus className="h-3.5 w-3.5" />
                Add System
              </button>
            ) : null}
          </div>
        </div>
      </header>

      {/* ── Main content ── */}
      <main className={`mx-auto max-w-[1500px] px-4 ${isHome ? "py-6" : "py-4"}`}>
        {children}
      </main>
    </div>
  );
}
