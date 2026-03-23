import Head from "next/head";
import Link from "next/link";
import type { GetServerSideProps } from "next";
import { useRouter } from "next/router";
import { useState, type FormEvent } from "react";

import { AuthShell } from "@/components/AuthShell";
import { fetchBootstrapStatus, fetchOptionalSession, getApiErrorMessage } from "@/lib/api";

type LoginPageProps = {
  initialError: string | null;
};

type ApiSuccess = {
  success: boolean;
};

const inputClassName =
  "h-11 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring";

const authServiceError =
  "Auth service is unavailable. Ensure the backend is running and Postgres migrations have been applied.";

export default function LoginPage({ initialError }: LoginPageProps) {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(initialError);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email,
          password,
        }),
      });

      const payload = (await response.json()) as ApiSuccess & {
        error?: { message?: string };
      };

      if (!response.ok || !payload.success) {
        throw new Error(payload.error?.message || "Login failed");
      }

      await router.push("/");
    } catch (submitError) {
      setError(getApiErrorMessage(submitError, "Login failed"));
      setSubmitting(false);
    }
  }

  return (
    <>
      <Head>
        <title>Login · Bifrost</title>
      </Head>
      <AuthShell
        eyebrow="Dashboard Access"
        title="Sign in to your tenant"
        subtitle="Use the admin or viewer account that belongs to this Bifrost install."
        footer={
          <span>
            First run?{" "}
            <Link className="text-foreground underline decoration-border underline-offset-4" href="/setup">
              Bootstrap the admin account
            </Link>
            .
          </span>
        }
      >
        <form className="grid gap-4" onSubmit={handleSubmit}>
          <label className="grid gap-2">
            <span className="text-sm font-medium text-foreground">Email</span>
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
              autoComplete="current-password"
              className={inputClassName}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter your password"
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
            {submitting ? "Signing in..." : "Sign In"}
          </button>
        </form>
      </AuthShell>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<LoginPageProps> = async (context) => {
  try {
    const currentUser = await fetchOptionalSession(context);
    if (currentUser) {
      return {
        redirect: {
          destination: "/",
          permanent: false,
        },
      };
    }
  } catch {
    return { props: { initialError: authServiceError } };
  }

  try {
    const bootstrap = await fetchBootstrapStatus(context);
    if (bootstrap.needs_bootstrap) {
      return {
        redirect: {
          destination: "/setup",
          permanent: false,
        },
      };
    }
  } catch {
    return { props: { initialError: authServiceError } };
  }

  return { props: { initialError: null } };
};
