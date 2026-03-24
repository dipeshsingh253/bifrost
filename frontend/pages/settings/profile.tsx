import Head from "next/head";
import type { GetServerSideProps } from "next";
import { useState, type FormEvent } from "react";

import { SettingsLayout } from "@/components/SettingsLayout";
import {
  getApiErrorMessage,
  readBrowserApiPayload,
  requireAuthenticatedPage,
  type AuthenticatedPageProps,
} from "@/lib/api";

export default function ProfileSettings({ currentUser }: AuthenticatedPageProps) {
  const [user, setUser] = useState(currentUser);
  const [name, setName] = useState(currentUser.name);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileSuccess, setProfileSuccess] = useState<string | null>(null);
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null);
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);

  async function handleProfileSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProfileError(null);
    setProfileSuccess(null);
    setSavingProfile(true);

    try {
      const response = await fetch("/api/auth/profile", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          name,
        }),
      });

      const updatedUser = await readBrowserApiPayload<typeof currentUser>(
        response,
        "Failed to update profile"
      );
      setUser(updatedUser);
      setName(updatedUser.name);
      setProfileSuccess("Profile updated.");
    } catch (error) {
      setProfileError(getApiErrorMessage(error, "Failed to update profile"));
    } finally {
      setSavingProfile(false);
    }
  }

  async function handlePasswordSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPasswordError(null);
    setPasswordSuccess(null);

    if (newPassword !== confirmPassword) {
      setPasswordError("New password and confirmation do not match.");
      return;
    }

    setSavingPassword(true);

    try {
      const response = await fetch("/api/auth/password", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });

      await readBrowserApiPayload<{ status: string }>(response, "Failed to update password");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setPasswordSuccess("Password updated.");
    } catch (error) {
      setPasswordError(getApiErrorMessage(error, "Failed to update password"));
    } finally {
      setSavingPassword(false);
    }
  }

  return (
    <>
      <Head>
        <title>Profile Settings - Bifrost</title>
      </Head>
      <SettingsLayout currentUser={user}>
        <div className="p-6">
          <h2 className="mb-6 text-lg font-medium text-foreground">Profile</h2>

          <div className="max-w-xl space-y-8">
            <section>
              <h3 className="mb-4 border-b border-border pb-2 text-sm font-medium text-muted-foreground">
                Basic Info
              </h3>

              <form className="space-y-4" onSubmit={handleProfileSubmit}>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground" htmlFor="profile-name">
                    Name
                  </label>
                  <div className="sm:col-span-2">
                    <input
                      id="profile-name"
                      type="text"
                      className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) => setName(event.target.value)}
                      value={name}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground" htmlFor="profile-email">
                    Email
                  </label>
                  <div className="sm:col-span-2">
                    <input
                      id="profile-email"
                      type="email"
                      className="w-full cursor-not-allowed rounded-md border border-border bg-muted/50 px-3 py-2 text-sm text-muted-foreground focus:outline-none"
                      defaultValue={user.email}
                      disabled
                      readOnly
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground">Role</label>
                  <div className="sm:col-span-2">
                    <span className="inline-flex items-center rounded-md bg-accent px-2.5 py-0.5 text-xs font-medium capitalize text-foreground">
                      {user.role}
                    </span>
                  </div>
                </div>

                {profileError ? (
                  <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {profileError}
                  </div>
                ) : null}
                {profileSuccess ? (
                  <div className="rounded-md border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 px-3 py-2 text-sm text-[hsl(140_50%_48%)]">
                    {profileSuccess}
                  </div>
                ) : null}

                <div className="flex justify-end">
                  <button
                    type="submit"
                    className="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={savingProfile}
                  >
                    {savingProfile ? "Saving..." : "Save Profile"}
                  </button>
                </div>
              </form>
            </section>

            <section>
              <h3 className="mb-4 border-b border-border pb-2 text-sm font-medium text-muted-foreground">
                Security
              </h3>

              <form className="space-y-4" onSubmit={handlePasswordSubmit}>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground" htmlFor="current-password">
                    Current Password
                  </label>
                  <div className="sm:col-span-2">
                    <input
                      id="current-password"
                      type="password"
                      autoComplete="current-password"
                      className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) => setCurrentPassword(event.target.value)}
                      value={currentPassword}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground" htmlFor="new-password">
                    New Password
                  </label>
                  <div className="sm:col-span-2">
                    <input
                      id="new-password"
                      type="password"
                      autoComplete="new-password"
                      className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) => setNewPassword(event.target.value)}
                      value={newPassword}
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3 sm:items-center sm:gap-4">
                  <label className="text-sm font-medium text-foreground" htmlFor="confirm-password">
                    Confirm Password
                  </label>
                  <div className="sm:col-span-2">
                    <input
                      id="confirm-password"
                      type="password"
                      autoComplete="new-password"
                      className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      onChange={(event) => setConfirmPassword(event.target.value)}
                      value={confirmPassword}
                    />
                  </div>
                </div>

                {passwordError ? (
                  <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {passwordError}
                  </div>
                ) : null}
                {passwordSuccess ? (
                  <div className="rounded-md border border-[hsl(140_50%_48%)]/30 bg-[hsl(140_50%_48%)]/10 px-3 py-2 text-sm text-[hsl(140_50%_48%)]">
                    {passwordSuccess}
                  </div>
                ) : null}

                <div className="flex justify-end">
                  <button
                    type="submit"
                    className="rounded-md border border-border bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
                    disabled={savingPassword}
                  >
                    {savingPassword ? "Updating..." : "Change Password"}
                  </button>
                </div>
              </form>
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
