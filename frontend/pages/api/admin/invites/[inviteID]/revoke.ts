import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const inviteID = Array.isArray(req.query.inviteID) ? req.query.inviteID[0] : req.query.inviteID;
  if (!inviteID) {
    res.status(400).json({
      success: false,
      error: {
        message: "invite id is required",
        code: "INVALID_REQUEST",
        details: null,
      },
    });
    return;
  }

  await proxyApiRequest(req, res, {
    method: "POST",
    path: `/api/v1/admin/invites/${inviteID}/revoke`,
  });
}
