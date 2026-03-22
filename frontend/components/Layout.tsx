import Link from "next/link";
import { useRouter } from "next/router";
import type { PropsWithChildren } from "react";
import { Search, Sun, Settings, User, Plus } from "lucide-react";

export function Layout({ children }: PropsWithChildren) {
  const router = useRouter();
  const isHome = router.pathname === "/";

  return (
    <div className="min-h-screen bg-background text-foreground">
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
            <button className="ml-2 flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-sm font-medium text-foreground hover:bg-accent transition-colors">
              <Plus className="h-3.5 w-3.5" />
              Add System
            </button>
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
