import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  const serverID = req.query.serverID;
  if (typeof serverID !== "string" || serverID.trim() === "") {
    res.status(400).json({
      success: false,
      error: {
        message: "server id is required",
        code: "INVALID_REQUEST",
        details: null,
      },
    });
    return;
  }

  await proxyApiRequest(req, res, {
    method: "GET",
    path: `/api/v1/servers/${encodeURIComponent(serverID)}`,
  });
}
