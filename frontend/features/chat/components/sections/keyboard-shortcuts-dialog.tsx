"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export function KeyboardShortcutsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const t = useTranslations("common");
  const shortcuts = [
    { keys: ["Enter"], label: t("shortcutSend") },
    { keys: ["Shift", "Enter"], label: t("shortcutNewline") },
    { keys: ["Ctrl", "K"], label: t("shortcutSearch") },
    { keys: ["Esc"], label: t("shortcutStop") },
    { keys: ["Ctrl", "/"], label: t("shortcutHelp") },
  ];
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("shortcuts")}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          {shortcuts.map((s) => (
            <div key={s.label} className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">{s.label}</span>
              <div className="flex items-center gap-1">
                {s.keys.map((k) => (
                  <kbd key={k} className="rounded border border-border bg-muted px-2 py-0.5 text-xs font-medium text-foreground">
                    {k}
                  </kbd>
                ))}
              </div>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
