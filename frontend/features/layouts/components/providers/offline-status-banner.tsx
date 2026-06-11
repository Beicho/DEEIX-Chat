"use client";

import * as React from "react";
import { WifiOff } from "lucide-react";
import { useTranslations } from "next-intl";

export function OfflineStatusBanner() {
  const t = useTranslations("common.network");
  const [offline, setOffline] = React.useState(false);

  React.useEffect(() => {
    const updateStatus = () => {
      setOffline(typeof navigator !== "undefined" ? !navigator.onLine : false);
    };

    updateStatus();
    window.addEventListener("online", updateStatus);
    window.addEventListener("offline", updateStatus);
    return () => {
      window.removeEventListener("online", updateStatus);
      window.removeEventListener("offline", updateStatus);
    };
  }, []);

  if (!offline) {
    return null;
  }

  return (
    <div
      role="status"
      aria-live="polite"
      className="fixed inset-x-3 top-[calc(0.75rem+env(safe-area-inset-top))] z-[90] mx-auto flex min-h-11 max-w-xl items-center gap-2 rounded-xl border border-border bg-popover px-4 py-2 text-sm font-medium text-popover-foreground shadow-xs"
    >
      <WifiOff className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.8} />
      <span className="min-w-0 flex-1">{t("offline")}</span>
    </div>
  );
}
