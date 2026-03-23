import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const systemID = req.query.systemID;
  if (typeof systemID !== "string" || systemID.trim() === "") {
    res.status(400).json({
      success: false,
      error: {
        message: "system id is required",
        code: "INVALID_REQUEST",
        details: null,
      },
    });
    return;
  }

  await proxyApiRequest(req, res, {
    method: "POST",
    path: `/api/v1/admin/systems/${encodeURIComponent(systemID)}/cancel`,
  });
}
