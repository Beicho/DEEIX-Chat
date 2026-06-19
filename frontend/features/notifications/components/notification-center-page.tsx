"use client";

import * as React from "react";
import { Bell, Check, CheckCheck, ExternalLink, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
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

const NOTIFICATION_PAGE_SIZE = 50;

function notificationSummary(body: string): string {
  return body
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/[#>*_[\]()!-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function notificationDateTime(value: string, locale: string): string {
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

function NotificationPageSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 5 }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border/60 p-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="mt-2 h-3 w-28" />
          <Skeleton className="mt-3 h-10 w-full" />
        </div>
      ))}
    </div>
  );
}

export function NotificationCenterPage() {
  const t = useTranslations("notifications");
  const locale = useLocale();
  const router = useRouter();
  const resolveErrorMessage = useLocalizedErrorMessage();
  const { accessToken, userStatus, user } = useAuthSession();
  const [items, setItems] = React.useState<NotificationDTO[]>([]);
  const [activeID, setActiveID] = React.useState<string | null>(null);
  const [total, setTotal] = React.useState(0);
  const [unreadCount, setUnreadCount] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [loadingMore, setLoadingMore] = React.useState(false);
  const [refreshing, setRefreshing] = React.useState(false);
  const [savingID, setSavingID] = React.useState<string | null>(null);
  const [markingAll, setMarkingAll] = React.useState(false);
  const canLoad = userStatus === "ready" && Boolean(accessToken) && !user?.initialSecurityRequired;

  const load = React.useCallback(async (silent = false, nextPage = 1, append = false) => {
    if (!canLoad || !accessToken) {
      setItems([]);
      setActiveID(null);
      setTotal(0);
      setUnreadCount(0);
      setLoading(false);
      setLoadingMore(false);
      setRefreshing(false);
      return;
    }
    if (append) {
      setLoadingMore(true);
    } else if (silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    try {
      const [data, countData] = await Promise.all([
        listNotifications(accessToken, {
          page: nextPage,
          pageSize: NOTIFICATION_PAGE_SIZE,
        }),
        getNotificationUnreadCount(accessToken),
      ]);
      const loadedItems = data.results ?? [];
      const loadedIDs = new Set(loadedItems.map((item) => item.id));
      setItems((current) => {
        if (!append) {
          return loadedItems;
        }
        const seen = new Set(current.map((item) => item.id));
        return [...current, ...loadedItems.filter((item) => !seen.has(item.id))];
      });
      setTotal(data.total ?? 0);
      setUnreadCount(countData.unreadCount);
      setPage(nextPage);
      setActiveID((current) => {
        if (append) {
          return current ?? loadedItems[0]?.id ?? null;
        }
        if (current && loadedIDs.has(current)) {
          return current;
        }
        return loadedItems[0]?.id ?? null;
      });
    } catch (error) {
      toast.error(t("toast.loadFailed"), { description: resolveErrorMessage(error, t("toast.retryLater")) });
      setItems([]);
      setActiveID(null);
      setTotal(0);
      if (!append) {
        setUnreadCount(0);
      }
    } finally {
      setLoading(false);
      setLoadingMore(false);
      setRefreshing(false);
    }
  }, [accessToken, canLoad, resolveErrorMessage, t]);

  React.useEffect(() => {
    void load(false);
  }, [load]);

  const active = React.useMemo(
    () => items.find((item) => item.id === activeID) ?? items[0] ?? null,
    [activeID, items],
  );
  const hasMore = items.length < total;

  const markRead = React.useCallback(async (item: NotificationDTO) => {
    if (!accessToken || isRead(item) || savingID) {
      return false;
    }
    setSavingID(item.id);
    try {
      await markNotificationRead(accessToken, item.id);
      const readAt = new Date().toISOString();
      setItems((current) => current.map((entry) => (
        entry.id === item.id ? { ...entry, readAt } : entry
      )));
      setUnreadCount((current) => Math.max(0, current - 1));
      return true;
    } catch (error) {
      toast.error(t("toast.markReadFailed"), { description: resolveErrorMessage(error, t("toast.retryLater")) });
      return false;
    } finally {
      setSavingID(null);
    }
  }, [accessToken, resolveErrorMessage, savingID, t]);

  const markAllRead = React.useCallback(async () => {
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

  const openLink = React.useCallback((item: NotificationDTO) => {
    if (item.link) {
      router.push(item.link);
    }
  }, [router]);

  const loadMore = React.useCallback(async () => {
    if (!hasMore || loadingMore || loading || refreshing) {
      return;
    }
    await load(true, page + 1, true);
  }, [hasMore, load, loading, loadingMore, page, refreshing]);

  const openNotificationLink = React.useCallback(async (item: NotificationDTO) => {
    if (!isRead(item)) {
      const saved = await markRead(item);
      if (!saved) {
        return;
      }
    }
    openLink(item);
  }, [markRead, openLink]);

  return (
    <main className="h-full min-h-0 overflow-hidden bg-background text-foreground">
      <div className="mx-auto flex h-full min-h-0 w-full max-w-5xl flex-col px-4 py-4 md:px-6 md:py-6">
        <div className="flex flex-col gap-3 border-b border-border/60 pb-4 md:flex-row md:items-end md:justify-between">
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <span className="flex size-8 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                <Bell className="size-4" strokeWidth={1.7} />
              </span>
              <h1 className="truncate text-lg font-semibold tracking-tight">{t("center.title")}</h1>
              {unreadCount > 0 ? (
                <Badge variant="secondary" className="shrink-0">
                  {t("unreadCount", { count: unreadCount })}
                </Badge>
              ) : null}
            </div>
            <p className="mt-1 text-sm text-muted-foreground">{t("center.description")}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              className="min-h-11 gap-2 rounded-lg md:min-h-9"
              disabled={refreshing || loading}
              onClick={() => void load(true)}
            >
              <RefreshCw className={cn("size-4", refreshing && "animate-spin")} strokeWidth={1.8} />
              {t("refresh")}
            </Button>
            <Button
              type="button"
              className="min-h-11 gap-2 rounded-lg md:min-h-9"
              disabled={unreadCount === 0 || markingAll}
              onClick={() => void markAllRead()}
            >
              <CheckCheck className="size-4" strokeWidth={1.8} />
              {t("markAllRead")}
            </Button>
          </div>
        </div>

        <div className="grid min-h-0 flex-1 gap-3 overflow-hidden pt-4 md:grid-cols-[21rem_minmax(0,1fr)]">
          <section className="min-h-0 overflow-y-auto rounded-xl border border-border/60 bg-card/45 p-2">
            {loading ? (
              <NotificationPageSkeleton />
            ) : items.length === 0 ? (
              <CenteredEmptyState
                className="min-h-64"
                title={t("emptyTitle")}
                description={t("emptyDescription")}
              />
            ) : (
              <div className="space-y-1.5">
                {items.map((item) => {
                  const read = isRead(item);
                  const selected = active?.id === item.id;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className={cn(
                        "flex min-h-14 w-full min-w-0 flex-col rounded-lg px-3 py-2 text-left text-sm transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50",
                        selected ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/70 hover:text-foreground",
                      )}
                      onClick={() => setActiveID(item.id)}
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <span className="min-w-0 flex-1 truncate font-medium">{item.title}</span>
                        <Badge variant={read ? "outline" : "secondary"} className="shrink-0">
                          {read ? t("read") : t("unread")}
                        </Badge>
                      </span>
                      <span className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
                        {notificationSummary(item.body) || t("emptyBody")}
                      </span>
                      <span className="mt-2 flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
                        <span className="truncate">{t(`sources.${notificationSourceKey(item)}`)}</span>
                        <span className="shrink-0 tabular-nums">{notificationDateTime(item.updatedAt, locale)}</span>
                      </span>
                    </button>
                  );
                })}
                {hasMore ? (
                  <Button
                    type="button"
                    variant="ghost"
                    className="mt-2 w-full min-h-11 rounded-lg md:min-h-9"
                    disabled={loadingMore}
                    onClick={() => void loadMore()}
                  >
                    {loadingMore ? t("loadingMore") : t("loadMore")}
                  </Button>
                ) : null}
              </div>
            )}
          </section>

          <section className="min-h-0 overflow-y-auto rounded-xl border border-border/60 bg-card/45">
            {loading ? (
              <div className="p-4">
                <Skeleton className="h-5 w-2/3" />
                <Skeleton className="mt-3 h-3 w-40" />
                <Skeleton className="mt-6 h-28 w-full" />
              </div>
            ) : active ? (
              <article className="p-4">
                <div className="flex min-w-0 flex-col gap-3 border-b border-border/60 pb-3 md:flex-row md:items-start md:justify-between">
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <h2 className="truncate text-base font-semibold text-foreground">{active.title}</h2>
                      <Badge variant={isRead(active) ? "outline" : "secondary"} className="shrink-0">
                        {isRead(active) ? t("read") : t("unread")}
                      </Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {notificationDateTime(active.updatedAt, locale)}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-wrap items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      className="min-h-11 gap-2 md:min-h-9"
                      disabled={isRead(active) || savingID === active.id}
                      onClick={() => void markRead(active)}
                    >
                      <Check className="size-4" strokeWidth={1.8} />
                      {t("center.markRead")}
                    </Button>
                    {active.link ? (
                      <Button
                        type="button"
                        className="min-h-11 gap-2 md:min-h-9"
                        onClick={() => void openNotificationLink(active)}
                      >
                        <ExternalLink className="size-4" strokeWidth={1.8} />
                        {t("center.openLink")}
                      </Button>
                    ) : null}
                  </div>
                </div>
                <div className="mt-4 whitespace-pre-wrap text-sm leading-6 text-foreground">
                  {active.body.trim() || t("emptyBody")}
                </div>
              </article>
            ) : (
              <CenteredEmptyState
                className="min-h-64"
                title={t("emptyTitle")}
                description={t("emptyDescription")}
              />
            )}
          </section>
        </div>

        {total > items.length ? (
          <p className="shrink-0 pt-3 text-xs text-muted-foreground">
            {t("partialList", { count: items.length, total })}
          </p>
        ) : null}
      </div>
    </main>
  );
}
