import Head from "next/head";
import type { GetServerSideProps } from "next";
import { useState } from "react";

import { SettingsLayout } from "@/components/SettingsLayout";
import {
  readAppearanceSettings,
  saveAppearanceSettings,
  type AppearanceSettings,
} from "@/lib/appearance";
import { requireAuthenticatedPage, type AuthenticatedPageProps } from "@/lib/api";

export default function AppearanceSettings({ currentUser }: AuthenticatedPageProps) {
  const [settings, setSettings] = useState<AppearanceSettings>(() => readAppearanceSettings());
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  function updateSetting<K extends keyof AppearanceSettings>(key: K, value: AppearanceSettings[K]) {
    setSuccessMessage(null);
    setSettings((current) => ({
      ...current,
      [key]: value,
    }));
  }

  function handleSave() {
    saveAppearanceSettings(settings);
    setSuccessMessage("Appearance preferences saved on this browser.");
  }

  return (
    <>
      <Head>
        <title>Appearance Settings - Bifrost</title>
      </Head>
      <SettingsLayout currentUser={currentUser}>
        <div className="p-6">
          <h2 className="mb-6 text-lg font-medium text-foreground">Appearance</h2>

          <div className="max-w-xl space-y-8">
            <section>
              <div className="space-y-6">
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <div>
                    <label className="text-sm font-medium text-foreground" htmlFor="theme-select">
                      Theme
                    </label>
                  </div>
                  <div className="relative sm:col-span-2">
                    <select
                      id="theme-select"
                      className="w-full appearance-none rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) => updateSetting("theme", event.target.value as AppearanceSettings["theme"])}
                      suppressHydrationWarning
                      value={settings.theme}
                    >
                      <option value="system">System</option>
                      <option value="dark">Dark</option>
                      <option value="light">Light</option>
                    </select>
                    <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-muted-foreground">
                      <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <div>
                    <label className="text-sm font-medium text-foreground" htmlFor="time-range-select">
                      Default Time Range
                    </label>
                  </div>
                  <div className="relative sm:col-span-2">
                    <select
                      id="time-range-select"
                      className="w-full appearance-none rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) =>
                        updateSetting("defaultTimeRange", event.target.value as AppearanceSettings["defaultTimeRange"])
                      }
                      suppressHydrationWarning
                      value={settings.defaultTimeRange}
                    >
                      <option value="1h">1 hour</option>
                      <option value="6h">6 hours</option>
                      <option value="24h">24 hours</option>
                      <option value="7d">7 days</option>
                    </select>
                    <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-muted-foreground">
                      <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </div>
                </div>
              </div>

              <p className="mt-6 text-sm text-muted-foreground">
                Theme and default time range are stored locally in this browser.
              </p>

              {successMessage ? (
                <div className="mt-4 rounded-md border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 px-3 py-2 text-sm text-[hsl(140_50%_48%)]">
                  {successMessage}
                </div>
              ) : null}

              <div className="mt-6 flex justify-end">
                <button
                  type="button"
                  className="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                  onClick={handleSave}
                >
                  Save Preferences
                </button>
              </div>
            </section>
          </div>
        </div>
      </SettingsLayout>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<AuthenticatedPageProps> = async (context) => {
  return requireAuthenticatedPage(context, async () => {
    return {};
  });
};
