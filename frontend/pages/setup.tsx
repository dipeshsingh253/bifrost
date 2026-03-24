import Head from "next/head";
import Link from "next/link";
import type { GetServerSideProps } from "next";
import { useRouter } from "next/router";
import { useState, type FormEvent } from "react";

import { AuthShell } from "@/components/AuthShell";
import {
  fetchBootstrapStatus,
  fetchOptionalSession,
  getApiErrorMessage,
  readBrowserApiPayload,
} from "@/lib/api";

type SetupPageProps = {
  initialError: string | null;
};

const inputClassName =
  "h-11 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring";

const authServiceError =
  "Auth service is unavailable. Ensure the backend is running and Postgres migrations have been applied.";

export default function SetupPage({ initialError }: SetupPageProps) {
  const router = useRouter();
  const [tenantName, setTenantName] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(initialError);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const response = await fetch("/api/auth/bootstrap", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          tenant_name: tenantName,
          name,
          email,
          password,
        }),
      });

      await readBrowserApiPayload<Record<string, never>>(response, "Bootstrap failed");

      await router.push("/");
    } catch (submitError) {
      setError(getApiErrorMessage(submitError, "Bootstrap failed"));
      setSubmitting(false);
    }
  }

  return (
    <>
      <Head>
        <title>Setup · Bifrost</title>
      </Head>
      <AuthShell
        eyebrow="First Run"
        title="Create the first admin account"
        subtitle="This one-time setup boots the self-hosted install, creates the tenant, and signs the admin into the dashboard."
        footer={
          <span>
            Already initialized?{" "}
            <Link className="text-foreground underline decoration-border underline-offset-4" href="/login">
              Go to login
            </Link>
            .
          </span>
        }
      >
        <form className="grid gap-4" onSubmit={handleSubmit}>
          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">Tenant Name</span>
            <input
              className={inputClassName}
              onChange={(event) => setTenantName(event.target.value)}
              placeholder="Acme Infrastructure"
              type="text"
              value={tenantName}
            />
          </label>

          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">Admin Name</span>
            <input
              autoComplete="name"
              className={inputClassName}
              onChange={(event) => setName(event.target.value)}
              placeholder="Dipesh Singh"
              type="text"
              value={name}
            />
          </label>

          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">Admin Email</span>
            <input
              autoComplete="email"
              className={inputClassName}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="admin@example.com"
              type="email"
              value={email}
            />
          </label>

          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">Password</span>
            <input
              autoComplete="new-password"
              className={inputClassName}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Create a password"
              type="password"
              value={password}
            />
          </label>

          {error ? (
            <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          ) : null}

          <button
            className="mt-2 inline-flex h-11 items-center justify-center rounded-md border border-border bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
            disabled={submitting}
            type="submit"
          >
            {submitting ? "Creating admin..." : "Create Admin"}
          </button>
        </form>
      </AuthShell>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<SetupPageProps> = async (context) => {
  try {
    const bootstrap = await fetchBootstrapStatus(context);
    if (!bootstrap.needs_bootstrap) {
      try {
        const currentUser = await fetchOptionalSession(context);
        return {
          redirect: {
            destination: currentUser ? "/" : "/login",
            permanent: false,
          },
        };
      } catch {
        return { props: { initialError: authServiceError } };
      }
    }
  } catch {
    return { props: { initialError: authServiceError } };
  }

  return { props: { initialError: null } };
};
