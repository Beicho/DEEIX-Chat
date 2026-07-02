"use client";

import { useTranslations } from "next-intl";

import { PanelRight } from "@/components/animate-ui/icons/panel-right";
import { Button } from "@/components/ui/button";
import { PlusIcon } from "@/components/ui/plus";
import { useSidebar } from "@/components/ui/sidebar";
import { NotificationCenterPopover } from "@/features/notifications/components/notification-center-popover";
import { AppLogo } from "@/shared/components/app-logo";

export function MobileHeader({ onCreateConversation }: { onCreateConversation: () => void }) {
  const t = useTranslations("common.navigation");
  const { isMobile, toggleSidebar } = useSidebar();

  return (
    <header className="grid min-h-12 shrink-0 grid-cols-[2.75rem_minmax(0,1fr)_auto] items-center px-3 pt-[env(safe-area-inset-top)] md:hidden">
      <div className="flex justify-start">
        <Button variant="ghost" size="icon" className="relative size-9 after:absolute after:-inset-1 after:content-['']" onClick={toggleSidebar}>
          <PanelRight size={18} strokeWidth={1.4} />
          <span className="sr-only">{t("openSidebar")}</span>
        </Button>
      </div>

      <div className="flex min-w-0 justify-center">
        <AppLogo
          width={64}
          height={48}
          priority
          className="h-5 w-auto object-contain"
        />
      </div>

<div className="flex justify-end gap-1">
        {isMobile ? <NotificationCenterPopover variant="icon" /> : null}
        <Button variant="ghost" size="icon" className="relative size-9 after:absolute after:-inset-1 after:content-['']" onClick={onCreateConversation}>
          <PlusIcon size={16} strokeWidth={1.6} />
          <span className="sr-only">{t("newChat")}</span>
        </Button>
      </div>
    </header>
  );
}
