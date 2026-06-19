"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Bell, Send } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import {
  getAlertingConfig,
  updateAlertingConfig,
  testTelegramAlert,
  testWebhookAlert,
  type AlertingConfigView,
} from "@/features/admin/api/alerting";

function Row({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-2 border-b py-3 last:border-0 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="text-sm font-medium">{label}</p>
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

export function AdminAlertingPage() {
  const t = useTranslations("admin.alerting");
  const [config, setConfig] = React.useState<AlertingConfigView | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [enabled, setEnabled] = React.useState(false);
  const [telegramEnabled, setTelegramEnabled] = React.useState(false);
  const [webhookEnabled, setWebhookEnabled] = React.useState(false);
  const [botToken, setBotToken] = React.useState("");
  const [chatID, setChatID] = React.useState("");
  const [webhookURL, setWebhookURL] = React.useState("");
  const [debounce, setDebounce] = React.useState("300");

  const load = React.useCallback(async () => {
    try {
      setLoading(true);
      const data = await getAlertingConfig();
      setConfig(data);
      setEnabled(data.enabled);
      setTelegramEnabled(data.enabledNotifiers.includes("telegram"));
      setWebhookEnabled(data.enabledNotifiers.includes("webhook"));
      setChatID(data.telegramChatId || "");
      setDebounce(String(data.debounceSeconds || 300));
    } catch {
      toast.error(t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function handleSave() {
    setSaving(true);
    try {
      const notifiers: string[] = [];
      if (telegramEnabled) notifiers.push("telegram");
      if (webhookEnabled) notifiers.push("webhook");
      const payload: Parameters<typeof updateAlertingConfig>[0] = {
        enabled,
        enabledNotifiers: notifiers,
        telegramChatId: chatID.trim(),
        debounceSeconds: Number(debounce) || 300,
      };
      if (botToken.trim()) payload.telegramBotToken = botToken.trim();
      if (webhookURL.trim()) payload.webhookUrl = webhookURL.trim();
      const updated = await updateAlertingConfig(payload);
      setConfig(updated);
      setBotToken("");
      setWebhookURL("");
      toast.success(t("saved"));
    } catch {
      toast.error(t("saveFailed"));
    } finally {
      setSaving(false);
    }
  }

  async function handleTestTelegram() {
    try {
      await testTelegramAlert(
        botToken.trim() || chatID.trim()
          ? { telegramBotToken: botToken.trim(), telegramChatId: chatID.trim() }
          : undefined
      );
      toast.success(t("testSent"));
    } catch {
      toast.error(t("testFailed"));
    }
  }

  async function handleTestWebhook() {
    try {
      await testWebhookAlert(webhookURL.trim() ? { webhookUrl: webhookURL.trim() } : undefined);
      toast.success(t("testSent"));
    } catch {
      toast.error(t("testFailed"));
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
      </div>

      <Card className="p-4">
        <h2 className="mb-2 text-sm font-semibold">{t("generalSection")}</h2>
        <Row label={t("enabledLabel")} hint={t("enabledHelp")}>
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
            className="size-4 rounded border-border"
          />
        </Row>
        <Row label={t("debounceLabel")} hint={t("debounceHelp")}>
          <Input
            type="number"
            min="30"
            max="3600"
            value={debounce}
            onChange={(e) => setDebounce(e.target.value)}
            className="w-[160px]"
          />
        </Row>
      </Card>

      <Card className="p-4">
        <h2 className="mb-2 text-sm font-semibold">{t("telegramSection")}</h2>
        <Row
          label={t("telegramEnabledLabel")}
          hint={config?.telegramConfigured ? t("configured") : t("notConfigured")}
        >
          <input
            type="checkbox"
            checked={telegramEnabled}
            onChange={(e) => setTelegramEnabled(e.target.checked)}
            className="size-4 rounded border-border"
          />
        </Row>
        <Row label={t("botTokenLabel")} hint={t("botTokenHelp")}>
          <Input
            type="password"
            value={botToken}
            onChange={(e) => setBotToken(e.target.value)}
            placeholder={config?.telegramConfigured ? "••••••••" : ""}
            className="w-[260px] max-w-full"
          />
        </Row>
        <Row label={t("chatIdLabel")}>
          <Input value={chatID} onChange={(e) => setChatID(e.target.value)} className="w-[260px] max-w-full" />
        </Row>
        <div className="pt-3">
          <Button type="button" variant="outline" size="sm" onClick={handleTestTelegram} className="gap-2">
            <Send className="size-3.5" />
            {t("testTelegram")}
          </Button>
        </div>
      </Card>

      <Card className="p-4">
        <h2 className="mb-2 text-sm font-semibold">{t("webhookSection")}</h2>
        <Row
          label={t("webhookEnabledLabel")}
          hint={config?.webhookConfigured ? t("configured") : t("notConfigured")}
        >
          <input
            type="checkbox"
            checked={webhookEnabled}
            onChange={(e) => setWebhookEnabled(e.target.checked)}
            className="size-4 rounded border-border"
          />
        </Row>
        <Row label={t("webhookUrlLabel")} hint={t("webhookUrlHelp")}>
          <Input
            value={webhookURL}
            onChange={(e) => setWebhookURL(e.target.value)}
            placeholder={config?.webhookConfigured ? "••••••••" : "https://..."}
            className="w-[260px] max-w-full"
          />
        </Row>
        <div className="pt-3">
          <Button type="button" variant="outline" size="sm" onClick={handleTestWebhook} className="gap-2">
            <Send className="size-3.5" />
            {t("testWebhook")}
          </Button>
        </div>
      </Card>

      <div className="flex items-center gap-2">
        <Button onClick={handleSave} disabled={saving} className="gap-2">
          {saving ? <Spinner className="size-4" /> : <Bell className="size-4" />}
          {saving ? t("saving") : t("save")}
        </Button>
      </div>
    </div>
  );
}
