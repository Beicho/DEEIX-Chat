export type NotificationDTO = {
  id: string;
  type: string;
  title: string;
  body: string;
  link: string;
  readAt: string | null;
  source: string;
  sourceId: string;
  createdAt: string;
  updatedAt: string;
};

export type NotificationListDTO = {
  total: number;
  results: NotificationDTO[];
};

export type NotificationUnreadCountDTO = {
  unreadCount: number;
};
