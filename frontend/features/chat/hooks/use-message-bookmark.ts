"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import type { ChatAreaMessage } from "@/features/chat/types/messages";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import { deleteMessageBookmark, setMessageBookmark } from "@/shared/api/conversation";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";

export function useMessageBookmark(messages: ChatAreaMessage[]) {
  const t = useTranslations("chat.bookmarks");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [overrides, setOverrides] = React.useState<Record<string, boolean>>({});
  const activeIDs = React.useMemo(() => new Set(messages.map((item) => item.publicID)), [messages]);
  const activeIDsRef = React.useRef(activeIDs);
  const activeIDsKey = React.useMemo(() => Array.from(activeIDs).join("|"), [activeIDs]);

  React.useEffect(() => {
    activeIDsRef.current = activeIDs;
  }, [activeIDs]);

  React.useEffect(() => {
    setOverrides((prev) => {
      const nextEntries = Object.entries(prev).filter(([publicID]) => activeIDsRef.current.has(publicID));
      if (nextEntries.length === Object.keys(prev).length) {
        return prev;
      }
      return Object.fromEntries(nextEntries);
    });
  }, [activeIDsKey]);

  const getBookmarked = React.useCallback(
    (item: ChatAreaMessage) => {
      if (Object.prototype.hasOwnProperty.call(overrides, item.publicID)) {
        return Boolean(overrides[item.publicID]);
      }
      return Boolean(item.bookmarked);
    },
    [overrides],
  );

  const onToggleMessageBookmark = React.useCallback(
    async (publicID: string) => {
      const target = messages.find((item) => item.publicID === publicID);
      if (!target) {
        return;
      }

      const previous = getBookmarked(target);
      const next = !previous;
      setOverrides((prev) => ({
        ...prev,
        [publicID]: next,
      }));

      try {
        const token = await resolveAccessToken();
        if (!token) {
          throw new Error(t("signInRequired"));
        }

        const result = next
          ? await setMessageBookmark(token, publicID, { bookmarked: true })
          : await deleteMessageBookmark(token, publicID);
        setOverrides((prev) => ({
          ...prev,
          [publicID]: Boolean(result.bookmarked),
        }));
        toast.success(result.bookmarked ? t("saved") : t("removed"));
      } catch (error) {
        setOverrides((prev) => ({
          ...prev,
          [publicID]: previous,
        }));
        const description = resolveErrorMessage(error, t("retryLater"));
        toast.error(t("failed"), { description });
      }
    },
    [getBookmarked, messages, resolveErrorMessage, t],
  );

  return {
    getBookmarked,
    onToggleMessageBookmark,
  };
}
