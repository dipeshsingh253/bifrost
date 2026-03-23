export type DetailNotFoundKind = "container" | "project" | "server" | null;

function apiErrorCode(error: unknown): string | null {
  if (!error || typeof error !== "object" || !("code" in error)) {
    return null;
  }

  const { code } = error as { code?: unknown };
  return typeof code === "string" ? code : null;
}

export function getProjectDetailNotFoundKind(error: unknown): DetailNotFoundKind {
  const code = apiErrorCode(error);
  if (code === "SERVER_NOT_FOUND") {
    return "server";
  }
  if (code === "PROJECT_NOT_FOUND") {
    return "project";
  }
  return null;
}

export function getContainerDetailNotFoundKind(error: unknown): DetailNotFoundKind {
  const code = apiErrorCode(error);
  if (code === "SERVER_NOT_FOUND") {
    return "server";
  }
  if (code === "CONTAINER_NOT_FOUND") {
    return "container";
  }
  return null;
}
