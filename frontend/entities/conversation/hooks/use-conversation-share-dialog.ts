"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import {
  createConversationShare,
  getConversationShare,
  regenerateConversationShare,
  revokeConversationShare,
} from "@/shared/api/conversation";
import type { ConversationShareDTO, CreateConversationShareRequest } from "@/shared/api/conversation.types";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import {
  buildConversationNativeShareData,
  isNativeShareAbortError,
} from "@/features/chat/model/conversation-share-utils";
import { resolveShareCanonicalPath } from "@/features/share/model/share-metadata";

function resolveShareURL(shareID: string): string {
  const path = `/share?conversation_id=${encodeURIComponent(shareID)}`;
  if (typeof window === "undefined") {
    return path;
  }
  return `${window.location.origin}${path}`;
}

function isActiveShare(share: ConversationShareDTO | null): share is ConversationShareDTO {
  return Boolean(share?.status === "active" && share.shareID.trim());
}

function normalizeShareScope(value: string): ShareScopeValue {
  return value === "full" ? "full" : "current";
}

function inferShareExpiryValue(expiresAt: string | null): ShareExpiryValue {
  if (!expiresAt) {
    return "0";
  }
  const expires = new Date(expiresAt).getTime();
  if (!Number.isFinite(expires)) {
    return "7";
  }
  const days = Math.ceil((expires - Date.now()) / 86400000);
  return days > 14 ? "30" : "7";
}

function formatShareExpiresAt(value: string | null, locale: string): string {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function sharePatchFromDTO(share: ConversationShareDTO) {
  const active = isActiveShare(share);
  return {
    shareStatus: share.status,
    shareID: active ? share.shareID : "",
    sharedAt: active ? share.createdAt : null,
    lastShareAccessedAt: share.lastAccessedAt,
  };
}

export function useConversationShareDialog({
  conversationPublicID,
  conversationTitle,
  defaultMessagePublicIDs,
  onExportImage,
  onShareChange,
  open,
}: {
  conversationPublicID: string;
  conversationTitle: string;
  defaultMessagePublicIDs?: string[];
  onShareChange?: (share: ConversationShareDTO) => void;
  open: boolean;
}) {
  const tCommon = useTranslations("common.actions");
  const tConversation = useTranslations("conversation");
  const t = useTranslations("conversation.shareDialog");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [share, setShare] = React.useState<ConversationShareDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [working, setWorking] = React.useState<"create" | "revoke" | "regenerate" | null>(null);
  const [nativeShareSupported, setNativeShareSupported] = React.useState(false);
  const [nativeShareWorking, setNativeShareWorking] = React.useState(false);
  const [shareScope, setShareScope] = React.useState<ShareScopeValue>("current");
  const [expiresInDays, setExpiresInDays] = React.useState<ShareExpiryValue>("7");
  const [passwordProtected, setPasswordProtected] = React.useState(false);
  const [sharePassword, setSharePassword] = React.useState("");
  const [includeThinking, setIncludeThinking] = React.useState(false);
  const active = isActiveShare(share);
  const currentURL = active ? resolveShareURL(share.shareID) : "";
  const snapshotMessageCount = active ? share.messageCount : (defaultMessagePublicIDs?.length ?? 0);
  const normalizedTitle = conversationTitle.trim() || tConversation("untitled");
  const headerDescription = snapshotMessageCount > 0
    ? t("snapshotMessages", { title: normalizedTitle, count: snapshotMessageCount })
    : normalizedTitle;
  const hasDefaultBranch = defaultMessagePublicIDs === undefined || defaultMessagePublicIDs.length > 0;
  const onShareChangeRef = React.useRef(onShareChange);
  const activeExpiryText = active ? formatShareExpiresAt(share.expiresAt, locale) : "";

  React.useEffect(() => {
    onShareChangeRef.current = onShareChange;
  }, [onShareChange]);

  React.useEffect(() => {
    setNativeShareSupported(typeof navigator !== "undefined" && typeof navigator.share === "function");
  }, []);

  const applyShare = React.useCallback((next: ConversationShareDTO) => {
    setShare(next);
    if (next.status === "active") {
      setShareScope(normalizeShareScope(next.scope));
      setExpiresInDays(inferShareExpiryValue(next.expiresAt));
      setPasswordProtected(Boolean(next.hasPassword));
      setSharePassword("");
      setIncludeThinking(Boolean(next.includeThinking));
    } else {
      setShareScope("current");
      setExpiresInDays("7");
      setPasswordProtected(false);
      setSharePassword("");
      setIncludeThinking(false);
    }
    onShareChangeRef.current?.(next);
  }, []);

  const buildSharePayload = React.useCallback((): CreateConversationShareRequest | null => {
    const password = sharePassword.trim();
    if (passwordProtected && !password) {
      toast.error(t("passwordRequiredToProtect"));
      return null;
    }
    return {
      defaultMessagePublicIDs,
      scope: shareScope,
      expiresInDays: Number(expiresInDays) as 0 | 7 | 30,
      password: passwordProtected ? password : "",
      includeThinking,
    };
  }, [defaultMessagePublicIDs, expiresInDays, includeThinking, passwordProtected, sharePassword, shareScope, t]);

  React.useEffect(() => {
    if (!open || !conversationPublicID.trim()) {
      return;
    }

    let cancelled = false;
    async function loadShare() {
      setLoading(true);
      try {
        const token = await resolveAccessToken();
        if (!token || cancelled) {
          return;
        }
        const data = await getConversationShare(token, conversationPublicID);
        if (!cancelled) {
          applyShare(data);
        }
      } catch (error) {
        if (!cancelled) {
          toast.error(t("loadFailed"), {
            description: resolveErrorMessage(error, tCommon("retry")),
          });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadShare();
    return () => {
      cancelled = true;
    };
  }, [applyShare, conversationPublicID, open, resolveErrorMessage, t, tCommon]);

  const runMutation = React.useCallback(
    async (mode: "create" | "revoke" | "regenerate") => {
      if (!conversationPublicID.trim() || working) {
        return;
      }
      if ((mode === "create" || mode === "regenerate") && !hasDefaultBranch) {
        toast.error(t("noMessages"));
        return;
      }

      const token = await resolveAccessToken();
      if (!token) {
        toast.error(t("signInRequired"));
        return;
      }

      setWorking(mode);
      try {
        let payload: CreateConversationShareRequest | null = null;
        if (mode === "create" || mode === "regenerate") {
          payload = buildSharePayload();
          if (!payload) {
            return;
          }
        }
        const next =
          mode === "create"
            ? await createConversationShare(token, conversationPublicID, payload ?? {})
            : mode === "regenerate"
              ? await regenerateConversationShare(token, conversationPublicID, payload ?? {})
              : await revokeConversationShare(token, conversationPublicID);
        applyShare(next);
        toast.success(
          mode === "revoke"
            ? t("closed")
            : mode === "regenerate"
              ? t("regenerated")
              : t("created"),
        );
      } catch (error) {
        toast.error(t("operationFailed"), {
          description: resolveErrorMessage(error, tCommon("retry")),
        });
      } finally {
        setWorking(null);
      }
    },
    [applyShare, buildSharePayload, conversationPublicID, hasDefaultBranch, resolveErrorMessage, t, tCommon, working],
  );

  return {
    active,
    currentURL,
    hasDefaultBranch,
    headerDescription,
    loading,
    runMutation,
    working,
  };
}
