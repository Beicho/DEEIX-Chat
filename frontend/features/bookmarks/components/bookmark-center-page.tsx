"use client";

import * as React from "react";
import { Bookmark, ExternalLink, RefreshCw, Search, X } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { deleteMessageBookmark, listMessageBookmarks, setMessageBookmark } from "@/shared/api/conversation";
import type { MessageBookmarkListItemDTO } from "@/shared/api/conversation.types";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { cn } from "@/lib/utils";

const BOOKMARK_PAGE_SIZE = 40;

function formatBookmarkTime(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function roleLabelKey(role: string) {
  if (role === "assistant") {
    return "assistantMessage";
  }
  if (role === "system") {
    return "systemMessage";
  }
  return "userMessage";
}

function parseTagsInput(value: string) {
  return value
    .split(/[,，]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 12);
}

function BookmarkListSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border/60 p-3">
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="mt-3 h-16 w-full" />
        </div>
      ))}
    </div>
  );
}

export function BookmarkCenterPage() {
  const t = useTranslations("bookmarks");
  const tCommon = useTranslations("common.actions");
  const locale = useLocale();
  const router = useRouter();
  const { accessToken, userStatus } = useAuthSession();
  const [items, setItems] = React.useState<MessageBookmarkListItemDTO[]>([]);
  const [query, setQuery] = React.useState("");
  const [loading, setLoading] = React.useState(true);
  const [refreshing, setRefreshing] = React.useState(false);
  const [editingID, setEditingID] = React.useState<number | null>(null);
  const [noteDraft, setNoteDraft] = React.useState("");
  const [tagsDraft, setTagsDraft] = React.useState("");
  const [savingIDs, setSavingIDs] = React.useState<Set<number>>(() => new Set());

  const load = React.useCallback(async (silent = false, searchQuery = query) => {
    if (!accessToken || userStatus !== "ready") {
      setItems([]);
      setLoading(false);
      return;
    }
    if (silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    try {
      const data = await listMessageBookmarks(accessToken, {
        query: searchQuery,
        page: 1,
        pageSize: BOOKMARK_PAGE_SIZE,
      });
      setItems(data.results ?? []);
    } catch {
      toast.error(t("loadFailed"));
      setItems([]);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [accessToken, query, t, userStatus]);

  React.useEffect(() => {
    const timer = window.setTimeout(() => {
      void load(false, query);
    }, 180);
    return () => window.clearTimeout(timer);
  }, [load, query]);

  const beginEdit = React.useCallback((item: MessageBookmarkListItemDTO) => {
    setEditingID(item.id);
    setNoteDraft(item.note ?? "");
    setTagsDraft((item.tags ?? []).join(", "));
  }, []);

  const cancelEdit = React.useCallback(() => {
    setEditingID(null);
    setNoteDraft("");
    setTagsDraft("");
  }, []);

  const markSaving = React.useCallback((id: number, saving: boolean) => {
    setSavingIDs((prev) => {
      const next = new Set(prev);
      if (saving) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }, []);

  const saveBookmark = React.useCallback(async (item: MessageBookmarkListItemDTO) => {
    markSaving(item.id, true);
    try {
      const result = await setMessageBookmark(accessToken, item.message.publicID, {
        bookmarked: true,
        note: noteDraft,
        tags: parseTagsInput(tagsDraft),
      });
      setItems((prev) => prev.map((entry) => entry.id === item.id ? {
        ...entry,
        note: result.note,
        tags: result.tags,
        updatedAt: new Date().toISOString(),
      } : entry));
      cancelEdit();
      toast.success(t("updated"));
    } catch {
      toast.error(t("updateFailed"));
    } finally {
      markSaving(item.id, false);
    }
  }, [accessToken, cancelEdit, markSaving, noteDraft, t, tagsDraft]);

  const removeBookmark = React.useCallback(async (item: MessageBookmarkListItemDTO) => {
    markSaving(item.id, true);
    try {
      await deleteMessageBookmark(accessToken, item.message.publicID);
      setItems((prev) => prev.filter((entry) => entry.id !== item.id));
      if (editingID === item.id) {
        cancelEdit();
      }
      toast.success(t("removed"));
    } catch {
      toast.error(t("removeFailed"));
    } finally {
      markSaving(item.id, false);
    }
  }, [accessToken, cancelEdit, editingID, markSaving, t]);

  const openConversation = React.useCallback((item: MessageBookmarkListItemDTO) => {
    router.push(`/chat?conversation_id=${encodeURIComponent(item.conversation.publicID)}`);
  }, [router]);

  const emptyState = (
    <CenteredEmptyState
      title={query.trim() ? t("searchEmptyTitle") : t("emptyTitle")}
      description={query.trim() ? t("searchEmptyDescription") : t("emptyDescription")}
    />
  );

  return (
    <main className="h-full min-h-0 overflow-hidden bg-background text-foreground">
      <div className="mx-auto flex h-full min-h-0 w-full max-w-5xl flex-col px-4 py-4 md:px-6">
        <div className="flex flex-col gap-3 border-b border-border/60 pb-4 md:flex-row md:items-end md:justify-between">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <Bookmark className="size-4 text-muted-foreground" strokeWidth={1.8} />
              <h1 className="truncate text-lg font-semibold tracking-tight">{t("title")}</h1>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            className="h-9 gap-2 rounded-lg"
            disabled={refreshing}
            onClick={() => void load(true)}
          >
            <RefreshCw className={cn("size-4", refreshing && "animate-spin")} strokeWidth={1.8} />
            {t("refresh")}
          </Button>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-3 pt-4">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" strokeWidth={1.8} />
            <Input
              value={query}
              placeholder={t("searchPlaceholder")}
              className="h-10 rounded-xl pl-9 pr-9"
              onChange={(event) => setQuery(event.target.value)}
            />
            {query ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1/2 size-8 -translate-y-1/2 rounded-full"
                aria-label={tCommon("reset")}
                onClick={() => setQuery("")}
              >
                <X className="size-4" strokeWidth={1.8} />
              </Button>
            ) : null}
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            {loading ? (
              <BookmarkListSkeleton />
            ) : items.length === 0 ? (
              emptyState
            ) : (
              <div className="space-y-2">
                {items.map((item) => {
                  const editing = editingID === item.id;
                  const saving = savingIDs.has(item.id);
                  const conversationTitle = item.conversation.title?.trim() || item.conversation.publicID;
                  const updatedAt = formatBookmarkTime(item.updatedAt, locale);
                  return (
                    <article key={item.id} className="rounded-xl border border-border/60 bg-card/60 p-3">
                      <div className="flex min-w-0 items-start justify-between gap-3">
                        <div className="min-w-0">
                          <div className="flex min-w-0 flex-wrap items-center gap-2">
                            <Badge variant="outline" className="h-5 rounded-md px-1.5 text-[11px]">
                              {t(roleLabelKey(item.message.role))}
                            </Badge>
                            <button
                              type="button"
                              className="min-w-0 truncate text-left text-xs font-medium text-muted-foreground hover:text-foreground"
                              onClick={() => openConversation(item)}
                            >
                              {t("conversation", { title: conversationTitle })}
                            </button>
                          </div>
                          {updatedAt ? (
                            <p className="mt-1 text-[11px] text-muted-foreground/70">
                              {t("updatedAt", { time: updatedAt })}
                            </p>
                          ) : null}
                        </div>
                        <div className="flex shrink-0 items-center gap-1">
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-8 gap-1.5 rounded-lg px-2 text-xs"
                            onClick={() => openConversation(item)}
                          >
                            <ExternalLink className="size-3.5" strokeWidth={1.8} />
                            {t("openConversation")}
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-8 rounded-lg px-2 text-xs"
                            disabled={saving}
                            onClick={() => beginEdit(item)}
                          >
                            {tCommon("edit")}
                          </Button>
                        </div>
                      </div>

                      <p className="mt-3 line-clamp-4 whitespace-pre-wrap text-sm leading-6 text-foreground [overflow-wrap:anywhere]">
                        {item.message.content}
                      </p>

                      {editing ? (
                        <div className="mt-3 space-y-2 rounded-lg bg-muted/30 p-2">
                          <Textarea
                            value={noteDraft}
                            placeholder={t("notePlaceholder")}
                            className="min-h-16 resize-none rounded-lg bg-background text-sm"
                            onChange={(event) => setNoteDraft(event.target.value)}
                          />
                          <Input
                            value={tagsDraft}
                            placeholder={t("tagsPlaceholder")}
                            className="h-9 rounded-lg bg-background text-sm"
                            onChange={(event) => setTagsDraft(event.target.value)}
                          />
                          <div className="flex justify-end gap-2">
                            <Button type="button" variant="ghost" size="sm" className="h-8 rounded-lg" onClick={cancelEdit}>
                              {tCommon("cancel")}
                            </Button>
                            <Button type="button" size="sm" className="h-8 rounded-lg" disabled={saving} onClick={() => void saveBookmark(item)}>
                              {tCommon("save")}
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="mt-3 flex flex-wrap items-center gap-2">
                          {item.note ? (
                            <span className="inline-flex max-w-full items-center gap-1 rounded-md bg-muted/40 px-2 py-1 text-xs text-muted-foreground">
                              <span className="shrink-0">{t("noteLabel")}</span>
                              <span className="truncate">{item.note}</span>
                            </span>
                          ) : null}
                          {(item.tags ?? []).map((tag) => (
                            <Badge key={tag} variant="secondary" className="rounded-md text-[11px]">
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      )}

                      <div className="mt-3 flex justify-end">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-8 rounded-lg px-2 text-xs text-muted-foreground hover:text-destructive"
                          disabled={saving}
                          onClick={() => void removeBookmark(item)}
                        >
                          {t("remove")}
                        </Button>
                      </div>
                    </article>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
