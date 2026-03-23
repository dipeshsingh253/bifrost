import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const token = Array.isArray(req.query.token) ? req.query.token[0] : req.query.token;
  if (!token) {
    res.status(400).json({
      success: false,
      error: {
        message: "invite token is required",
        code: "INVALID_REQUEST",
        details: null,
      },
    });
    return;
  }

  await proxyApiRequest(req, res, {
    method: "GET",
    path: `/api/v1/auth/invites/${token}`,
  });
}
