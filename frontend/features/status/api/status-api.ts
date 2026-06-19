import { authedRequest } from "@/shared/api/authed-client";
import { readAccessToken } from "@/shared/auth/session";
import type { SystemStatus } from "../types/status";

export async function getSystemStatus(): Promise<SystemStatus> {
  // Public endpoint - no auth required, but pass token if available for consistency
  const accessToken = readAccessToken();
  const response = await authedRequest<SystemStatus>(
    "/api/v1/status/models",
    { accessToken: accessToken ?? "" }
  );
  return response;
}
