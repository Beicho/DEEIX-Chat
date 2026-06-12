"use client";

import * as React from "react";
import { Download, Upload } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { SpinnerLabel } from "@/components/ui/spinner";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { exportConversationTakeout, importConversationTakeout } from "@/shared/api/conversation";
import type { ConversationTakeoutDTO } from "@/shared/api/conversation.types";
import { SettingsFieldRow, SettingsSection } from "@/shared/components/settings-layout";

function downloadJSON(data: ConversationTakeoutDTO): void {
  const stamp = new Date().toISOString().slice(0, 10);
  const blob = new Blob([`${JSON.stringify(data, null, 2)}\n`], {
    type: "application/json;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `deeix-chat-takeout-${stamp}.json`;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function safeCount(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

export function SettingsData() {
  const t = useTranslations("settings.generalPage.data");
  const { accessToken } = useAuthSession();
  const fileInputRef = React.useRef<HTMLInputElement | null>(null);
  const [exporting, setExporting] = React.useState(false);
  const [importing, setImporting] = React.useState(false);

  const handleExport = React.useCallback(async () => {
    if (!accessToken || exporting) {
      return;
    }
    try {
      setExporting(true);
      const data = await exportConversationTakeout(accessToken);
      downloadJSON(data);
      toast.success(t("toast.exported"));
    } catch {
      toast.error(t("toast.exportFailed"));
    } finally {
      setExporting(false);
    }
  }, [accessToken, exporting, t]);

  const handleImportFile = React.useCallback(async (file: File) => {
    if (!accessToken || importing) {
      return;
    }
    try {
      setImporting(true);
      const text = await file.text();
      const payload = JSON.parse(text) as unknown;
      const result = await importConversationTakeout(accessToken, payload);
      toast.success(t("toast.imported", {
        conversations: safeCount(result.importedConversationCount),
        messages: safeCount(result.importedMessageCount),
      }));
    } catch {
      toast.error(t("toast.importFailed"), {
        description: t("toast.importFailedDescription"),
      });
    } finally {
      setImporting(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  }, [accessToken, importing, t]);

  return (
    <SettingsSection title={t("title")}>
      <SettingsFieldRow
        title={t("takeoutTitle")}
        description={t("takeoutDescription")}
        controlClassName="sm:w-auto md:w-auto"
      >
        <div className="flex w-full flex-wrap justify-start gap-2 md:justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="relative min-h-9 after:absolute after:-inset-1.5 after:content-['']"
            disabled={!accessToken || exporting || importing}
            onClick={() => void handleExport()}
          >
            {exporting ? (
              <SpinnerLabel>{t("exporting")}</SpinnerLabel>
            ) : (
              <>
                <Download className="size-4" aria-hidden="true" />
                {t("export")}
              </>
            )}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="relative min-h-9 after:absolute after:-inset-1.5 after:content-['']"
            disabled={!accessToken || exporting || importing}
            onClick={() => fileInputRef.current?.click()}
          >
            {importing ? (
              <SpinnerLabel>{t("importing")}</SpinnerLabel>
            ) : (
              <>
                <Upload className="size-4" aria-hidden="true" />
                {t("import")}
              </>
            )}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept="application/json,.json"
            className="sr-only"
            aria-label={t("fileInput")}
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                void handleImportFile(file);
              }
            }}
          />
        </div>
      </SettingsFieldRow>
    </SettingsSection>
  );
}
