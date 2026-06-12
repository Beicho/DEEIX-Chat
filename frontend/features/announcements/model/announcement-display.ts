import type { AnnouncementDTO } from "@/shared/api/announcements.types";

export type AnnouncementFilter = "all" | "unread" | "read";

export type AnnouncementType = "critical" | "warning" | "info" | "normal" | "general";

export function normalizeAnnouncementType(value: string): AnnouncementType {
  switch (value) {
    case "critical":
    case "warning":
    case "info":
    case "normal":
    case "general":
      return value;
    default:
      return "general";
  }
}

export function isAnnouncementRead(item: AnnouncementDTO): boolean {
  return Boolean(item.closedAt);
}

export function announcementTime(value: string): number {
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

export function announcementTypeRank(value: string): number {
  switch (normalizeAnnouncementType(value)) {
    case "critical":
      return 5;
    case "warning":
      return 4;
    case "info":
      return 3;
    case "normal":
      return 2;
    default:
      return 1;
  }
}

export function compareAnnouncementReadState(a: AnnouncementDTO, b: AnnouncementDTO): number {
  return Number(isAnnouncementRead(a)) - Number(isAnnouncementRead(b));
}

export function sortAnnouncementsByPriority(items: AnnouncementDTO[]): AnnouncementDTO[] {
  return [...items].sort((a, b) =>
    compareAnnouncementReadState(a, b)
    || Number(b.pinned) - Number(a.pinned)
    || announcementTypeRank(b.type) - announcementTypeRank(a.type)
    || b.priority - a.priority
    || announcementTime(b.updatedAt) - announcementTime(a.updatedAt)
    || b.id - a.id,
  );
}

export function filterAnnouncements(items: AnnouncementDTO[], filter: AnnouncementFilter): AnnouncementDTO[] {
  if (filter === "unread") {
    return items.filter((item) => !isAnnouncementRead(item));
  }
  if (filter === "read") {
    return items.filter(isAnnouncementRead);
  }
  return items;
}

export function announcementTypeAccentClassName(value: string): string {
  switch (normalizeAnnouncementType(value)) {
    case "critical":
      return "before:bg-destructive";
    case "warning":
      return "before:bg-primary/80";
    case "info":
      return "before:bg-primary/60";
    case "normal":
      return "before:bg-muted-foreground/70";
    default:
      return "before:bg-border";
  }
}

export function formatAnnouncementDate(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
}

export function formatAnnouncementDateTime(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function formatAnnouncementTime(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat(locale, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}
