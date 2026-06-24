import { generateLongImage, downloadDataUrl, type LongImageMessage } from "@/features/chat/model/generate-long-image";
"use client";
import * as React from "react";

import * as React from "react";
import { Copy, ExternalLink, ImageDown, Share2 } from "lucide-react";
import { ExternalLink } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SpinnerLabel } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import {
  createConversationShare,
  getConversationShare,
  regenerateConversationShare,
  revokeConversationShare,
} from "@/shared/api/conversation";
import type { ConversationShareDTO, CreateConversationShareRequest } from "@/shared/api/conversation.types";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { useAppLocale } from "@/i18n/app-i18n-provider";
import { CopyActionButton } from "@/shared/components/copy-action";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";
import {
  buildConversationNativeShareData,
  isNativeShareAbortError,
} from "@/features/chat/model/conversation-share-utils";
import { resolveShareCanonicalPath } from "@/features/share/model/share-metadata";

type ConversationShareDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conversationPublicID: string;
  conversationTitle: string;
  defaultMessagePublicIDs?: string[];
  onExportImage?: () => void | Promise<void>;
  onShareChange?: (share: ConversationShareDTO) => void;
};

type ShareScopeValue = "current" | "full";
type ShareExpiryValue = "7" | "30" | "0";

function shareURL(shareID: string): string {
  const path = resolveShareCanonicalPath(shareID);
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

export function ConversationShareDialog({
  open,
  onOpenChange,
  conversationPublicID,
  conversationTitle,
  defaultMessagePublicIDs,
  onExportImage,
  onShareChange,
}: ConversationShareDialogProps) {
  const tCommon = useTranslations("common.actions");
  const tChat = useTranslations("chat");
  const t = useTranslations("chat.shareDialog");
  const { locale } = useAppLocale();
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
  const currentURL = active ? shareURL(share.shareID) : "";
  const snapshotMessageCount = active ? share.messageCount : (defaultMessagePublicIDs?.length ?? 0);
  const normalizedTitle = conversationTitle.trim() || tChat("untitledConversation");
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
  
  async function handleDownloadImage() {
    try {
      setGeneratingImage(true);
      const messages: LongImageMessage[] = [];
      const dataUrl = await generateLongImage(messages);
      downloadDataUrl(dataUrl, "deeix-chat-" + Date.now() + ".png");
      toast.success(t("share.imageGenerated"));
    } catch {
      toast.error(t("share.generating"));
    } finally {
      setGeneratingImage(false);
    }
  }

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

  const openLink = React.useCallback(() => {
    if (!currentURL) {
      return;
    }
    window.open(currentURL, "_blank", "noopener,noreferrer");
  }, [currentURL]);

  const nativeShare = React.useCallback(async () => {
    if (!currentURL || typeof navigator === "undefined" || typeof navigator.share !== "function") {
      return;
    }
    setNativeShareWorking(true);
    try {
      await navigator.share(
        buildConversationNativeShareData({
          title: normalizedTitle,
          text: headerDescription,
          url: currentURL,
        }),
      );
    } catch (error) {
      if (!isNativeShareAbortError(error)) {
        toast.error(t("nativeShareFailed"));
      }
    } finally {
      setNativeShareWorking(false);
    }
  }, [currentURL, headerDescription, normalizedTitle, t]);


  async function handleDownloadImage() {
    try {
      setGeneratingImage(true);
      const messages: LongImageMessage[] = [];
      const dataUrl = await generateLongImage(messages);
      downloadDataUrl(dataUrl, "deeix-chat-" + Date.now() + ".png");
      toast.success(t("share.imageGenerated"));
    } catch {
      toast.error(t("share.generating"));
    } finally {
      setGeneratingImage(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[540px]">
        <div className="flex items-start justify-between gap-4">
          <DialogHeader className="min-w-0 flex-1">
            <DialogTitle>{t("title")}</DialogTitle>
            <DialogDescription>{headerDescription}</DialogDescription>
          </DialogHeader>
          <Badge variant="secondary">{active ? t("statusShared") : t("statusNotShared")}</Badge>
        </div>

        <div className="space-y-4">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t("publicLink")}</p>
            <div className="flex items-center gap-2">
              <Input
                readOnly
                value={currentURL || t("emptyLink")}
                className={!currentURL ? "text-muted-foreground" : undefined}
              />
              <CopyActionButton
                type="button"
                variant="ghost"
                size="icon"
                disabled={!active}
                value={currentURL}
                messages={{ copied: t("linkCopied"), failed: t("copyFailed") }}
                iconClassName="size-4"
                aria-label={t("copyLink")}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                disabled={!active}
                onClick={openLink}
                aria-label={t("openLink")}
              >
                <ExternalLink className="size-4" />
              </Button>
            </div>
          </div>
          <div className="space-y-3 rounded-lg border border-border bg-card/60 p-3">
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1">
                <Label>{t("scopeLabel")}</Label>
                <Select value={shareScope} onValueChange={(value) => setShareScope(normalizeShareScope(value))}>
                  <SelectTrigger size="sm" className="h-9">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="current">{t("scopeCurrent")}</SelectItem>
                    <SelectItem value="full">{t("scopeFull")}</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs leading-5 text-muted-foreground">{t("scopeDescription")}</p>
              </div>
              <div className="space-y-1">
                <Label>{t("expiresLabel")}</Label>
                <Select value={expiresInDays} onValueChange={(value) => setExpiresInDays(value as ShareExpiryValue)}>
                  <SelectTrigger size="sm" className="h-9">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">{t("expires7")}</SelectItem>
                    <SelectItem value="30">{t("expires30")}</SelectItem>
                    <SelectItem value="0">{t("expiresNever")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="flex items-start justify-between gap-3 rounded-md border border-border/60 p-3">
              <div className="min-w-0 space-y-1">
                <Label htmlFor="share-password-toggle" className="mb-0 text-foreground">
                  {t("passwordLabel")}
                </Label>
                <p className="text-xs leading-5 text-muted-foreground">{t("passwordHelp")}</p>
              </div>
              <Switch
                id="share-password-toggle"
                checked={passwordProtected}
                onCheckedChange={(checked) => {
                  setPasswordProtected(checked);
                  if (!checked) {
                    setSharePassword("");
                  }
                }}
                aria-label={t("passwordLabel")}
              />
            </div>
            {passwordProtected ? (
              <Input
                type="password"
                value={sharePassword}
                onChange={(event) => setSharePassword(event.target.value)}
                placeholder={t("passwordPlaceholder")}
                autoComplete="off"
                className="h-9"
              />
            ) : null}

            <div className="flex items-start justify-between gap-3 rounded-md border border-border/60 p-3">
              <div className="min-w-0 space-y-1">
                <Label htmlFor="share-thinking-toggle" className="mb-0 text-foreground">
                  {t("includeThinking")}
                </Label>
                <p className="text-xs leading-5 text-muted-foreground">{t("includeThinkingHelp")}</p>
              </div>
              <Switch
                id="share-thinking-toggle"
                checked={includeThinking}
                onCheckedChange={setIncludeThinking}
                aria-label={t("includeThinking")}
              />
            </div>
            {active ? (
              <div className="flex flex-wrap gap-1.5 pt-1">
                <Badge variant="outline">
                  {normalizeShareScope(share.scope) === "full" ? t("scopeFull") : t("scopeCurrent")}
                </Badge>
                {share.hasPassword ? <Badge variant="outline">{t("protected")}</Badge> : null}
                {share.includeThinking ? <Badge variant="outline">{t("thinkingIncluded")}</Badge> : null}
                {activeExpiryText ? (
                  <Badge variant="outline">{t("expiresAt", { time: activeExpiryText })}</Badge>
                ) : (
                  <Badge variant="outline">{t("expiresNever")}</Badge>
                )}
              </div>
            ) : null}
          </div>
        </div>

        <DialogFooter className="flex-wrap">
          {onExportImage ? (
            <Button
              type="button"
              variant="ghost"
              onClick={() => void onExportImage()}
              disabled={Boolean(working) || loading}
            >
              <ImageDown className="size-4" />
              {t("exportImage")}
            </Button>
          ) : null}
          {active ? (
            <>
              {nativeShareSupported ? (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => void nativeShare()}
                  disabled={Boolean(working) || loading || nativeShareWorking || !currentURL}
                >
                  {nativeShareWorking ? <SpinnerLabel>{t("nativeSharing")}</SpinnerLabel> : (
                    <>
                      <Share2 className="size-4" />
                      {t("nativeShare")}
                    </>
                  )}
                </Button>
              ) : null}
              <Button
                type="button"
                variant="ghost"
                onClick={() => void runMutation("revoke")}
                disabled={Boolean(working) || loading}
              >
                {working === "revoke" ? <SpinnerLabel>{t("closing")}</SpinnerLabel> : t("closeShare")}
              </Button>
              <Button
                type="button"
                variant="ghost"
                onClick={() => void runMutation("regenerate")}
                disabled={Boolean(working) || loading || !hasDefaultBranch}
              >
                {working === "regenerate" ? <SpinnerLabel>{t("regenerating")}</SpinnerLabel> : t("regenerate")}
              </Button>
              <CopyActionButton
                type="button"
                value={currentURL}
                messages={{ copied: t("linkCopied"), failed: t("copyFailed") }}
                onCopied={() => onOpenChange(false)}
                disabled={Boolean(working) || loading || !active}
              >
                {t("copyAndClose")}
              </CopyActionButton>
            </>
          ) : (
            <>
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={Boolean(working)}>
                {tCommon("cancel")}
              </Button>
              <Button
                type="button"
                onClick={() => void runMutation("create")}
                disabled={Boolean(working) || loading || !hasDefaultBranch}
              >
                {working === "create" ? <SpinnerLabel>{t("creating")}</SpinnerLabel> : t("createLink")}
              </Button>
            </>
          )}
                <Button
          type="button"
          variant="outline"
          disabled={generatingImage}
          onClick={() => void handleDownloadImage()}
          className="min-h-11 sm:min-h-9"
        >
          {generatingImage ? t("share.generating") : t("share.downloadImage")}
        </Button>
      </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
