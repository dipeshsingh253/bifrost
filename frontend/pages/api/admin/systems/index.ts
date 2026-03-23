import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method === "GET") {
    await proxyApiRequest(req, res, {
      method: "GET",
      path: "/api/v1/admin/systems",
    });
    return;
  }

  if (req.method === "POST") {
    await proxyApiRequest(req, res, {
      allowBody: true,
      method: "POST",
      path: "/api/v1/admin/systems",
    });
    return;
  }

  res.setHeader("Allow", "GET, POST");
  res.status(405).json({
    success: false,
    error: {
      message: "method not allowed",
      code: "METHOD_NOT_ALLOWED",
      details: null,
    },
  });
}
