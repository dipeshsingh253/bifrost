import type { NextApiRequest, NextApiResponse } from "next";

import { getApiBaseUrl } from "@/lib/api";

type ProxyOptions = {
	allowBody?: boolean;
	method: "DELETE" | "GET" | "PATCH" | "POST";
	path: string;
};

export async function proxyApiRequest(
  req: NextApiRequest,
  res: NextApiResponse,
  options: ProxyOptions
): Promise<void> {
  if (req.method !== options.method) {
    res.setHeader("Allow", options.method);
    res.status(405).json({
      success: false,
      error: {
        message: "method not allowed",
        code: "METHOD_NOT_ALLOWED",
        details: null,
      },
    });
    return;
  }

  const headers = new Headers();
  if (req.headers.cookie) {
    headers.set("Cookie", req.headers.cookie);
  }
  if (options.allowBody) {
    headers.set("Content-Type", "application/json");
  }

  try {
    const response = await fetch(`${getApiBaseUrl()}${options.path}`, {
      body: options.allowBody ? JSON.stringify(req.body ?? {}) : undefined,
      headers,
      method: options.method,
    });

    const setCookie = response.headers.get("set-cookie");
    if (setCookie) {
      res.setHeader("Set-Cookie", setCookie);
    }

    const contentType = response.headers.get("content-type");
    if (contentType?.toLowerCase().includes("application/json")) {
      const body = await response.json();
      res.status(response.status).json(body);
      return;
    }

    if (contentType) {
      res.setHeader("Content-Type", contentType);
    }

    const body = await response.text();
    res.status(response.status).send(body);
  } catch {
    res.status(502).json({
      success: false,
      error: {
        message: "failed to reach backend service",
        code: "API_PROXY_FAILED",
        details: null,
      },
    });
  }
}
