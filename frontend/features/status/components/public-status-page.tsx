"use client";

import * as React from "react";
import { useLocale, useTranslations } from "next-intl";
import { CircleAlert } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getModelAvailability } from "@/shared/api/status";
import type { ModelAvailabilityDTO, ModelAvailabilityItemDTO } from "@/shared/api/status.types";
import { cn } from "@/lib/utils";

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

function statusTone(status: string): string {
  if (status === "normal") return "bg-primary";
  if (status === "down") return "bg-destructive";
  return "bg-muted-foreground";
}

function statusLabelKey(status: string): "normal" | "degraded" | "down" | "unknown" {
  if (status === "normal" || status === "degraded" || status === "down") return status;
  return "unknown";
}

function formatPercent(value: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 1 }).format(value || 0);
}

function ModelRow({ item }: { item: ModelAvailabilityItemDTO }) {
  const t = useTranslations("status");
  const locale = useLocale();
  const labelKey = statusLabelKey(item.status);

  return (
    <TableRow>
      <TableCell className="max-w-0">
        <div className="truncate font-medium text-foreground">{item.modelName}</div>
      </TableCell>
      <TableCell>
        <div className="inline-flex items-center gap-2 text-sm text-foreground">
          <span className={cn("size-2 rounded-full", statusTone(item.status))} aria-hidden="true" />
          <span>{t(`status.${labelKey}`)}</span>
        </div>
      </TableCell>
      <TableCell>{formatPercent(item.successRate, locale)}</TableCell>
      <TableCell>{new Intl.NumberFormat(locale).format(item.callCount)}</TableCell>
    </TableRow>
  );
}

export function PublicStatusPage() {
  const t = useTranslations("status");
  const locale = useLocale();
  const [data, setData] = React.useState<ModelAvailabilityDTO | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [failed, setFailed] = React.useState(false);

  React.useEffect(() => {
    let active = true;
    setLoading(true);
    getModelAvailability()
      .then((next) => {
        if (!active) return;
        setData(next);
        setFailed(false);
      })
      .catch(() => {
        if (!active) return;
        setFailed(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const updatedAt = data?.generatedAt ? formatDate(data.generatedAt, locale) : "";
  const windowHours = data?.windowHours ?? 24;

  return (
    <main className="min-h-dvh bg-background px-4 py-8 text-foreground md:px-6 md:py-12">
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        <header className="space-y-2">
          <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">{t("title")}</h1>
          <p className="max-w-2xl text-sm text-muted-foreground md:text-base">{t("description", { hours: windowHours })}</p>
          {updatedAt ? <p className="text-xs text-muted-foreground">{t("updatedAt", { time: updatedAt })}</p> : null}
        </header>

        {failed ? (
          <Alert variant="destructive">
            <CircleAlert className="size-4" />
            <AlertDescription>{t("loadFailed")}</AlertDescription>
          </Alert>
        ) : null}

        <Card>
          <CardHeader>
            <CardTitle>{t("title")}</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-2/3" />
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t("table.model")}</TableHead>
                      <TableHead>{t("table.status")}</TableHead>
                      <TableHead>{t("table.successRate")}</TableHead>
                      <TableHead>{t("table.calls")}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data?.models.length ? (
                      data.models.map((item) => <ModelRow key={item.modelName} item={item} />)
                    ) : (
                      <TableEmptyRow colSpan={4}>
                        <div className="space-y-1">
                          <p className="font-medium text-foreground">{t("emptyTitle")}</p>
                          <p>{t("emptyDescription")}</p>
                        </div>
                      </TableEmptyRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
