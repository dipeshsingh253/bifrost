import Link from "next/link";
import { useRouter } from "next/router";
import { useState, type PropsWithChildren } from "react";
import { Search, Plus } from "lucide-react";

import { AddSystemShell } from "@/components/AddSystemShell";
import { hasAdminAccess, type AuthUser } from "@/lib/api";
import { NavbarAvatar } from "@/components/NavbarAvatar";

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
          <div className="flex items-center gap-3">
            {canManageSystems ? (
              <button
                className="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground transition-colors hover:bg-accent shadow-sm"
                onClick={() => setIsAddSystemOpen(true)}
                type="button"
              >
                <Plus className="h-3.5 w-3.5" />
                Add System
              </button>
            ) : null}
            {currentUser ? (
              <NavbarAvatar
                currentUser={currentUser}
                onLogout={handleLogout}
                loggingOut={loggingOut}
              />
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
