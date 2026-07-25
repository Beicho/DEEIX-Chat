import { authedRequest } from "@/shared/api/authed-client";
import { resolveApiBaseURL } from "@/shared/api/http-client";

export type AdminInvitationCode = {
  publicID: string;
  code?: string;
  label: string;
  maxUses: number;
  usedCount: number;
  enabled: boolean;
  expiresAt?: string | null;
  lastUsedAt?: string | null;
  createdAt: string;
};

export type AdminInvitationCodeListData = {
  results: AdminInvitationCode[];
  total: number;
};

export type CreateInvitationCodesInput = {
  label?: string;
  count?: number;
  maxUses?: number;
  expiresAt?: string | null;
};

export async function listAdminInvitationCodes(accessToken: string): Promise<AdminInvitationCodeListData> {
  return authedRequest<AdminInvitationCodeListData>("/api/v1/admin/auth/invitation-codes", { accessToken }, true);
}

export async function createAdminInvitationCodes(
  accessToken: string,
  input: CreateInvitationCodesInput,
): Promise<AdminInvitationCodeListData> {
  return authedRequest<AdminInvitationCodeListData>(
    "/api/v1/admin/auth/invitation-codes",
    { accessToken, method: "POST", body: input },
    true,
  );
}

export async function updateAdminInvitationCode(
  accessToken: string,
  publicID: string,
  enabled: boolean,
): Promise<AdminInvitationCode> {
  return authedRequest<AdminInvitationCode>(
    `/api/v1/admin/auth/invitation-codes/${encodeURIComponent(publicID)}`,
    { accessToken, method: "PATCH", body: { enabled } },
    true,
  );
}

/**
 * 批量生成并直接下载 CSV。明文邀请码只在这一次下载中出现，服务端只保存哈希。
 */
export async function downloadAdminInvitationCodes(
  accessToken: string,
  input: CreateInvitationCodesInput,
): Promise<{ filename: string; blob: Blob }> {
  const response = await fetch(`${resolveApiBaseURL()}/api/v1/admin/auth/invitation-codes/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(input),
    credentials: "include",
  });
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = (await response.json()) as { message?: string; error?: string };
      message = payload.message ?? payload.error ?? message;
    } catch {
      // 保留默认状态码信息
    }
    throw new Error(message);
  }
  const disposition = response.headers.get("Content-Disposition") ?? "";
  const match = /filename=([^;]+)/i.exec(disposition);
  const filename = match?.[1]?.trim().replace(/^"|"$/g, "") || "invitation-codes.csv";
  return { filename, blob: await response.blob() };
}
