import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const userID = Array.isArray(req.query.userID) ? req.query.userID[0] : req.query.userID;
  if (!userID) {
    res.status(400).json({
      success: false,
      error: {
        message: "user id is required",
        code: "INVALID_REQUEST",
        details: null,
      },
    });
    return;
  }

  await proxyApiRequest(req, res, {
    method: "POST",
    path: `/api/v1/admin/viewers/${userID}/disable`,
  });
}
