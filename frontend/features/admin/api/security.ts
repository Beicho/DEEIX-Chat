import { authedRequest } from "@/shared/api/authed-client";
import type { AdminMultiAccountCandidatesData } from "@/features/admin/api/admin.types";

export async function listAdminMultiAccountCandidates(accessToken: string): Promise<AdminMultiAccountCandidatesData> {
  return authedRequest<AdminMultiAccountCandidatesData>(
    "/api/v1/admin/security/multi-accounts?limit=50",
    { accessToken },
    true,
  );
}
