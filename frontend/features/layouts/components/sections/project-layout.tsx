"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";

import { AnnouncementDialogHost } from "@/features/announcements/components/announcement-dialog-host";
import { AppearancePreferencesSync } from "@/features/settings/components/appearance-preferences-sync";
import { AppSidebar } from "@/features/layouts/components/navigation/app-sidebar";
import { InitialSecurityGuard } from "@/features/layouts/components/sections/initial-security-guard";
import { MobileHeader } from "@/features/layouts/components/sections/mobile-header";
import { SidebarRouteCloser } from "@/features/layouts/components/sections/sidebar-route-closer";
import { ChatSessionProvider, useChatSession } from "@/features/chat/context/chat-session-context";
import { SidebarRecentsProvider } from "@/features/recent/context/sidebar-recents-context";
import { UserLocaleSync } from "@/i18n/user-locale-sync";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";

function ProjectLayoutContent({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { requestNewConversation } = useChatSession();

  const onCreateConversation = React.useCallback(() => {
    requestNewConversation({ projectID: "" });
    if (pathname === "/chat") {
      window.history.pushState(null, "", "/chat");
      return;
    }
    router.push("/chat");
  }, [pathname, requestNewConversation, router]);

  return (
    <>
      <UserLocaleSync />
      <AppearancePreferencesSync />
      <InitialSecurityGuard />
      <AnnouncementDialogHost />
      <SidebarRouteCloser />
      <AppSidebar onCreateConversation={onCreateConversation} />
      <SidebarInset>
        <MobileHeader onCreateConversation={onCreateConversation} />
        <main
          id="main-content"
          tabIndex={-1}
          className="flex h-full min-h-0 flex-1 flex-col gap-4 overflow-hidden px-0 pb-[calc(0.5rem+env(safe-area-inset-bottom))] pt-0 outline-none md:p-4 md:pt-0"
        >
          {children}
        </main>
      </SidebarInset>
    </>
  );
}

export function ProjectLayout({
  children,
  defaultSidebarOpen = true,
}: {
  children: React.ReactNode;
  defaultSidebarOpen?: boolean;
}) {
  const t = useTranslations("common.navigation");

  return (
    <SidebarProvider className="h-svh overflow-hidden" defaultOpen={defaultSidebarOpen}>
      <SidebarRecentsProvider>
        <ChatSessionProvider>
          <a
            href="#main-content"
            className="sr-only fixed left-3 top-3 z-50 rounded-md bg-background px-3 py-2 text-sm font-medium text-foreground shadow-md focus:not-sr-only focus:outline-none focus:ring-[3px] focus:ring-ring/50"
          >
            {t("skipToContent")}
          </a>
          <ProjectLayoutContent>{children}</ProjectLayoutContent>
        </ChatSessionProvider>
      </SidebarRecentsProvider>
    </SidebarProvider>
  );
}
