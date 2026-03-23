import Head from "next/head";
import Link from "next/link";
import type { GetServerSideProps } from "next";
import { useRouter } from "next/router";
import { useState, type FormEvent } from "react";

import { AuthShell } from "@/components/AuthShell";
import {
  fetchInviteDetail,
  fetchOptionalSession,
  getApiErrorMessage,
  isApiRequestError,
  type ViewerInvite,
} from "@/lib/api";

type InvitePageProps = {
  invite: ViewerInvite | null;
  message: string | null;
};

type ApiSuccess = {
  success: boolean;
};

const inputClassName =
  "h-11 w-full rounded-md border border-border bg-background px-3 text-sm text-foreground placeholder:text-muted-foreground outline-none transition focus:ring-1 focus:ring-ring";

export default function InvitePage({ invite, message }: InvitePageProps) {
  const router = useRouter();
  const token = typeof router.query.token === "string" ? router.query.token : "";
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(message);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!invite) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const response = await fetch("/api/auth/invites/accept", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          token,
          name,
          password,
        }),
      });

      const payload = (await response.json()) as ApiSuccess & {
        error?: { message?: string };
      };

      if (!response.ok || !payload.success) {
        throw new Error(payload.error?.message || "Invite acceptance failed");
      }

      await router.push("/");
    } catch (submitError) {
      setError(getApiErrorMessage(submitError, "Invite acceptance failed"));
      setSubmitting(false);
    }
  }

  return (
    <>
      <Head>
        <title>Accept Invite · Bifrost</title>
      </Head>
      <AuthShell
        eyebrow="Viewer Invite"
        title={invite ? "Join this tenant as a viewer" : "Invite unavailable"}
        subtitle={
          invite
            ? `This invite is for ${invite.email}. Viewer access is read-only and scoped to the tenant.`
            : "This invite link is missing, expired, revoked, or has already been used."
        }
        footer={
          <span>
            Already have an account?{" "}
            <Link className="text-foreground underline decoration-border underline-offset-4" href="/login">
              Go to login
            </Link>
            .
          </span>
        }
      >
        {invite ? (
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <div className="rounded-md border border-border bg-background/60 px-4 py-3 text-sm text-muted-foreground">
              <div className="font-medium text-foreground">{invite.email}</div>
              <div className="mt-1 capitalize">Role: {invite.role}</div>
            </div>

            <label className="grid gap-2">
              <span className="text-sm font-medium text-foreground">Name</span>
              <input
                autoComplete="name"
                className={inputClassName}
                onChange={(event) => setName(event.target.value)}
                placeholder="Your name"
                type="text"
                value={name}
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
              {submitting ? "Joining..." : "Accept Invite"}
            </button>
          </form>
        ) : (
          <div className="rounded-md border border-border bg-background/60 px-4 py-3 text-sm text-muted-foreground">
            {message}
          </div>
        )}
      </AuthShell>
    </>
  );
}

export const getServerSideProps: GetServerSideProps<InvitePageProps> = async (context) => {
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
    return {
      props: {
        invite: null,
        message: "Auth service is unavailable. Ensure the backend is running and Postgres migrations have been applied.",
      },
    };
  }

  const token = context.params?.token;
  if (typeof token !== "string") {
    return {
      props: {
        invite: null,
        message: "Invite token is missing.",
      },
    };
  }

  try {
    const invite = await fetchInviteDetail(token, context);
    return {
      props: {
        invite,
        message: null,
      },
    };
  } catch (error) {
    const message = isApiRequestError(error)
      ? error.message
      : "This invite is no longer available.";
    return {
      props: {
        invite: null,
        message,
      },
    };
  }
};
