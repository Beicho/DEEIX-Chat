import { authedRequest } from "@/shared/api/authed-client";
import type { PagePayload } from "@/shared/api/common.types";
import type { ModerationEventDTO } from "@/shared/api/moderation.types";

export async function listModerationEvents(
  accessToken: string,
  options: { page?: number; pageSize?: number } = {},
): Promise<PagePayload<ModerationEventDTO>> {
  const params = new URLSearchParams();
  if (options.page) params.set("page", String(options.page));
  if (options.pageSize) params.set("page_size", String(options.pageSize));
  const query = params.toString();
  return authedRequest<PagePayload<ModerationEventDTO>>(
    `/api/v1/admin/moderation/events${query ? `?${query}` : ""}`,
    { accessToken },
  );
}
