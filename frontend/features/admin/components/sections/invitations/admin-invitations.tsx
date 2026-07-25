"use client";

import * as React from "react";
import { Copy, Download, Plus, RefreshCw } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableEmptyRow, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  createAdminInvitationCodes,
  downloadAdminInvitationCodes,
  listAdminInvitationCodes,
  updateAdminInvitationCode,
  type AdminInvitationCode,
} from "@/features/admin/api/invitations";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

function formatDate(value: string | null | undefined, locale: string): string {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";
  return date.toLocaleString(locale, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export function AdminInvitationsPage() {
  const t = useTranslations("adminInvitations");
  const locale = useLocale();
  const resolveErrorMessage = useLocalizedErrorMessage();

  const [items, setItems] = React.useState<AdminInvitationCode[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [pending, setPending] = React.useState(false);
  const [count, setCount] = React.useState("10");
  const [maxUses, setMaxUses] = React.useState("1");
  const [label, setLabel] = React.useState("");
  const [expiresAt, setExpiresAt] = React.useState("");
  const [generated, setGenerated] = React.useState<AdminInvitationCode[]>([]);

  const loadItems = React.useCallback(async () => {
    setLoading(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const data = await listAdminInvitationCodes(token);
      setItems(data.results ?? []);
    } catch (error) {
      toast.error(t("loadFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setLoading(false);
    }
  }, [resolveErrorMessage, t]);

  React.useEffect(() => {
    void loadItems();
  }, [loadItems]);

  const buildInput = React.useCallback(() => {
    const parsedCount = Number.parseInt(count, 10);
    const parsedMaxUses = Number.parseInt(maxUses, 10);
    const trimmedExpiry = expiresAt.trim();
    return {
      label: label.trim(),
      count: Number.isFinite(parsedCount) && parsedCount > 0 ? parsedCount : 1,
      maxUses: Number.isFinite(parsedMaxUses) && parsedMaxUses > 0 ? parsedMaxUses : 1,
      expiresAt: trimmedExpiry ? new Date(trimmedExpiry).toISOString() : null,
    };
  }, [count, expiresAt, label, maxUses]);

  const handleGenerate = React.useCallback(async () => {
    setPending(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const data = await createAdminInvitationCodes(token, buildInput());
      setGenerated(data.results ?? []);
      toast.success(t("generateSuccess", { count: data.results?.length ?? 0 }));
      await loadItems();
    } catch (error) {
      toast.error(t("generateFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setPending(false);
    }
  }, [buildInput, loadItems, resolveErrorMessage, t]);

  const handleDownload = React.useCallback(async () => {
    setPending(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const { filename, blob } = await downloadAdminInvitationCodes(token, buildInput());
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      toast.success(t("downloadSuccess"));
      await loadItems();
    } catch (error) {
      toast.error(t("downloadFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setPending(false);
    }
  }, [buildInput, loadItems, resolveErrorMessage, t]);

  const handleToggle = React.useCallback(
    async (item: AdminInvitationCode, enabled: boolean) => {
      try {
        const token = await resolveAccessToken();
        if (!token) return;
        await updateAdminInvitationCode(token, item.publicID, enabled);
        setItems((current) =>
          current.map((entry) => (entry.publicID === item.publicID ? { ...entry, enabled } : entry)),
        );
      } catch (error) {
        toast.error(t("updateFailed"), { description: resolveErrorMessage(error) });
      }
    },
    [resolveErrorMessage, t],
  );

  const copyGenerated = React.useCallback(async () => {
    const text = generated.map((item) => item.code).filter(Boolean).join("\n");
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      toast.success(t("copySuccess"));
    } catch {
      toast.error(t("copyFailed"));
    }
  }, [generated, t]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h1 className="text-lg font-semibold tracking-tight">{t("title")}</h1>
          <p className="text-sm text-muted-foreground">{t("description")}</p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void loadItems()} disabled={loading}>
          <RefreshCw className="size-4" aria-hidden="true" />
          {t("refresh")}
        </Button>
      </div>

      <section className="space-y-4 rounded-xl border bg-background p-4 shadow-sm">
        <h2 className="text-sm font-semibold">{t("generateTitle")}</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-2">
            <Label htmlFor="invitation-count">{t("countLabel")}</Label>
            <Input
              id="invitation-count"
              inputMode="numeric"
              value={count}
              onChange={(event) => setCount(event.target.value)}
              placeholder="10"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="invitation-max-uses">{t("maxUsesLabel")}</Label>
            <Input
              id="invitation-max-uses"
              inputMode="numeric"
              value={maxUses}
              onChange={(event) => setMaxUses(event.target.value)}
              placeholder="1"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="invitation-label">{t("labelLabel")}</Label>
            <Input
              id="invitation-label"
              value={label}
              onChange={(event) => setLabel(event.target.value)}
              placeholder={t("labelPlaceholder")}
              maxLength={80}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="invitation-expires">{t("expiresLabel")}</Label>
            <Input
              id="invitation-expires"
              type="datetime-local"
              value={expiresAt}
              onChange={(event) => setExpiresAt(event.target.value)}
            />
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button type="button" size="sm" onClick={() => void handleGenerate()} disabled={pending}>
            <Plus className="size-4" aria-hidden="true" />
            {t("generate")}
          </Button>
          <Button type="button" size="sm" variant="outline" onClick={() => void handleDownload()} disabled={pending}>
            <Download className="size-4" aria-hidden="true" />
            {t("generateAndDownload")}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">{t("plaintextNotice")}</p>
      </section>

      {generated.length > 0 ? (
        <section className="space-y-3 rounded-xl border bg-background p-4 shadow-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-sm font-semibold">{t("generatedTitle", { count: generated.length })}</h2>
            <Button type="button" size="sm" variant="outline" onClick={() => void copyGenerated()}>
              <Copy className="size-4" aria-hidden="true" />
              {t("copyAll")}
            </Button>
          </div>
          <pre className="max-h-64 overflow-auto rounded-md bg-muted/60 p-3 text-xs leading-6">
            {generated.map((item) => item.code).filter(Boolean).join("\n")}
          </pre>
        </section>
      ) : null}

      <section className="rounded-xl border bg-background shadow-sm">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Spinner />
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("columns.label")}</TableHead>
                <TableHead>{t("columns.usage")}</TableHead>
                <TableHead>{t("columns.expires")}</TableHead>
                <TableHead>{t("columns.lastUsed")}</TableHead>
                <TableHead className="text-right">{t("columns.enabled")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.length === 0 ? (
                <TableEmptyRow colSpan={5}>{t("empty")}</TableEmptyRow>
              ) : (
                items.map((item) => (
                  <TableRow key={item.publicID}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-medium">{item.label || t("untitled")}</span>
                        <span className="text-xs text-muted-foreground">{formatDate(item.createdAt, locale)}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={item.usedCount >= item.maxUses ? "secondary" : "outline"} className="rounded-md font-normal">
                        {item.usedCount} / {item.maxUses}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{formatDate(item.expiresAt, locale)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{formatDate(item.lastUsedAt, locale)}</TableCell>
                    <TableCell className="text-right">
                      <Switch
                        checked={item.enabled}
                        onCheckedChange={(checked) => void handleToggle(item, checked)}
                        aria-label={t("columns.enabled")}
                      />
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </section>
    </div>
  );
}
