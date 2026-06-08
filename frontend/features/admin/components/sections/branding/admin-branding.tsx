"use client";

import * as React from "react";
import { Save } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { listAdminSettingsByNamespace, patchAdminSettings } from "@/features/admin/api";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import type { PatchSettingItem, SettingItem } from "@/shared/api/settings.types";
import { SettingsFieldItem, SettingsFieldList, SettingsPage, SettingsSection } from "@/shared/components/settings-layout";

type BrandingDraft = {
  appName: string;
  logoURL: string;
  logoDarkURL: string;
};

const DEFAULT_DRAFT: BrandingDraft = {
  appName: "DEEIX Chat",
  logoURL: "/logo.svg",
  logoDarkURL: "/logo-white.svg",
};

function toDraft(items: SettingItem[]): BrandingDraft {
  const map = new Map(items.map((item) => [item.key, item.value ?? ""]));
  return {
    appName: map.get("app_name")?.trim() || DEFAULT_DRAFT.appName,
    logoURL: map.get("logo_url")?.trim() || DEFAULT_DRAFT.logoURL,
    logoDarkURL: map.get("logo_dark_url")?.trim() || DEFAULT_DRAFT.logoDarkURL,
  };
}

function changedItems(draft: BrandingDraft, saved: BrandingDraft): PatchSettingItem[] {
  const rows: Array<[keyof BrandingDraft, string]> = [
    ["appName", "app_name"],
    ["logoURL", "logo_url"],
    ["logoDarkURL", "logo_dark_url"],
  ];
  return rows
    .filter(([field]) => draft[field].trim() !== saved[field].trim())
    .map(([field, key]) => ({ namespace: "branding", key, value: draft[field].trim() }));
}

function resolvePreviewSource(src: string): string {
  const trimmed = src.trim();
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) return trimmed;
  if (trimmed.startsWith("/") && !trimmed.startsWith("//")) return trimmed;
  return "/logo.svg";
}

function LogoPreview({ title, src }: { title: string; src: string }) {
  const previewSrc = resolvePreviewSource(src);
  return (
    <div className="rounded-xl border bg-background p-4">
      <div className="mb-3 text-xs font-medium text-muted-foreground">{title}</div>
      <div className="flex h-20 items-center justify-center rounded-lg bg-muted/30 px-4">
        {/* eslint-disable-next-line @next/next/no-img-element -- Branding previews allow arbitrary admin-configured URLs. */}
        <img
          src={previewSrc}
          alt={title}
          className="max-h-12 w-auto object-contain"
        />
      </div>
    </div>
  );
}

export function AdminBrandingSettingsPage() {
  const t = useTranslations("adminUsers.brandingPage");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [draft, setDraft] = React.useState<BrandingDraft>(DEFAULT_DRAFT);
  const [saved, setSaved] = React.useState<BrandingDraft>(DEFAULT_DRAFT);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);

  const loadSettings = React.useCallback(async () => {
    setLoading(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const next = toDraft(await listAdminSettingsByNamespace(token, "branding"));
      setDraft(next);
      setSaved(next);
    } catch (error) {
      toast.error(t("loadFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setLoading(false);
    }
  }, [resolveErrorMessage, t]);

  React.useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  const dirtyItems = React.useMemo(() => changedItems(draft, saved), [draft, saved]);

  const save = React.useCallback(async () => {
    if (dirtyItems.length === 0) return;
    setSaving(true);
    try {
      const token = await resolveAccessToken();
      if (!token) return;
      const grouped = await patchAdminSettings(token, { items: dirtyItems });
      const next = toDraft(grouped.branding ?? []);
      setDraft(next);
      setSaved(next);
      window.dispatchEvent(new Event("deeix:branding-updated"));
      toast.success(t("saved"));
    } catch (error) {
      toast.error(t("saveFailed"), { description: resolveErrorMessage(error) });
    } finally {
      setSaving(false);
    }
  }, [dirtyItems, resolveErrorMessage, t]);

  return (
    <SettingsPage className="space-y-6">
      <div className="flex flex-col gap-3 px-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">{t("title")}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
        </div>
        <Button type="button" size="sm" disabled={loading || saving || dirtyItems.length === 0} onClick={() => void save()}>
          {saving ? <Spinner className="size-3.5" /> : <Save className="size-3.5" />}
          {saving ? t("saving") : t("save")}
        </Button>
      </div>

      <SettingsSection title={t("formTitle")}>
        <SettingsFieldList>
          <SettingsFieldItem index={0}>
            <div className="grid gap-2">
              <label className="text-sm font-medium" htmlFor="branding-app-name">{t("appName")}</label>
              <Input id="branding-app-name" value={draft.appName} disabled={loading || saving} onChange={(event) => setDraft((current) => ({ ...current, appName: event.target.value }))} />
            </div>
          </SettingsFieldItem>
          <SettingsFieldItem index={1}>
            <div className="grid gap-2">
              <label className="text-sm font-medium" htmlFor="branding-logo-url">{t("logoURL")}</label>
              <Input id="branding-logo-url" value={draft.logoURL} disabled={loading || saving} placeholder="/logo.svg" onChange={(event) => setDraft((current) => ({ ...current, logoURL: event.target.value }))} />
            </div>
          </SettingsFieldItem>
          <SettingsFieldItem index={2}>
            <div className="grid gap-2">
              <label className="text-sm font-medium" htmlFor="branding-logo-dark-url">{t("logoDarkURL")}</label>
              <Input id="branding-logo-dark-url" value={draft.logoDarkURL} disabled={loading || saving} placeholder="/logo-white.svg" onChange={(event) => setDraft((current) => ({ ...current, logoDarkURL: event.target.value }))} />
            </div>
          </SettingsFieldItem>
        </SettingsFieldList>
      </SettingsSection>

      <div className="grid gap-3 md:grid-cols-2">
        <LogoPreview title={t("lightPreview")} src={draft.logoURL} />
        <LogoPreview title={t("darkPreview")} src={draft.logoDarkURL || draft.logoURL} />
      </div>
    </SettingsPage>
  );
}
