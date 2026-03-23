import type { NextApiRequest, NextApiResponse } from "next";

import { proxyApiRequest } from "@/lib/auth-proxy";

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  await proxyApiRequest(req, res, {
    method: "POST",
    path: "/api/v1/auth/logout",
  });
}
