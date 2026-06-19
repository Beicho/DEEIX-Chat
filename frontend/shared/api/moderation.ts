import { authedRequest } from "@/shared/api/authed-client";
import type { PagePayload } from "@/shared/api/common.types";
import { pathParam } from "@/shared/api/http-client";
import type { ModerationEventDTO, ModerationReviewRequest } from "@/shared/api/moderation.types";

export type ModerationEventListOptions = {
  page?: number;
  pageSize?: number;
  userID?: string;
  direction?: string;
  reviewStatus?: string;
  disposition?: string;
  eventType?: string;
  flagged?: string;
  createdFrom?: string;
  createdTo?: string;
};

export async function listModerationEvents(
  accessToken: string,
  options: ModerationEventListOptions = {},
): Promise<PagePayload<ModerationEventDTO>> {
  const params = new URLSearchParams();
  if (options.page) params.set("page", String(options.page));
  if (options.pageSize) params.set("page_size", String(options.pageSize));
  if (options.userID) params.set("user_id", options.userID);
  if (options.direction) params.set("direction", options.direction);
  if (options.reviewStatus) params.set("review_status", options.reviewStatus);
  if (options.disposition) params.set("disposition", options.disposition);
  if (options.eventType) params.set("event_type", options.eventType);
  if (options.flagged) params.set("flagged", options.flagged);
  if (options.createdFrom) params.set("created_from", options.createdFrom);
  if (options.createdTo) params.set("created_to", options.createdTo);
  const query = params.toString();
  return authedRequest<PagePayload<ModerationEventDTO>>(
    `/api/v1/admin/moderation/events${query ? `?${query}` : ""}`,
    { accessToken },
  );
}

export async function updateModerationReview(
  accessToken: string,
  id: number,
  body: ModerationReviewRequest,
): Promise<ModerationEventDTO> {
  return authedRequest<ModerationEventDTO>(
    `/api/v1/admin/moderation/events/${pathParam(id)}/review`,
    { method: "PATCH", body, accessToken },
  );
}

export async function releaseModerationDisposition(
  accessToken: string,
  id: number,
  note?: string,
): Promise<ModerationEventDTO> {
  return authedRequest<ModerationEventDTO>(
    `/api/v1/admin/moderation/events/${pathParam(id)}/release`,
    { method: "POST", body: { note: note ?? "" }, accessToken },
  );
}
