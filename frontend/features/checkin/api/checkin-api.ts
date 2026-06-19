import { authedRequest } from "@/shared/api/authed-client";
import { readAccessToken } from "@/shared/auth/session";
import type { CheckInStatus, CheckInClaim } from "../types/checkin";

export async function getCheckInStatus(): Promise<CheckInStatus> {
  const accessToken = readAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const response = await authedRequest<{ checkIn: CheckInStatus }>(
    "/api/v1/billing/checkin",
    { accessToken }
  );
  return response.checkIn;
}

export async function claimDailyCheckIn(): Promise<CheckInClaim> {
  const accessToken = readAccessToken();
  if (!accessToken) throw new Error("Not authenticated");

  const response = await authedRequest<{ checkIn: CheckInClaim }>(
    "/api/v1/billing/checkin",
    { method: "POST", accessToken }
  );
  return response.checkIn;
}
