"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Bell, CheckCheck, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import { useSidebarIsMobile, useSidebarVisualState } from "@/components/ui/sidebar";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { cn } from "@/lib/utils";
import {
  getNotificationUnreadCount,
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from "@/shared/api/notifications";
import type { NotificationDTO } from "@/shared/api/notifications.types";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { useIsMobile } from "@/shared/hooks/use-mobile";

type NotificationCenterPopoverProps = {
  variant?: "sidebar" | "icon";
  className?: string;
};

const NOTIFICATION_PAGE_SIZE = 20;
const UNREAD_POLL_INTERVAL_MS = 60_000;

function notificationSummary(body: string): string {
  return body
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/[#>*_[\]()!-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function notificationTime(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat(locale, {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function isRead(item: NotificationDTO): boolean {
  return Boolean(item.readAt);
}

type NotificationSourceKey =
  | "announcement"
  | "auth"
  | "billing"
  | "billing_expiry"
  | "moderation"
  | "mcp"
  | "scheduled_prompt"
  | "weekly_summary"
  | "system";

function notificationSourceKey(item: NotificationDTO): NotificationSourceKey {
  const type = (item.type || "").trim();
  switch (type) {
    case "announcement":
    case "auth":
    case "billing":
    case "billing_expiry":
    case "moderation":
    case "mcp":
    case "scheduled_prompt":
    case "weekly_summary":
      return type;
    default:
      return "system";
  }
}

function NotificationListSkeleton() {
  return (
    <div className="space-y-2 p-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-lg border border-border/60 p-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="mt-2 h-3 w-24" />
        </div>
      ))}
    </div>
  );
}

