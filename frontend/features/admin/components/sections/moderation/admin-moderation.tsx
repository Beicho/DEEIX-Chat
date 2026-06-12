"use client";

import * as React from "react";
import { useLocale, useTranslations } from "next-intl";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { TablePagination } from "@/components/ui/table-tools";
import { listModerationEvents } from "@/shared/api/moderation";
import type { ModerationEventDTO } from "@/shared/api/moderation.types";
import { resolveAdminErrorMessage } from "@/features/admin/utils/admin-error";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { toast } from "sonner";

const PAGE_SIZE = 20;

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

export function AdminModerationPage() {
  const t = useTranslations("adminModeration");
  const locale = useLocale();
  const { accessToken } = useAuthSession();
  const [items, setItems] = React.useState<ModerationEventDTO[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(PAGE_SIZE);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    let active = true;
    setLoading(true);
    listModerationEvents(accessToken, { page, pageSize })
      .then((data) => {
        if (!active) return;
        setItems(data.results);
        setTotal(data.total);
      })
      .catch((error) => {
        if (active) toast.error(t("loadFailed"), { description: resolveAdminErrorMessage(error) });
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [accessToken, page, pageSize, t]);

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

  return (
    <section className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("description")}</p>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>{t("title")}</CardTitle>
          <CardDescription>{t("description")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-full" />
              <Skeleton className="h-8 w-3/4" />
            </div>
          ) : (
            <div className="overflow-x-auto">
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
                    <TableHead>{t("fields.reason")}</TableHead>
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
                        <TableCell className="max-w-48 truncate">{reasonLabel(item.reason)}</TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableEmptyRow colSpan={8}>{t("empty")}</TableEmptyRow>
                  )}
                </TableBody>
              </Table>
            </div>
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
    </section>
  );
}
