"use client";

import * as React from "react";
import { RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getAdminDashboard } from "@/features/admin/api";
import type { AdminDashboardDTO } from "@/features/admin/api/admin.types";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";

function formatUSDFromCents(cents: number): string {
  return `$${(cents / 100).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function formatUSDFromNanousd(nanousd: number): string {
  return `$${(nanousd / 1_000_000_000).toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

function formatNumber(value: number, locale: string): string {
  return value.toLocaleString(locale);
}

function formatDateTime(value: string | null, locale: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString(locale, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function defaultDashboard(): AdminDashboardDTO {
  return {
    usage: { recordCount: 0, activeUserCount: 0, callCount: 0, tokenCount: 0, durationSeconds: 0, billedNanousd: 0 },
    sales: { paidOrderCount: 0, baseAmountCents: 0, creditNanousd: 0 },
    topModels: [],
    generatedAt: null,
    periodStart: null,
    periodEnd: null,
  };
}

function StatCard({ label, value, meta }: { label: string; value: string; meta?: string }) {
  return (
    <div className="rounded-xl border bg-background px-4 py-3 shadow-sm transition-colors hover:border-border">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-2 text-2xl font-semibold tracking-tight text-foreground">{value}</div>
      {meta ? <div className="mt-1 text-xs text-muted-foreground">{meta}</div> : null}
    </div>
  );
}

export function AdminDashboardPage() {
  const t = useTranslations("adminUsers.dashboardPage");
  const locale = useLocale();
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [dashboard, setDashboard] = React.useState<AdminDashboardDTO>(() => defaultDashboard());
  const [loading, setLoading] = React.useState(true);

  const loadDashboard = React.useCallback(async () => {
    setLoading(true);
    try {
      const token = await resolveAccessToken();
      if (!token) {
        return;
      }
      const data = await getAdminDashboard(token);
      setDashboard(data.dashboard ?? defaultDashboard());
    } catch (error) {
      toast.error(t("loadFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setLoading(false);
    }
  }, [resolveErrorMessage, t]);

  React.useEffect(() => {
    void loadDashboard();
  }, [loadDashboard]);

  const topModel = dashboard.topModels[0];

  return (
    <section className="space-y-5 px-1">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
          <div className="mt-2 text-xs text-muted-foreground">{t("generatedAt", { time: formatDateTime(dashboard.generatedAt, locale) })}</div>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void loadDashboard()} disabled={loading}>
          {loading ? <Spinner className="mr-2 size-3.5" /> : <RefreshCw className="mr-2 size-3.5" />}
          {loading ? t("refreshing") : t("refresh")}
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label={t("todaySales")}
          value={formatUSDFromCents(dashboard.sales.baseAmountCents)}
          meta={`${t("paidOrders", { count: dashboard.sales.paidOrderCount })} · ${t("creditGranted", { value: formatUSDFromNanousd(dashboard.sales.creditNanousd) })}`}
        />
        <StatCard
          label={t("usageCost")}
          value={formatUSDFromNanousd(dashboard.usage.billedNanousd)}
          meta={`${formatNumber(dashboard.usage.callCount, locale)} ${t("calls")}`}
        />
        <StatCard
          label={t("tokens")}
          value={formatNumber(dashboard.usage.tokenCount, locale)}
          meta={`${formatNumber(dashboard.usage.activeUserCount, locale)} ${t("activeUsers")}`}
        />
        <StatCard
          label={t("topModel")}
          value={topModel?.platformModelName || "-"}
          meta={topModel ? `${formatNumber(topModel.callCount, locale)} ${t("calls")}` : t("empty")}
        />
      </div>

      <div className="rounded-xl border bg-background shadow-sm">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <h3 className="text-sm font-semibold">{t("topModels")}</h3>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("model")}</TableHead>
              <TableHead>{t("activeUsers")}</TableHead>
              <TableHead className="text-right">{t("calls")}</TableHead>
              <TableHead className="text-right">{t("tokens")}</TableHead>
              <TableHead className="text-right">{t("usageCost")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {dashboard.topModels.length === 0 ? (
              <TableEmptyRow colSpan={5}>{loading ? <Spinner className="mx-auto size-4" /> : t("empty")}</TableEmptyRow>
            ) : (
              dashboard.topModels.map((item) => (
                <TableRow key={item.platformModelName}>
                  <TableCell className="font-medium">
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate">{item.platformModelName || "-"}</span>
                      <Badge variant="secondary" className="rounded-md">{t("records", { count: item.recordCount })}</Badge>
                    </div>
                  </TableCell>
                  <TableCell className="tabular-nums">{formatNumber(item.activeUserCount, locale)}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatNumber(item.callCount, locale)}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatNumber(item.tokenCount, locale)}</TableCell>
                  <TableCell className="text-right tabular-nums">{formatUSDFromNanousd(item.billedNanousd)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}