export function NotificationCenterPopover({ variant = "icon", className }: NotificationCenterPopoverProps) {
  const t = useTranslations("notifications");
  const locale = useLocale();
  const router = useRouter();
  const isMobileViewport = useIsMobile();
  const state = useSidebarVisualState();
  const sidebarMobile = useSidebarIsMobile();
  const { accessToken, userStatus, user } = useAuthSession();
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [open, setOpen] = React.useState(false);
  const [items, setItems] = React.useState<NotificationDTO[]>([]);
  const [total, setTotal] = React.useState(0);
  const [unreadCount, setUnreadCount] = React.useState(0);
  const [loading, setLoading] = React.useState(false);
  const [refreshing, setRefreshing] = React.useState(false);
  const [savingID, setSavingID] = React.useState<string | null>(null);
  const [markingAll, setMarkingAll] = React.useState(false);
  const isMobile = isMobileViewport || sidebarMobile;
  const isSidebar = variant === "sidebar";
  const isCollapsed = isSidebar && !isMobile && state === "collapsed";
  const canLoad = userStatus === "ready" && Boolean(accessToken) && !user?.initialSecurityRequired;

  const loadCount = React.useCallback(async () => {
    if (!canLoad || !accessToken) {
      setUnreadCount(0);
      return;
    }
    try {
      const data = await getNotificationUnreadCount(accessToken);
      setUnreadCount(data.unreadCount);
    } catch {
      setUnreadCount(0);
    }
  }, [accessToken, canLoad]);

  const loadItems = React.useCallback(async (silent = false) => {
    if (!canLoad || !accessToken) {
      setItems([]);
      setTotal(0);
      setLoading(false);
      setRefreshing(false);
      return;
    }
    if (silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    try {
      const [listData, countData] = await Promise.all([
        listNotifications(accessToken, { page: 1, pageSize: NOTIFICATION_PAGE_SIZE }),
        getNotificationUnreadCount(accessToken),
      ]);
      setItems(listData.results);
      setTotal(listData.total);
      setUnreadCount(countData.unreadCount);
    } catch (error) {
      toast.error(t("toast.loadFailed"), { description: resolveErrorMessage(error, t("toast.retryLater")) });
      setItems([]);
      setTotal(0);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [accessToken, canLoad, resolveErrorMessage, t]);

  React.useEffect(() => {
    void loadCount();
    if (!canLoad) {
      return;
    }
    const timer = window.setInterval(() => void loadCount(), UNREAD_POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [canLoad, loadCount]);

  React.useEffect(() => {
    if (open) {
      void loadItems(false);
    }
  }, [loadItems, open]);

  const closePanel = React.useCallback(() => {
    setOpen(false);
  }, []);

  const onOpenCenter = React.useCallback(() => {
    closePanel();
    router.push("/notifications");
  }, [closePanel, router]);

  const onOpenNotification = React.useCallback(async (item: NotificationDTO) => {
    if (!accessToken || savingID) {
      return;
    }
    const wasUnread = !isRead(item);
    if (wasUnread) {
      setSavingID(item.id);
      try {
        await markNotificationRead(accessToken, item.id);
        const readAt = new Date().toISOString();
        setItems((current) => current.map((entry) => (
          entry.id === item.id ? { ...entry, readAt } : entry
        )));
        setUnreadCount((current) => Math.max(0, current - 1));
      } catch (error) {
        toast.error(t("toast.markReadFailed"), { description: resolveErrorMessage(error, t("toast.retryLater")) });
        setSavingID(null);
        return;
      } finally {
        setSavingID(null);
      }
    }
    if (item.link) {
      closePanel();
      router.push(item.link);
    }
  }, [accessToken, closePanel, resolveErrorMessage, router, savingID, t]);

  const onMarkAllRead = React.useCallback(async () => {
    if (!accessToken || unreadCount === 0 || markingAll) {
      return;
    }
    setMarkingAll(true);
    try {
      await markAllNotificationsRead(accessToken);
      const readAt = new Date().toISOString();
      setItems((current) => current.map((entry) => (
        isRead(entry) ? entry : { ...entry, readAt }
      )));
      setUnreadCount(0);
    } catch (error) {
      toast.error(t("toast.markAllReadFailed"), { description: resolveErrorMessage(error, t("toast.retryLater")) });
    } finally {
      setMarkingAll(false);
    }
  }, [accessToken, markingAll, resolveErrorMessage, t, unreadCount]);

  const trigger = isSidebar ? (
    <button
      type="button"
      aria-label={t("open")}
      className={cn(
        "relative flex h-8 items-center rounded-md text-sm transition-colors outline-hidden ring-sidebar-ring focus-visible:ring-2",
        isCollapsed
          ? "w-8 justify-center"
          : "w-full hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
        className,
      )}
    >
      <span className="flex w-8 items-center justify-center">
        <Bell size={18} strokeWidth={1.6} className="text-current" />
      </span>
      <span
        className={cn(
          "ml-1 min-w-0 flex-1 truncate text-left transition-[opacity,max-width,margin-left] duration-200 ease-linear",
          isCollapsed ? "ml-0 max-w-0 opacity-0" : "opacity-100",
        )}
      >
        {t("title")}
      </span>
      {unreadCount > 0 ? (
        <span
          className={cn(
            "absolute flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-xs font-medium leading-none text-primary-foreground tabular-nums",
            isCollapsed ? "right-0 top-0" : "right-2 top-2",
          )}
        >
          {unreadCount > 99 ? t("countOverflow") : unreadCount}
        </span>
      ) : null}
    </button>
  ) : (
    <Button type="button" variant="ghost" size="icon" className={cn("relative size-9 after:absolute after:-inset-1 after:content-['']", className)} aria-label={t("open")}>
      <Bell size={18} strokeWidth={1.6} />
      <span className="sr-only">{t("open")}</span>
      {unreadCount > 0 ? (
        <span className="absolute right-1.5 top-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-xs font-medium leading-none text-primary-foreground tabular-nums">
          {unreadCount > 99 ? t("countOverflow") : unreadCount}
        </span>
      ) : null}
    </Button>
  );

  const content = (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-start justify-between gap-3 border-b border-border/60 px-4 py-3">
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-2">
            <h2 className="truncate text-sm font-semibold text-foreground">{t("title")}</h2>
            {unreadCount > 0 ? (
              <Badge variant="secondary" className="shrink-0">
                {t("unreadCount", { count: unreadCount })}
              </Badge>
            ) : null}
          </div>
          <p className="mt-1 text-xs text-muted-foreground">{t("description")}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="relative size-9 after:absolute after:-inset-1 after:content-['']"
            onClick={() => void loadItems(true)}
            disabled={refreshing || loading}
          >
            <RefreshCw className={cn("size-4", refreshing && "animate-spin")} strokeWidth={1.7} />
            <span className="sr-only">{t("refresh")}</span>
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="relative size-9 after:absolute after:-inset-1 after:content-['']"
            onClick={() => void onMarkAllRead()}
            disabled={unreadCount === 0 || markingAll}
          >
            <CheckCheck className="size-4" strokeWidth={1.7} />
            <span className="sr-only">{t("markAllRead")}</span>
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-2 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
        {loading ? (
          <NotificationListSkeleton />
        ) : items.length === 0 ? (
          <CenteredEmptyState
            className="min-h-56"
            title={t("emptyTitle")}
            description={t("emptyDescription")}
          />
        ) : (
          <div className="space-y-1.5">
            {items.map((item) => {
              const read = isRead(item);
              const summary = notificationSummary(item.body);
              return (
                <button
                  key={item.id}
                  type="button"
                  className={cn(
                    "flex min-h-14 w-full min-w-0 flex-col rounded-lg border border-border/60 px-3 py-2 text-left text-sm transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50",
                    read
                      ? "bg-card/45 text-muted-foreground hover:bg-muted/60 hover:text-foreground"
                      : "bg-card text-foreground hover:bg-muted/70",
                  )}
                  onClick={() => void onOpenNotification(item)}
                  disabled={savingID === item.id}
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="min-w-0 flex-1 truncate font-medium">{item.title}</span>
                    <Badge variant={read ? "outline" : "secondary"} className="shrink-0">
                      {read ? t("read") : t("unread")}
                    </Badge>
                  </span>
                  <span className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
                    {summary || t("emptyBody")}
                  </span>
                  <span className="mt-2 flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
                    <span className="truncate">{t(`sources.${notificationSourceKey(item)}`)}</span>
                    <span className="shrink-0 tabular-nums">{notificationTime(item.updatedAt, locale)}</span>
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      <div className="shrink-0 border-t border-border/60 px-4 py-2">
        <div className="flex min-w-0 items-center justify-between gap-3">
          <p className="min-w-0 truncate text-xs text-muted-foreground">
            {total > items.length ? t("partialList", { count: items.length, total }) : t("center.allLoaded")}
          </p>
          <Button
            type="button"
            variant="ghost"
            className="h-8 shrink-0 rounded-lg px-2 text-xs"
            onClick={onOpenCenter}
          >
            {t("viewAll")}
          </Button>
        </div>
      </div>
    </div>
  );

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={setOpen} direction="bottom">
        <DrawerTrigger asChild>{trigger}</DrawerTrigger>
        <DrawerContent className="flex max-h-[min(64dvh,28rem)] flex-col overflow-hidden rounded-t-xl pb-[env(safe-area-inset-bottom)]">
          <DrawerHeader className="sr-only">
            <DrawerTitle>{t("title")}</DrawerTitle>
            <DrawerDescription>{t("description")}</DrawerDescription>
          </DrawerHeader>
          {content}
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{trigger}</PopoverTrigger>
      <PopoverContent
        side={isSidebar ? "right" : "bottom"}
        align={isSidebar ? "start" : "end"}
        className="flex h-[min(36rem,calc(100svh-3rem))] w-[24rem] max-w-[calc(100vw-2rem)] flex-col overflow-hidden p-0"
      >
        {content}
      </PopoverContent>
    </Popover>
  );
}
