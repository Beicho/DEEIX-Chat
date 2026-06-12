import { authedRequest } from "@/shared/api/authed-client";
import { pathParam } from "@/shared/api/http-client";
import type { NotificationListDTO, NotificationUnreadCountDTO } from "@/shared/api/notifications.types";

export type ListNotificationsOptions = {
  page?: number;
  pageSize?: number;
  unread?: boolean;
};

function notificationQuery(options?: ListNotificationsOptions): string {
  const params = new URLSearchParams();
  if (options?.page) {
    params.set("page", String(options.page));
  }
  if (options?.pageSize) {
    params.set("page_size", String(options.pageSize));
  }
  if (options?.unread) {
    params.set("unread", "true");
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

export async function listNotifications(accessToken: string, options?: ListNotificationsOptions): Promise<NotificationListDTO> {
  return authedRequest<NotificationListDTO>(`/api/v1/notifications${notificationQuery(options)}`, { accessToken }, true);
}

export async function getNotificationUnreadCount(accessToken: string): Promise<NotificationUnreadCountDTO> {
  return authedRequest<NotificationUnreadCountDTO>("/api/v1/notifications/unread-count", { accessToken }, true);
}

export async function markNotificationRead(accessToken: string, notificationID: string): Promise<void> {
  await authedRequest<{ read: boolean }>(
    `/api/v1/notifications/${pathParam(notificationID)}/read`,
    { method: "POST", accessToken },
    true,
  );
}

export async function markAllNotificationsRead(accessToken: string): Promise<void> {
  await authedRequest<{ read: boolean }>(
    "/api/v1/notifications/read-all",
    { method: "POST", accessToken },
    true,
  );
}
