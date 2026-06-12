"use client";

import * as React from "react";
import { Bell, CheckCheck, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StreamdownRender } from "@/features/chat/components/markdown/streamdown-render";
import {
  announcementTypeAccentClassName,
  filterAnnouncements,
  formatAnnouncementDate,
  formatAnnouncementDateTime,
  isAnnouncementRead,
  normalizeAnnouncementType,
  sortAnnouncementsByPriority,
  type AnnouncementFilter,
} from "@/features/announcements/model/announcement-display";
import { closeAnnouncement, listAnnouncements } from "@/shared/api/announcements";
import type { AnnouncementDTO } from "@/shared/api/announcements.types";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { cn } from "@/lib/utils";

function AnnouncementListSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border/60 p-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="mt-2 h-3 w-24" />
        </div>
      ))}
    </div>
  );
}

export function AnnouncementCenterPage() {
  const t = useTranslations("announcements");
  const locale = useLocale();
  const { accessToken, userStatus } = useAuthSession();
  const [items, setItems] = React.useState<AnnouncementDTO[]>([]);
  const [activeID, setActiveID] = React.useState<number | null>(null);
  const [filter, setFilter] = React.useState<AnnouncementFilter>("all");
  const [loading, setLoading] = React.useState(true);
  const [refreshing, setRefreshing] = React.useState(false);
  const [savingIDs, setSavingIDs] = React.useState<Set<number>>(() => new Set());

  const load = React.useCallback(async (silent = false) => {
    if (!accessToken || userStatus !== "ready") {
      setItems([]);
      setActiveID(null);
      setLoading(false);
      return;
    }
    if (silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    try {
      const nextItems = sortAnnouncementsByPriority(await listAnnouncements(accessToken, { includeDismissed: true }));
      setItems(nextItems);
      setActiveID((current) => {
        if (current && nextItems.some((item) => item.id === current)) {
          return current;
        }
        return nextItems[0]?.id ?? null;
      });
    } catch {
      toast.error(t("center.loadFailed"));
      setItems([]);
      setActiveID(null);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [accessToken, t, userStatus]);

  React.useEffect(() => {
    void load(false);
  }, [load]);

  const filteredItems = React.useMemo(() => filterAnnouncements(items, filter), [filter, items]);
  const unreadItems = React.useMemo(() => items.filter((item) => !isAnnouncementRead(item)), [items]);
  const active = React.useMemo(
    () => filteredItems.find((item) => item.id === activeID) ?? filteredItems[0] ?? null,
    [activeID, filteredItems],
  );

  React.useEffect(() => {
    if (!active && filteredItems.length > 0) {
      setActiveID(filteredItems[0].id);
    }
  }, [active, filteredItems]);

  const setSaving = React.useCallback((ids: number[], saving: boolean) => {
    setSavingIDs((current) => {
      const next = new Set(current);
      for (const id of ids) {
        if (saving) {
          next.add(id);
        } else {
          next.delete(id);
        }
      }
      return next;
    });
  }, []);

  const markRead = React.useCallback(async (item: AnnouncementDTO) => {
    if (!accessToken || isAnnouncementRead(item) || savingIDs.has(item.id)) {
      return;
    }
    setSaving([item.id], true);
    try {
      await closeAnnouncement(accessToken, item.id, item.updatedAt);
      const closedAt = new Date().toISOString();
      setItems((current) => current.map((entry) => (
        entry.id === item.id ? { ...entry, closedAt } : entry
      )));
    } catch {
      toast.error(t("center.markReadFailed"));
    } finally {
      setSaving([item.id], false);
    }
  }, [accessToken, savingIDs, setSaving, t]);

  const markAllRead = React.useCallback(async () => {
    if (!accessToken || unreadItems.length === 0) {
      return;
    }
    const ids = unreadItems.map((item) => item.id);
    setSaving(ids, true);
    try {
      await Promise.all(unreadItems.map((item) => closeAnnouncement(accessToken, item.id, item.updatedAt)));
      const closedAt = new Date().toISOString();
      setItems((current) => current.map((entry) => (
        ids.includes(entry.id) ? { ...entry, closedAt } : entry
      )));
    } catch {
      toast.error(t("center.markReadFailed"));
    } finally {
      setSaving(ids, false);
    }
  }, [accessToken, setSaving, t, unreadItems]);

  return (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden">
      <div className="mx-auto flex h-full min-h-0 w-full max-w-[960px] flex-1 flex-col px-3 pb-8 pt-6 md:pt-15">
        <div className="flex shrink-0 flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="flex size-8 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                <Bell className="size-4" strokeWidth={1.7} />
              </span>
              <h1 className="truncate text-lg font-semibold text-foreground">{t("center.title")}</h1>
              {unreadItems.length > 0 ? (
                <Badge variant="secondary" className="shrink-0">
                  {t("center.unreadCount", { count: unreadItems.length })}
                </Badge>
              ) : null}
            </div>
            <p className="mt-2 max-w-2xl text-sm text-muted-foreground">{t("center.description")}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void load(true)}
              disabled={refreshing}
            >
              <RefreshCw className={cn("size-3.5", refreshing && "animate-spin")} strokeWidth={1.8} />
              {t("center.refresh")}
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => void markAllRead()}
              disabled={unreadItems.length === 0 || savingIDs.size > 0}
            >
              <CheckCheck className="size-3.5" strokeWidth={1.8} />
              {t("center.markAllRead")}
            </Button>
          </div>
        </div>

        <Tabs value={filter} onValueChange={(value) => setFilter(value as AnnouncementFilter)} className="mt-5 shrink-0">
          <TabsList className="grid h-8 w-full grid-cols-3 md:w-[22rem]">
            <TabsTrigger value="all">{t("center.filters.all")}</TabsTrigger>
            <TabsTrigger value="unread">{t("center.filters.unread")}</TabsTrigger>
            <TabsTrigger value="read">{t("center.filters.read")}</TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="mt-4 grid min-h-0 flex-1 gap-3 overflow-hidden md:grid-cols-[20rem_minmax(0,1fr)]">
          <div className="min-h-0 overflow-y-auto rounded-xl border border-border/60 bg-card/45 p-2">
            {loading ? (
              <AnnouncementListSkeleton />
            ) : filteredItems.length === 0 ? (
              <CenteredEmptyState
                className="min-h-52"
                title={t("center.emptyTitle")}
                description={t("center.emptyDescription")}
              />
            ) : (
              <div className="space-y-1.5">
                {filteredItems.map((item) => {
                  const read = isAnnouncementRead(item);
                  const selected = active?.id === item.id;
                  const announcementType = normalizeAnnouncementType(item.type);
                  return (
                    <button
                      key={`${item.id}:${item.updatedAt}`}
                      type="button"
                      className={cn(
                        "relative flex w-full min-w-0 flex-col rounded-lg py-2 pl-3.5 pr-3 text-left text-sm transition-colors before:absolute before:left-1.5 before:top-2.5 before:bottom-2.5 before:w-0.5 before:rounded-full",
                        announcementTypeAccentClassName(item.type),
                        selected ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/70 hover:text-foreground",
                      )}
                      onClick={() => setActiveID(item.id)}
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <span className="min-w-0 flex-1 truncate font-medium">{item.title}</span>
                        <Badge variant={read ? "outline" : "secondary"} className="shrink-0">
                          {read ? t("center.read") : t("center.unread")}
                        </Badge>
                      </span>
                      <span className="mt-1 flex min-w-0 items-center justify-between gap-2 text-[11px] text-muted-foreground">
                        <span className="truncate">{t(`types.${announcementType}`)}</span>
                        <span className="shrink-0 tabular-nums">{formatAnnouncementDate(item.updatedAt, locale)}</span>
                      </span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div className="min-h-0 overflow-y-auto rounded-xl border border-border/60 bg-card/45">
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
                      <Badge variant={isAnnouncementRead(active) ? "outline" : "secondary"}>
                        {isAnnouncementRead(active) ? t("center.read") : t("center.unread")}
                      </Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {formatAnnouncementDateTime(active.updatedAt, locale)}
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="shrink-0"
                    disabled={isAnnouncementRead(active) || savingIDs.has(active.id)}
                    onClick={() => void markRead(active)}
                  >
                    <CheckCheck className="size-3.5" strokeWidth={1.8} />
                    {t("center.markRead")}
                  </Button>
                </div>
                <StreamdownRender content={active.contentMarkdown} className="mt-4 text-sm" />
              </article>
            ) : (
              <CenteredEmptyState
                className="min-h-72"
                title={t("center.emptyTitle")}
                description={t("center.emptyDescription")}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
