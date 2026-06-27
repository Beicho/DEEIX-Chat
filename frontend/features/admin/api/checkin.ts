import { authedRequest } from "@/shared/api/authed-client";
import { readAccessToken } from "@/shared/auth/session";

export type AdminCheckInStats = {
  todayCheckIns: number;
  activeUsersLast7Days: number;
  totalClaims: number;
  totalRewardUSD: number;
  totalRewardNanousd: number;
  averageConsecutiveDays: number;
};

export type AdminCheckInConfig = {
  rewardUSD: number;
  rewardNanousd: number;
};

export type AdminCheckInView = {
  stats: AdminCheckInStats;
  config: AdminCheckInConfig;
};

function token(): string {
  const value = readAccessToken();
  if (!value) throw new Error("Not authenticated");
  return value;
}

export async function getAdminCheckInView(): Promise<AdminCheckInView> {
  return authedRequest<AdminCheckInView>("/api/v1/admin/checkin", {
    accessToken: token(),
  });
}

export async function updateAdminCheckInConfig(payload: { rewardUSD: number }): Promise<AdminCheckInView> {
  return authedRequest<AdminCheckInView>("/api/v1/admin/checkin/config", {
    method: "PATCH",
    accessToken: token(),
    body: payload,
  });
}
