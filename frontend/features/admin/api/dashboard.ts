import { authedRequest } from "@/shared/api/authed-client";
import type { AdminDashboardData } from "@/features/admin/api/admin.types";

export async function getAdminDashboard(accessToken: string): Promise<AdminDashboardData> {
  return authedRequest<AdminDashboardData>(
    "/api/v1/admin/dashboard",
    { accessToken },
    true,
  );
}
