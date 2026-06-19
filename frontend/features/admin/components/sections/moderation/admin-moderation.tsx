"use client";

import * as React from "react";
import { CheckCircle2, Eye, MoreHorizontal, RefreshCw, RotateCcw, Save, ShieldCheck, XCircle } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { TablePagination } from "@/components/ui/table-tools";
import { Textarea } from "@/components/ui/textarea";
import { SettingsFieldEditor } from "../shared/settings-runtime-panel";
import { listAdminSettings, patchAdminSettings } from "@/features/admin/api";
import {
  buildModerationSettingsFields,
  flattenModerationSettings,
  moderationFieldID,
  resolveModerationSettingsError,
  toModerationEditorField,
  type ModerationSettingsField,
} from "@/features/admin/model/moderation-settings";
import {
  SettingsFieldItem,
  SettingsFieldList,
  SettingsSection,
} from "@/shared/components/settings-layout";
import { configuredSettingsMap } from "@/shared/lib/settings-meta";
import { listModerationEvents, releaseModerationDisposition, updateModerationReview } from "@/shared/api/moderation";
import type { ModerationEventDTO, ModerationReviewRequest } from "@/shared/api/moderation.types";
import type { PatchSettingItem } from "@/shared/api/settings.types";
import { resolveAdminErrorMessage } from "@/features/admin/utils/admin-error";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { toast } from "sonner";

const PAGE_SIZE = 20;
const ALL_FILTER_VALUE = "all";
type ReviewStatus = ModerationReviewRequest["status"];
type ModerationDialogState =
  | { type: "review"; item: ModerationEventDTO; status: ReviewStatus }
  | { type: "release"; item: ModerationEventDTO };

function formatDate(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatScore(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { maximumFractionDigits: 3 }).format(value || 0);
}

function filterValue(value: string): string {
  return value === ALL_FILTER_VALUE ? "" : value;
}

function DetailRow({ label, value }: { label: React.ReactNode; value: React.ReactNode }) {
  return (
    <div className="grid gap-1 rounded-md border border-border/60 bg-muted/20 p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="min-w-0 break-words text-sm text-foreground">{value}</div>
    </div>
  );
}

