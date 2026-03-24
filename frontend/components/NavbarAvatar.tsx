import Link from "next/link";
import { useState, useRef, useEffect } from "react";
import { User, Settings as SettingsIcon, LogOut } from "lucide-react";
import type { AuthUser } from "@/lib/api";

type NavbarAvatarProps = {
  currentUser: AuthUser;
  onLogout: () => void;
  loggingOut: boolean;
};

export function NavbarAvatar({ currentUser, onLogout, loggingOut }: NavbarAvatarProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const initials = currentUser.name
    ? currentUser.name.substring(0, 2).toUpperCase()
    : currentUser.email.substring(0, 2).toUpperCase();

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-medium text-foreground transition-colors hover:bg-accent/80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:ring-offset-background"
        title="Account menu"
      >
        {initials}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-48 origin-top-right rounded-md border border-border bg-popover shadow-md ring-1 ring-black/5 focus:outline-none z-50 overflow-hidden">
          <div className="px-4 py-3 border-b border-border">
            <p className="text-sm font-medium text-foreground truncate">{currentUser.name}</p>
            <p className="text-xs text-muted-foreground truncate">{currentUser.email}</p>
          </div>
          <div className="py-1">
            <Link
              href="/settings/profile"
              onClick={() => setIsOpen(false)}
              className="flex items-center px-4 py-2 text-sm text-foreground hover:bg-accent transition-colors"
            >
              <User className="mr-2 h-4 w-4 text-muted-foreground" />
              Profile
            </Link>
            <Link
              href="/settings"
              onClick={() => setIsOpen(false)}
              className="flex items-center px-4 py-2 text-sm text-foreground hover:bg-accent transition-colors"
            >
              <SettingsIcon className="mr-2 h-4 w-4 text-muted-foreground" />
              Settings
            </Link>
          </div>
          <div className="py-1 border-t border-border">
            <button
              type="button"
              onClick={() => {
                setIsOpen(false);
                onLogout();
              }}
              disabled={loggingOut}
              className="flex w-full items-center px-4 py-2 text-sm text-foreground hover:bg-accent transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <LogOut className="mr-2 h-4 w-4 text-muted-foreground" />
              {loggingOut ? "Logging out..." : "Log out"}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
