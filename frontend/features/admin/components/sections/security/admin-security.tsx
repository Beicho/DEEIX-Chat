"use client";

import * as React from "react";
import { Ban, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { listAdminMultiAccountCandidates, updateAdminUserStatus } from "@/features/admin/api";
import type { AdminMultiAccountCandidateDTO, AdminMultiAccountUserDTO } from "@/features/admin/api/admin.types";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

function formatPercent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

function formatDate(value: string, locale: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString(locale, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function userLabel(user: AdminMultiAccountUserDTO): string {
  return user.displayName || user.username || `#${user.id}`;
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border bg-background px-4 py-3 shadow-sm">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-2 text-2xl font-semibold tracking-tight">{value}</div>
    </div>
  );
}

export function AdminSecurityPage() {
  const t = useTranslations("adminUsers.securityPage");
  const locale = useLocale();
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [items, setItems] = React.useState<AdminMultiAccountCandidateDTO[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [pending, setPending] = React.useState(false);
  const [target, setTarget] = React.useState<AdminMultiAccountCandidateDTO | null>(null);

  const loadItems = React.useCallback(async () => {
    setLoading(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const data = await listAdminMultiAccountCandidates(token);
      setItems(data.candidates ?? []);
    } catch (error) {
      toast.error(t("loadFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setLoading(false);
    }
  }, [resolveErrorMessage, t]);

  React.useEffect(() => {
    void loadItems();
  }, [loadItems]);

  const suspendTarget = React.useCallback(async () => {
    if (!target) return;
    setPending(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const activeUsers = target.users.filter((user) => user.status !== "suspended");
      for (const user of activeUsers) {
        await updateAdminUserStatus(token, user.id, { status: "suspended", reason: t("suspendReason") });
      }
      toast.success(t("suspended", { count: activeUsers.length }));
      setTarget(null);
      await loadItems();
    } catch (error) {
      toast.error(t("suspendFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setPending(false);
    }
  }, [loadItems, resolveErrorMessage, t, target]);

  const summary = React.useMemo(() => {
    const linkedAccountIDs = new Set<number>();
    let highRiskCount = 0;
    let activeToSuspend = 0;
    items.forEach((item) => {
      if (item.riskLevel === "high" || item.confidenceScore >= 0.9) {
        highRiskCount++;
      }
      item.users.forEach((user) => {
        linkedAccountIDs.add(user.id);
        if (user.status !== "suspended") {
          activeToSuspend++;
        }
      });
    });
    return { clusters: items.length, linkedAccounts: linkedAccountIDs.size, highRiskCount, activeToSuspend };
  }, [items]);

  return (
    <section className="space-y-5 px-1">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void loadItems()} disabled={loading}>
          {loading ? <Spinner className="size-3.5" /> : <RefreshCw className="size-3.5" />}
          {t("refresh")}
        </Button>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <SummaryCard label={t("clusters")} value={String(summary.clusters)} />
        <SummaryCard label={t("linkedAccounts")} value={String(summary.linkedAccounts)} />
        <SummaryCard label={t("highRisk")} value={String(summary.highRiskCount)} />
        <SummaryCard label={t("activeToSuspend")} value={String(summary.activeToSuspend)} />
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("cluster")}</TableHead>
            <TableHead>{t("users")}</TableHead>
            <TableHead>{t("risk")}</TableHead>
            <TableHead className="text-right">{t("confidence")}</TableHead>
            <TableHead>{t("detectedAt")}</TableHead>
            <TableHead className="text-right">{t("action")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.length === 0 ? (
            <TableEmptyRow colSpan={6}>{loading ? <Spinner className="mx-auto size-4" /> : t("empty")}</TableEmptyRow>
          ) : (
            items.map((item) => (
              <TableRow key={item.associationID}>
                <TableCell>
                  <div className="space-y-1">
                    <div className="font-medium">#{item.associationID}</div>
                    <div className="max-w-44 truncate text-[11px] text-muted-foreground">{item.fingerprintID}</div>
                    {item.reason ? <div className="max-w-44 truncate text-[11px] text-muted-foreground">{item.reason}</div> : null}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex max-w-xl flex-wrap gap-1.5">
                    {item.users.map((user) => (
                      <Badge key={user.id} variant={user.status === "suspended" ? "secondary" : "outline"} className="rounded-md font-normal">
                        {userLabel(user)} · #{user.id}
                      </Badge>
                    ))}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant={item.riskLevel === "high" || item.confidenceScore >= 0.9 ? "destructive" : "secondary"} className="rounded-md">
                    {item.riskLevel || "-"}
                  </Badge>
                </TableCell>
                <TableCell className="text-right tabular-nums">{formatPercent(item.confidenceScore)}</TableCell>
                <TableCell>{formatDate(item.detectedAt, locale)}</TableCell>
                <TableCell className="text-right">
                  <Button type="button" size="sm" variant="destructive" disabled={pending || item.users.every((user) => user.status === "suspended")} onClick={() => setTarget(item)}>
                    <Ban className="size-3.5" />
                    {t("suspend")}
                  </Button>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <AlertDialog open={Boolean(target)} onOpenChange={(open) => !open && setTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("confirmTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("confirmDescription", { count: target?.users.filter((user) => user.status !== "suspended").length ?? 0 })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction disabled={pending} onClick={() => void suspendTarget()}>
              {pending ? <Spinner className="size-3.5" /> : null}
              {t("confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