export function AdminModerationPage() {
  const t = useTranslations("adminModeration");
  const locale = useLocale();
  const { accessToken } = useAuthSession();
  const settingsFields = React.useMemo(() => buildModerationSettingsFields(t), [t]);
  const [items, setItems] = React.useState<ModerationEventDTO[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(PAGE_SIZE);
  const [loading, setLoading] = React.useState(true);
  const [dialog, setDialog] = React.useState<ModerationDialogState | null>(null);
  const [detailItem, setDetailItem] = React.useState<ModerationEventDTO | null>(null);
  const [note, setNote] = React.useState("");
  const [pendingAction, setPendingAction] = React.useState(false);
  const [userIDFilter, setUserIDFilter] = React.useState("");
  const [directionFilter, setDirectionFilter] = React.useState(ALL_FILTER_VALUE);
  const [reviewStatusFilter, setReviewStatusFilter] = React.useState(ALL_FILTER_VALUE);
  const [dispositionFilter, setDispositionFilter] = React.useState(ALL_FILTER_VALUE);
  const [eventTypeFilter, setEventTypeFilter] = React.useState(ALL_FILTER_VALUE);
  const [flaggedFilter, setFlaggedFilter] = React.useState(ALL_FILTER_VALUE);
  const [createdFromFilter, setCreatedFromFilter] = React.useState("");
  const [createdToFilter, setCreatedToFilter] = React.useState("");
  const [settingsLoading, setSettingsLoading] = React.useState(true);
  const [settingsSaving, setSettingsSaving] = React.useState(false);
  const [settingsMap, setSettingsMap] = React.useState<Record<string, string>>({});
  const [savedSettingsMap, setSavedSettingsMap] = React.useState<Record<string, string>>({});
  const [configuredMap, setConfiguredMap] = React.useState<Record<string, boolean>>({});

  const loadItems = React.useCallback((activeRef?: { active: boolean }) => {
    setLoading(true);
    return listModerationEvents(accessToken, {
      page,
      pageSize,
      userID: userIDFilter.trim(),
      direction: filterValue(directionFilter),
      reviewStatus: filterValue(reviewStatusFilter),
      disposition: filterValue(dispositionFilter),
      eventType: filterValue(eventTypeFilter),
      flagged: filterValue(flaggedFilter),
      createdFrom: createdFromFilter,
      createdTo: createdToFilter,
    })
      .then((data) => {
        if (activeRef && !activeRef.active) return;
        setItems(data.results);
        setTotal(data.total);
      })
      .catch((error) => {
        if (!activeRef || activeRef.active) {
          toast.error(t("loadFailed"), { description: resolveAdminErrorMessage(error) });
        }
      })
      .finally(() => {
        if (!activeRef || activeRef.active) setLoading(false);
      });
  }, [
    accessToken,
    createdFromFilter,
    createdToFilter,
    directionFilter,
    dispositionFilter,
    eventTypeFilter,
    flaggedFilter,
    page,
    pageSize,
    reviewStatusFilter,
    t,
    userIDFilter,
  ]);

  const loadSettings = React.useCallback((activeRef?: { active: boolean }) => {
    setSettingsLoading(true);
    return listAdminSettings(accessToken)
      .then((grouped) => {
        if (activeRef && !activeRef.active) return;
        const flattened = flattenModerationSettings(grouped);
        setSettingsMap(flattened);
        setSavedSettingsMap(flattened);
        setConfiguredMap(configuredSettingsMap(grouped));
      })
      .catch((error) => {
        if (!activeRef || activeRef.active) {
          toast.error(t("settings.toast.loadFailed"), { description: resolveModerationSettingsError(error) });
        }
      })
      .finally(() => {
        if (!activeRef || activeRef.active) setSettingsLoading(false);
      });
  }, [accessToken, t]);

  React.useEffect(() => {
    const activeRef = { active: true };
    void loadItems(activeRef);
    return () => {
      activeRef.active = false;
    };
  }, [loadItems]);

  React.useEffect(() => {
    const activeRef = { active: true };
    void loadSettings(activeRef);
    return () => {
      activeRef.active = false;
    };
  }, [loadSettings]);

  const dirtySettingIDs = React.useMemo(() => {
    const result = new Set<string>();
    for (const field of settingsFields) {
      const id = moderationFieldID(field);
      if ((settingsMap[id] ?? "") !== (savedSettingsMap[id] ?? "")) {
        result.add(id);
      }
    }
    return result;
  }, [savedSettingsMap, settingsFields, settingsMap]);

  const saveSettings = React.useCallback(async () => {
    const changedItems: PatchSettingItem[] = settingsFields
      .filter((field) => dirtySettingIDs.has(moderationFieldID(field)))
      .map((field) => ({
        namespace: field.namespace,
        key: field.key,
        value: settingsMap[moderationFieldID(field)] ?? "",
      }));
    if (!changedItems.length) return;
    setSettingsSaving(true);
    try {
      const grouped = await patchAdminSettings(accessToken, { items: changedItems });
      const flattened = flattenModerationSettings(grouped);
      setSettingsMap(flattened);
      setSavedSettingsMap(flattened);
      setConfiguredMap(configuredSettingsMap(grouped));
      toast.success(t("settings.toast.saved"));
    } catch (error) {
      toast.error(t("settings.toast.saveFailed"), { description: resolveModerationSettingsError(error) });
    } finally {
      setSettingsSaving(false);
    }
  }, [accessToken, dirtySettingIDs, settingsFields, settingsMap, t]);

  const openDialog = React.useCallback((next: ModerationDialogState) => {
    setDialog(next);
    setNote(next.type === "review" ? next.item.reviewNote || "" : "");
  }, []);

  const resetFilters = React.useCallback(() => {
    setUserIDFilter("");
    setDirectionFilter(ALL_FILTER_VALUE);
    setReviewStatusFilter(ALL_FILTER_VALUE);
    setDispositionFilter(ALL_FILTER_VALUE);
    setEventTypeFilter(ALL_FILTER_VALUE);
    setFlaggedFilter(ALL_FILTER_VALUE);
    setCreatedFromFilter("");
    setCreatedToFilter("");
    setPage(1);
  }, []);

  const closeDialog = React.useCallback(() => {
    if (pendingAction) return;
    setDialog(null);
    setNote("");
  }, [pendingAction]);

  const submitDialog = React.useCallback(async () => {
    if (!dialog) return;
    setPendingAction(true);
    try {
      if (dialog.type === "review") {
        await updateModerationReview(accessToken, dialog.item.id, { status: dialog.status, note: note.trim() });
        toast.success(t("toast.reviewSaved"));
      } else {
        await releaseModerationDisposition(accessToken, dialog.item.id, note.trim());
        toast.success(t("toast.releaseSaved"));
      }
      setDialog(null);
      setNote("");
      await loadItems();
    } catch (error) {
      toast.error(t(dialog.type === "review" ? "toast.reviewFailed" : "toast.releaseFailed"), {
        description: resolveAdminErrorMessage(error),
      });
    } finally {
      setPendingAction(false);
    }
  }, [accessToken, dialog, loadItems, note, t]);

  const directionLabel = (value: string) => {
    if (value === "input" || value === "output") return t(`direction.${value}`);
    return value;
  };
  const actionLabel = (value: string) => {
    if (value === "block") return t("action.block");
    return value;
  };
  const reasonLabel = (value: string) => {
    if (value === "blocked" || value === "check_failed") return t(`reason.${value}`);
    return value;
  };
  const eventTypeLabel = (value: string) => {
    if (value === "policy_hit" || value === "engine_error") return t(`eventType.${value}`);
    return value || t("none");
  };
  const statusLabel = (value: string) => {
    if (value === "pending" || value === "false_positive" || value === "confirmed" || value === "resolved") {
      return t(`status.${value}`);
    }
    return value || t("none");
  };
  const dispositionLabel = (value: string) => {
    if (value === "rate_limited" || value === "suspended") return t(`disposition.${value}`);
    return t("disposition.none");
  };
  const optionalDateLabel = React.useCallback((value?: string | null) => (
    value ? formatDate(value, locale) : t("none")
  ), [locale, t]);
  const actionDialogTitle =
    dialog?.type === "review" ? t(`dialog.reviewTitle.${dialog.status}`) : dialog?.type === "release" ? t("dialog.releaseTitle") : "";
  const actionDialogDescription =
    dialog?.type === "review"
      ? t("dialog.reviewDescription", { id: dialog.item.id })
      : dialog?.type === "release"
        ? t("dialog.releaseDescription", { id: dialog.item.id })
        : "";
  const renderActionMenu = React.useCallback((item: ModerationEventDTO) => (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          className="min-h-11 min-w-11 text-muted-foreground shadow-none md:min-h-6 md:min-w-6"
          aria-label={t("actions.openMenu")}
        >
          <MoreHorizontal className="size-3.5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-48">
        <DropdownMenuItem className="min-h-11 md:min-h-8" onSelect={() => setDetailItem(item)}>
          <Eye className="size-3.5" />
          {t("actions.viewDetails")}
        </DropdownMenuItem>
        <DropdownMenuItem className="min-h-11 md:min-h-8" onSelect={() => openDialog({ type: "review", item, status: "confirmed" })}>
          <ShieldCheck className="size-3.5" />
          {t("actions.markConfirmed")}
        </DropdownMenuItem>
        <DropdownMenuItem className="min-h-11 md:min-h-8" onSelect={() => openDialog({ type: "review", item, status: "false_positive" })}>
          <XCircle className="size-3.5" />
          {t("actions.markFalsePositive")}
        </DropdownMenuItem>
        <DropdownMenuItem className="min-h-11 md:min-h-8" onSelect={() => openDialog({ type: "review", item, status: "resolved" })}>
          <CheckCircle2 className="size-3.5" />
          {t("actions.markResolved")}
        </DropdownMenuItem>
        <DropdownMenuItem
          className="min-h-11 md:min-h-8"
          disabled={!item.disposition || Boolean(item.dispositionReleasedAt)}
          onSelect={() => openDialog({ type: "release", item })}
        >
          <RotateCcw className="size-3.5" />
          {t("actions.release")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ), [openDialog, t]);

  const renderSettingsField = React.useCallback((field: ModerationSettingsField, index: number) => {
    const id = moderationFieldID(field);
    return (
      <SettingsFieldItem key={id} index={index}>
        <SettingsFieldEditor
          field={toModerationEditorField(field)}
          value={settingsMap[id] ?? ""}
          configured={configuredMap[id]}
          dirty={(settingsMap[id] ?? "") !== (savedSettingsMap[id] ?? "")}
          disabled={settingsLoading || settingsSaving}
          onChange={(value) => setSettingsMap((prev) => ({ ...prev, [id]: value }))}
        />
      </SettingsFieldItem>
    );
  }, [configuredMap, savedSettingsMap, settingsLoading, settingsMap, settingsSaving]);

  const settingsActions = dirtySettingIDs.size ? (
    <Button type="button" size="sm" disabled={settingsLoading || settingsSaving} onClick={() => void saveSettings()}>
      {settingsSaving ? <Spinner className="size-3.5" /> : <Save className="size-3.5" />}
      {t("settings.actions.save")}
    </Button>
  ) : null;

  return (
    <section className="space-y-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("description")}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void loadItems()} disabled={loading}>
          {loading ? <Spinner className="size-3.5" /> : <RefreshCw className="size-3.5" />}
          {t("actions.refresh")}
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.title")}</CardTitle>
          <CardDescription>{t("settings.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          {settingsLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-3/4" />
            </div>
          ) : (
            <SettingsSection title={t("settings.sectionTitle")} actions={settingsActions}>
              <SettingsFieldList>
                {settingsFields.map((field, index) => renderSettingsField(field, index))}
              </SettingsFieldList>
            </SettingsSection>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("title")}</CardTitle>
          <CardDescription>{t("description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 rounded-lg border border-border/60 bg-muted/20 p-3 md:grid-cols-4">
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.userID")}
              <Input
                value={userIDFilter}
                onChange={(event) => {
                  setUserIDFilter(event.target.value);
                  setPage(1);
                }}
                inputMode="numeric"
                placeholder={t("filters.userIDPlaceholder")}
                className="bg-background"
              />
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.direction")}
              <Select value={directionFilter} onValueChange={(value) => { setDirectionFilter(value); setPage(1); }}>
                <SelectTrigger className="bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value={ALL_FILTER_VALUE}>{t("filters.all")}</SelectItem>
                  <SelectItem value="input">{t("direction.input")}</SelectItem>
                  <SelectItem value="output">{t("direction.output")}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.reviewStatus")}
              <Select value={reviewStatusFilter} onValueChange={(value) => { setReviewStatusFilter(value); setPage(1); }}>
                <SelectTrigger className="bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value={ALL_FILTER_VALUE}>{t("filters.all")}</SelectItem>
                  <SelectItem value="pending">{t("status.pending")}</SelectItem>
                  <SelectItem value="confirmed">{t("status.confirmed")}</SelectItem>
                  <SelectItem value="false_positive">{t("status.false_positive")}</SelectItem>
                  <SelectItem value="resolved">{t("status.resolved")}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.disposition")}
              <Select value={dispositionFilter} onValueChange={(value) => { setDispositionFilter(value); setPage(1); }}>
                <SelectTrigger className="bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value={ALL_FILTER_VALUE}>{t("filters.all")}</SelectItem>
                  <SelectItem value="none">{t("disposition.none")}</SelectItem>
                  <SelectItem value="rate_limited">{t("disposition.rate_limited")}</SelectItem>
                  <SelectItem value="suspended">{t("disposition.suspended")}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.eventType")}
              <Select value={eventTypeFilter} onValueChange={(value) => { setEventTypeFilter(value); setPage(1); }}>
                <SelectTrigger className="bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value={ALL_FILTER_VALUE}>{t("filters.all")}</SelectItem>
                  <SelectItem value="policy_hit">{t("eventType.policy_hit")}</SelectItem>
                  <SelectItem value="engine_error">{t("eventType.engine_error")}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.flagged")}
              <Select value={flaggedFilter} onValueChange={(value) => { setFlaggedFilter(value); setPage(1); }}>
                <SelectTrigger className="bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="start">
                  <SelectItem value={ALL_FILTER_VALUE}>{t("filters.all")}</SelectItem>
                  <SelectItem value="true">{t("flagged.true")}</SelectItem>
                  <SelectItem value="false">{t("flagged.false")}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.createdFrom")}
              <Input
                type="date"
                value={createdFromFilter}
                onChange={(event) => {
                  setCreatedFromFilter(event.target.value);
                  setPage(1);
                }}
                className="bg-background"
              />
            </label>
            <label className="grid gap-1.5 text-xs text-muted-foreground">
              {t("filters.createdTo")}
              <Input
                type="date"
                value={createdToFilter}
                onChange={(event) => {
                  setCreatedToFilter(event.target.value);
                  setPage(1);
                }}
                className="bg-background"
              />
            </label>
            <div className="md:col-span-4">
              <Button type="button" variant="outline" size="sm" onClick={resetFilters}>
                {t("actions.resetFilters")}
              </Button>
            </div>
          </div>
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-3/4" />
            </div>
          ) : (
            <>
            <div className="space-y-3 md:hidden">
              {items.length ? (
                items.map((item) => (
                  <article key={item.id} className="rounded-lg border border-border/60 bg-background p-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0 space-y-1">
                        <div className="truncate text-sm font-medium">{formatDate(item.createdAt, locale)}</div>
                        <div className="flex flex-wrap gap-1.5">
                          <Badge variant={item.reviewStatus === "pending" ? "outline" : "secondary"} className="rounded-md">
                            {statusLabel(item.reviewStatus)}
                          </Badge>
                          <Badge variant={item.disposition ? "destructive" : "outline"} className="rounded-md">
                            {dispositionLabel(item.disposition)}
                          </Badge>
                        </div>
                      </div>
                      {renderActionMenu(item)}
                    </div>
                    <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-2">
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.direction")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{directionLabel(item.direction)}</dd>
                      </div>
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.action")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{actionLabel(item.action)}</dd>
                      </div>
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.user")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{item.userID}</dd>
                      </div>
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.conversation")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{item.conversationID}</dd>
                      </div>
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.score")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{formatScore(item.score, locale)}</dd>
                      </div>
                      <div className="grid gap-0.5">
                        <dt className="text-muted-foreground">{t("fields.threshold")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{formatScore(item.threshold, locale)}</dd>
                      </div>
                      <div className="grid gap-0.5 sm:col-span-2">
                        <dt className="text-muted-foreground">{t("fields.reason")}</dt>
                        <dd className="min-w-0 truncate text-foreground">{reasonLabel(item.reason)}</dd>
                      </div>
                    </dl>
                  </article>
                ))
              ) : (
                <div className="rounded-lg border border-border/60 bg-muted/20 p-6 text-center text-sm text-muted-foreground">
                  {t("empty")}
                </div>
              )}
            </div>
            <div className="hidden overflow-x-auto md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("fields.time")}</TableHead>
                    <TableHead>{t("fields.direction")}</TableHead>
                    <TableHead>{t("fields.action")}</TableHead>
                    <TableHead>{t("fields.score")}</TableHead>
                    <TableHead>{t("fields.threshold")}</TableHead>
                    <TableHead>{t("fields.user")}</TableHead>
                    <TableHead>{t("fields.conversation")}</TableHead>
                    <TableHead>{t("fields.status")}</TableHead>
                    <TableHead>{t("fields.disposition")}</TableHead>
                    <TableHead>{t("fields.reason")}</TableHead>
                    <TableHead stickyEnd className="w-10" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.length ? (
                    items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell className="whitespace-nowrap">{formatDate(item.createdAt, locale)}</TableCell>
                        <TableCell>{directionLabel(item.direction)}</TableCell>
                        <TableCell>{actionLabel(item.action)}</TableCell>
                        <TableCell>{formatScore(item.score, locale)}</TableCell>
                        <TableCell>{formatScore(item.threshold, locale)}</TableCell>
                        <TableCell>{item.userID}</TableCell>
                        <TableCell>{item.conversationID}</TableCell>
                        <TableCell>
                          <Badge variant={item.reviewStatus === "pending" ? "outline" : "secondary"} className="rounded-md">
                            {statusLabel(item.reviewStatus)}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={item.disposition ? "destructive" : "outline"} className="rounded-md">
                            {dispositionLabel(item.disposition)}
                          </Badge>
                        </TableCell>
                        <TableCell className="max-w-48 truncate" title={reasonLabel(item.reason)}>
                          {reasonLabel(item.reason)}
                        </TableCell>
                        <TableCell stickyEnd className="w-10 text-right">
                          {renderActionMenu(item)}
                        </TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableEmptyRow colSpan={11}>{t("empty")}</TableEmptyRow>
                  )}
                </TableBody>
              </Table>
            </div>
            </>
          )}
          <TablePagination
            page={page}
            pageSize={pageSize}
            pageCount={Math.max(1, Math.ceil(total / pageSize))}
            total={total}
            onPageChange={setPage}
            onPageSizeChange={(next) => {
              setPageSize(next);
              setPage(1);
            }}
            loading={loading}
          />
        </CardContent>
      </Card>
      <Dialog open={Boolean(dialog)} onOpenChange={(open) => !open && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{actionDialogTitle}</DialogTitle>
            <DialogDescription>{actionDialogDescription}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <label className="text-xs font-medium text-foreground" htmlFor="moderation-review-note">
              {t("dialog.noteLabel")}
            </label>
            <Textarea
              id="moderation-review-note"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder={t("dialog.notePlaceholder")}
              disabled={pendingAction}
              className="min-h-24"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={closeDialog} disabled={pendingAction}>
              {t("actions.cancel")}
            </Button>
            <Button type="button" onClick={() => void submitDialog()} disabled={pendingAction}>
              {pendingAction ? <Spinner className="size-3.5" /> : null}
              {t("actions.submit")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={Boolean(detailItem)} onOpenChange={(open) => !open && setDetailItem(null)}>
        <DialogContent className="sm:max-w-[720px]">
          <DialogHeader>
            <DialogTitle>{t("detail.title", { id: detailItem?.id ?? 0 })}</DialogTitle>
            <DialogDescription>{t("detail.description")}</DialogDescription>
          </DialogHeader>
          {detailItem ? (
            <div className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <DetailRow label={t("fields.time")} value={formatDate(detailItem.createdAt, locale)} />
                <DetailRow label={t("fields.eventType")} value={eventTypeLabel(detailItem.eventType)} />
                <DetailRow label={t("fields.direction")} value={directionLabel(detailItem.direction)} />
                <DetailRow label={t("fields.action")} value={actionLabel(detailItem.action)} />
                <DetailRow label={t("fields.user")} value={detailItem.userID} />
                <DetailRow label={t("fields.conversation")} value={detailItem.conversationID} />
                <DetailRow label={t("fields.message")} value={detailItem.messageID || t("none")} />
                <DetailRow label={t("fields.run")} value={detailItem.runID || t("none")} />
                <DetailRow label={t("fields.model")} value={detailItem.model || t("none")} />
                <DetailRow label={t("fields.flagged")} value={detailItem.flagged ? t("flagged.true") : t("flagged.false")} />
                <DetailRow label={t("fields.score")} value={formatScore(detailItem.score, locale)} />
                <DetailRow label={t("fields.threshold")} value={formatScore(detailItem.threshold, locale)} />
                <DetailRow label={t("fields.status")} value={statusLabel(detailItem.reviewStatus)} />
                <DetailRow label={t("fields.reviewedAt")} value={optionalDateLabel(detailItem.reviewedAt)} />
                <DetailRow label={t("fields.disposition")} value={dispositionLabel(detailItem.disposition)} />
                <DetailRow label={t("fields.dispositionAppliedAt")} value={optionalDateLabel(detailItem.dispositionAppliedAt)} />
                <DetailRow label={t("fields.dispositionReleasedAt")} value={optionalDateLabel(detailItem.dispositionReleasedAt)} />
                <DetailRow label={t("fields.dispositionReleasedBy")} value={detailItem.dispositionReleasedBy || t("none")} />
                <DetailRow label={t("fields.contentHash")} value={detailItem.contentHash || t("none")} />
                <DetailRow label={t("fields.snapshotTruncated")} value={detailItem.snapshotTruncated ? t("flagged.true") : t("flagged.false")} />
              </div>
              <div className="grid gap-1.5">
                <div className="text-xs font-medium text-muted-foreground">{t("fields.contentSnapshot")}</div>
                <pre className="max-h-72 overflow-y-auto whitespace-pre-wrap break-words rounded-md border border-border/60 bg-muted/20 p-3 text-xs text-foreground">
                  {detailItem.contentSnapshot || t("detail.emptySnapshot")}
                </pre>
              </div>
              <div className="grid gap-1.5">
                <div className="text-xs font-medium text-muted-foreground">{t("fields.categories")}</div>
                <pre className="max-h-48 overflow-y-auto whitespace-pre-wrap break-words rounded-md border border-border/60 bg-muted/20 p-3 text-xs text-foreground">
                  {detailItem.categoriesJSON || t("none")}
                </pre>
              </div>
              <div className="grid gap-1.5">
                <div className="text-xs font-medium text-muted-foreground">{t("fields.reviewNote")}</div>
                <div className="min-h-16 rounded-md border border-border/60 bg-muted/20 p-3 text-sm text-foreground">
                  {detailItem.reviewNote || t("none")}
                </div>
              </div>
            </div>
          ) : null}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDetailItem(null)}>
              {t("actions.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}
