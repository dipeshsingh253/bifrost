import Link from "next/link";
import { useRouter } from "next/router";
import type { PropsWithChildren } from "react";
import { Layout } from "@/components/Layout";
import { hasAdminAccess, type AuthUser } from "@/lib/api";

type SettingsLayoutProps = PropsWithChildren<{
  currentUser: AuthUser;
}>;

export function SettingsLayout({ children, currentUser }: SettingsLayoutProps) {
  const router = useRouter();
  const isAdmin = hasAdminAccess(currentUser);

  const navItems = [
    { name: "Profile", href: "/settings/profile" },
    { name: "Appearance", href: "/settings/appearance" },
    ...(isAdmin ? [{ name: "Admin", href: "/settings/admin" }] : []),
  ];

  return (
    <Layout currentUser={currentUser}>
      <div className="mx-auto max-w-5xl pt-4 pb-12">
        <h1 className="text-2xl font-semibold tracking-tight text-foreground mb-8">Settings</h1>
        
        <div className="flex flex-col md:flex-row gap-8">
          {/* Sidebar */}
          <aside className="w-full md:w-56 shrink-0">
            <nav className="flex flex-col space-y-1">
              {navItems.map((item) => {
                const isActive = router.pathname === item.href;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? "bg-accent text-foreground"
                        : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                    }`}
                  >
                    {item.name}
                  </Link>
                );
              })}
            </nav>
          </aside>

          {/* Content Area */}
          <main className="flex-1 min-w-0">
            <div className="rounded-lg border border-border bg-card">
              {children}
            </div>
          </main>
        </div>
      </div>
    </Layout>
  );
}
