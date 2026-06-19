"use client";

import * as React from "react";
import { WifiOff } from "lucide-react";
import { useTranslations } from "next-intl";

export function OfflineStatusBanner() {
  const t = useTranslations("common.network");
  const [offline, setOffline] = React.useState(false);

  React.useEffect(() => {
    let cancelled = false;

    // 仅凭 navigator.onLine 会误报：部分环境初始即返回 false 且此后不再触发 online 事件，
    // 横幅会一直卡在顶部。这里在显示前用一次同源探测确认确实断网，网络恢复后清除。
    const probe = async () => {
      try {
        await fetch(`/manifest.webmanifest?_=${Date.now()}`, { method: "HEAD", cache: "no-store" });
        if (!cancelled) setOffline(false);
      } catch {
        if (!cancelled) setOffline(typeof navigator !== "undefined" ? !navigator.onLine : false);
      }
    };

    const handleOffline = () => {
      void probe();
    };
    const handleOnline = () => {
      if (!cancelled) setOffline(false);
    };

    if (typeof navigator !== "undefined" && !navigator.onLine) {
      void probe();
    }
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);
    return () => {
      cancelled = true;
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
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
