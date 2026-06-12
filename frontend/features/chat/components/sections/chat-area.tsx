"use client";

import * as React from "react";
import { motion, useReducedMotion } from "motion/react";
import { ArrowDownToLine, ChevronDown, ChevronUp, Copy, Loader2, Quote, Search, X } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { ChatLabel } from "@/features/chat/components/sections/chat-label";
import { useMessageBookmark } from "@/features/chat/hooks/use-message-bookmark";
import { useMessageFeedback } from "@/features/chat/hooks/use-message-feedback";
import {
  AssistantMessageSkeleton,
  ChatInlineAlertCard,
  ChatMessageBot,
} from "@/features/chat/components/message/message-bot";
import { areChatAreaMessagesRenderEqual } from "@/features/chat/model/chat-message-render";
import { type AssistantReaction } from "@/features/chat/components/message/message-meta";
import type { ChatAreaMessage, MessageAttachment } from "@/features/chat/types/messages";
import { ChatMessageUser } from "@/features/chat/components/message/message-user";
import { StreamdownRender } from "@/features/chat/components/markdown/streamdown-render";
import type { OpenCodeArtifactInput } from "@/features/chat/model/chat-artifacts";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConversationShareExportIconDropdown } from "@/shared/components/conversation-share-export-menu";
import { useAppLocale } from "@/i18n/app-i18n-provider";
import { useMessageSpeech } from "@/features/chat/hooks/use-message-speech";
import type { ChatModelOption } from "@/features/chat/types/chat-runtime";
import {
  findConversationMatches,
  nextConversationMatchIndex,
} from "@/features/chat/model/conversation-find-utils";
import { writeClipboardText } from "@/shared/lib/clipboard";
import { cn } from "@/lib/utils";

function CompactDivider({ summaryPreview }: { summaryPreview: string }) {
  const t = useTranslations("chat.messages");
  const [expanded, setExpanded] = React.useState(false);
  return (
    <div className="my-4 flex flex-col items-center gap-1">
      <div className="flex w-full items-center gap-3">
        <div className="h-px flex-1 bg-border/50" />
        <button
          type="button"
          className="relative shrink-0 cursor-pointer text-[11px] text-muted-foreground/60 hover:text-muted-foreground after:absolute after:-inset-y-2 after:inset-x-0 after:content-[''] md:after:hidden"
          onClick={() => setExpanded((v) => !v)}
        >
          {t("contextCompressed")}
        </button>
        <div className="h-px flex-1 bg-border/50" />
      </div>
      {expanded && summaryPreview ? (
        <p className="max-w-lg text-center text-[11px] leading-relaxed text-muted-foreground/70">
          {summaryPreview}
        </p>
      ) : null}
    </div>
  );
}

const MESSAGE_SWITCH_TRANSITION = {
  layout: {
    duration: 0.22,
    ease: [0.16, 1, 0.3, 1] as const,
  },
  opacity: {
    duration: 0.16,
    ease: "easeOut" as const,
  },
};

type ChatAreaProps = {
  title: string;
  starred: boolean;
  canOperateConversation: boolean;
  messages: ChatAreaMessage[];
  busy: boolean;
  messageViewportRef: React.RefObject<HTMLDivElement | null>;
  messageContentRef: React.RefObject<HTMLDivElement | null>;
  messageEndRef: React.RefObject<HTMLDivElement | null>;
  onScroll: () => void;
  onScrollToLatest: () => void;
  showScrollToLatestButton: boolean;
  hasOlderMessages?: boolean;
  loadingOlderMessages?: boolean;
  onLoadOlderMessages?: () => boolean | Promise<boolean>;
  onRetryUserMessage: (message: ChatAreaMessage) => Promise<void> | void;
  onRetryAssistantMessage: (message: ChatAreaMessage, platformModelName?: string) => Promise<void> | void;
  onContinueAssistantMessage?: (message: ChatAreaMessage) => Promise<void> | void;
  onDeleteMessage: (message: ChatAreaMessage) => Promise<void> | void;
  onEditAssistantMessage: (message: ChatAreaMessage, content: string) => Promise<boolean> | boolean;
  onEditUserMessage: (message: ChatAreaMessage, content: string) => Promise<boolean> | boolean;
  onEditImageAttachment?: (attachment: MessageAttachment, sourceModelName?: string) => void;
  onOpenCodeArtifact?: (message: ChatAreaMessage, artifact: OpenCodeArtifactInput) => void;
  onCycleMessageBranch: (parentPublicID: string | null, direction: "previous" | "next") => void;
  onToggleStar?: () => void | Promise<void>;
  onRename?: (title: string) => void | Promise<void>;
  projectMenu?: React.ComponentProps<typeof ChatLabel>["projectMenu"];
  onShare?: () => void;
  shareActive?: boolean;
  onExport?: () => void | Promise<void>;
  onExportMarkdown?: () => void | Promise<void>;
  onExportImage?: () => void | Promise<void>;
  onCopyMarkdown?: () => void | Promise<void>;
  onDelete?: () => void | Promise<void>;
  onQuoteSelection?: (text: string) => void;
  readOnly?: boolean;
  modelOptions?: ChatModelOption[];
  selectedPlatformModelName?: string;
  markdownRender?: boolean;
  showModelInfo?: boolean;
  showLatency?: boolean;
  showTokenUsage?: boolean;
  showBillingCost?: boolean;
  splitRightInset?: boolean;
};

type SelectionToolbarState = {
  text: string;
  top: number;
  left: number;
};

function useStableEvent<Args extends unknown[], Return>(callback: (...args: Args) => Return) {
  const callbackRef = React.useRef(callback);
  React.useLayoutEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  return React.useCallback((...args: Args) => callbackRef.current(...args), []);
}

function ConversationFindBar({
  query,
  matchCount,
  activeIndex,
  hasOlderMessages,
  loadingOlderMessages,
  onQueryChange,
  onStep,
  onLoadOlder,
  onClose,
}: {
  query: string;
  matchCount: number;
  activeIndex: number;
  hasOlderMessages: boolean;
  loadingOlderMessages: boolean;
  onQueryChange: (query: string) => void;
  onStep: (direction: "previous" | "next") => void;
  onLoadOlder: () => void;
  onClose: () => void;
}) {
  const t = useTranslations("chat.find");
  const trimmedQuery = query.trim();
  const countLabel = trimmedQuery
    ? matchCount > 0 && activeIndex >= 0
      ? t("matchCount", { current: activeIndex + 1, total: matchCount })
      : t("noMatches")
    : t("idle");

  return (
    <div className="flex min-w-0 flex-1 items-center gap-1 rounded-xl border border-border/70 bg-background/95 p-1 shadow-xs">
      <Search className="ml-2 size-4 shrink-0 text-muted-foreground" strokeWidth={1.8} />
      <Input
        value={query}
        placeholder={t("placeholder")}
        className="h-8 border-0 px-1 shadow-none focus-visible:ring-0"
        autoFocus
        onChange={(event) => onQueryChange(event.target.value)}
      />
      <span className="hidden shrink-0 px-1 text-[11px] text-muted-foreground sm:inline">
        {countLabel}
      </span>
      {hasOlderMessages ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 px-2"
          disabled={loadingOlderMessages}
          onClick={onLoadOlder}
        >
          {loadingOlderMessages ? <Loader2 className="size-3.5 animate-spin" /> : null}
          <span className="hidden sm:inline">{t("loadOlder")}</span>
          <span className="sm:hidden">{t("loadOlderShort")}</span>
        </Button>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-8"
        disabled={matchCount === 0}
        aria-label={t("previous")}
        onClick={() => onStep("previous")}
      >
        <ChevronUp className="size-4" strokeWidth={1.8} />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-8"
        disabled={matchCount === 0}
        aria-label={t("next")}
        onClick={() => onStep("next")}
      >
        <ChevronDown className="size-4" strokeWidth={1.8} />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-8"
        aria-label={t("close")}
        onClick={onClose}
      >
        <X className="size-4" strokeWidth={1.8} />
      </Button>
    </div>
  );
}

function SelectionQuoteToolbar({
  state,
  onQuote,
  onCopy,
}: {
  state: SelectionToolbarState;
  onQuote: () => void;
  onCopy: () => void;
}) {
  const t = useTranslations("chat.selection");

  return (
    <div
      data-selection-toolbar="true"
      className="fixed z-50 flex -translate-x-1/2 items-center gap-1 rounded-full border border-border/70 bg-popover p-1 text-popover-foreground shadow-lg"
      style={{ top: state.top, left: state.left }}
      onMouseDown={(event) => event.preventDefault()}
      onTouchStart={(event) => event.preventDefault()}
    >
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="relative h-8 rounded-full px-3 text-xs after:absolute after:-inset-y-1.5 after:inset-x-0 after:content-[''] md:after:hidden"
        onClick={onQuote}
      >
        <Quote className="size-3.5" strokeWidth={1.8} />
        {t("quote")}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="relative h-8 rounded-full px-3 text-xs after:absolute after:-inset-y-1.5 after:inset-x-0 after:content-[''] md:after:hidden"
        onClick={onCopy}
      >
        <Copy className="size-3.5" strokeWidth={1.8} />
        {t("copy")}
      </Button>
    </div>
  );
}

const ChatMessageRow = React.memo(function ChatMessageRow({
  item,
  busy,
  reaction,
  bookmarked,
  onRetryUserMessage,
  onRetryAssistantMessage,
  onContinueAssistantMessage,
  onDeleteMessage,
  onEditAssistantMessage,
  onEditUserMessage,
  onEditImageAttachment,
  onCycleMessageBranch,
  onReactAssistantMessage,
  onToggleMessageBookmark,
  onOpenCodeArtifact,
  activeSpeechMessageKey,
  speechPaused,
  speechSupported,
  onToggleMessageSpeech,
  retryModelOptions,
  selectedPlatformModelName,
  markdownRender,
  showModelInfo,
  showLatency,
  showTokenUsage,
  showBillingCost,
  readOnly,
}: {
  item: ChatAreaMessage;
  busy: boolean;
  reaction: AssistantReaction;
  bookmarked: boolean;
  onRetryUserMessage: (message: ChatAreaMessage) => Promise<void> | void;
  onRetryAssistantMessage: (message: ChatAreaMessage, platformModelName?: string) => Promise<void> | void;
  onContinueAssistantMessage?: (message: ChatAreaMessage) => Promise<void> | void;
  onDeleteMessage: (message: ChatAreaMessage) => Promise<void> | void;
  onEditAssistantMessage: (message: ChatAreaMessage, content: string) => Promise<boolean> | boolean;
  onEditUserMessage: (message: ChatAreaMessage, content: string) => Promise<boolean> | boolean;
  onEditImageAttachment?: (attachment: MessageAttachment, sourceModelName?: string) => void;
  onCycleMessageBranch: (parentPublicID: string | null, direction: "previous" | "next") => void;
  onReactAssistantMessage: (publicID: string, reaction: AssistantReaction) => void;
  onToggleMessageBookmark: (publicID: string) => void;
  onOpenCodeArtifact?: (message: ChatAreaMessage, artifact: OpenCodeArtifactInput) => void;
  activeSpeechMessageKey: string | null;
  speechPaused: boolean;
  speechSupported: boolean;
  onToggleMessageSpeech: (message: ChatAreaMessage) => void;
  retryModelOptions: ChatModelOption[];
  selectedPlatformModelName: string;
  markdownRender: boolean;
  showModelInfo: boolean;
  showLatency: boolean;
  showTokenUsage: boolean;
  showBillingCost: boolean;
  readOnly: boolean;
}) {
  const t = useTranslations("chat.messages");
  const isUser = item.role === "user";
  const isAssistant = item.role === "assistant";
  const artifactActions = React.useMemo(
    () =>
      isAssistant && onOpenCodeArtifact
        ? {
            onOpenCodeArtifact: (artifact: OpenCodeArtifactInput) => onOpenCodeArtifact(item, artifact),
          }
        : undefined,
    [isAssistant, item, onOpenCodeArtifact],
  );

  const onCopy = React.useCallback(async () => {
    try {
      await navigator.clipboard.writeText(item.content);
      toast.success(t("copied"));
    } catch {
      toast.error(t("copyFailed"), { description: t("copyFailedDescription") });
    }
  }, [item.content, t]);

  if (isUser) {
    return (
      <ChatMessageUser
        item={item}
        busy={busy}
        onRetryUserMessage={onRetryUserMessage}
        onDeleteMessage={() => onDeleteMessage(item)}
        onEditUserMessage={onEditUserMessage}
        onCycleMessageBranch={onCycleMessageBranch}
        onCopy={() => void onCopy()}
        bookmarked={bookmarked}
        onToggleBookmark={() => onToggleMessageBookmark(item.publicID)}
        readOnly={readOnly}
      />
    );
  }

  if (isAssistant) {
    return (
      <ChatMessageBot
        item={item}
        busy={busy}
        reaction={reaction}
        onRetryAssistantMessage={onRetryAssistantMessage}
        onContinueAssistantMessage={onContinueAssistantMessage}
        onDeleteMessage={() => onDeleteMessage(item)}
        onEditAssistantMessage={onEditAssistantMessage}
        onCycleMessageBranch={onCycleMessageBranch}
        onReactAssistantMessage={onReactAssistantMessage}
        onCopy={() => void onCopy()}
        bookmarked={bookmarked}
        onToggleBookmark={() => onToggleMessageBookmark(item.publicID)}
        onEditImageAttachment={onEditImageAttachment}
        artifactActions={artifactActions}
        speechSupported={speechSupported}
        speechActive={activeSpeechMessageKey === item.key}
        speechPaused={speechPaused}
        onToggleSpeech={() => onToggleMessageSpeech(item)}
        retryModelOptions={retryModelOptions}
        selectedPlatformModelName={selectedPlatformModelName}
        markdownRender={markdownRender}
        showModelInfo={showModelInfo}
        showLatency={showLatency}
        showTokenUsage={showTokenUsage}
        showBillingCost={showBillingCost}
        readOnly={readOnly}
      />
    );
  }

  return (
    <div className="min-w-0 max-w-none overflow-hidden text-sm leading-8 text-foreground [overflow-wrap:anywhere]">
      {item.content.trim() && markdownRender ? (
        <StreamdownRender content={item.content} streaming={Boolean(item.isStreaming)} />
      ) : item.content.trim() ? (
        <p className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]">{item.content}</p>
      ) : null}
      {item.inlineAlert ? (
        <ChatInlineAlertCard alert={item.inlineAlert} className={item.content.trim() ? "my-4" : "mb-4"} />
      ) : null}
    </div>
  );
}, (previous, next) => (
  previous.busy === next.busy &&
  previous.reaction === next.reaction &&
  previous.markdownRender === next.markdownRender &&
  previous.showModelInfo === next.showModelInfo &&
  previous.showLatency === next.showLatency &&
  previous.showTokenUsage === next.showTokenUsage &&
  previous.showBillingCost === next.showBillingCost &&
  previous.activeSpeechMessageKey === next.activeSpeechMessageKey &&
  previous.speechPaused === next.speechPaused &&
  previous.speechSupported === next.speechSupported &&
  previous.retryModelOptions === next.retryModelOptions &&
  previous.selectedPlatformModelName === next.selectedPlatformModelName &&
  previous.onEditImageAttachment === next.onEditImageAttachment &&
  previous.onOpenCodeArtifact === next.onOpenCodeArtifact &&
  areChatAreaMessagesRenderEqual(previous.item, next.item)
));

export function ChatArea({
  title,
  starred,
  canOperateConversation,
  messages,
  busy,
  messageViewportRef,
  messageContentRef,
  messageEndRef,
  onScroll,
  onScrollToLatest,
  showScrollToLatestButton,
  hasOlderMessages = false,
  loadingOlderMessages = false,
  onLoadOlderMessages,
  onRetryUserMessage,
  onRetryAssistantMessage,
  onContinueAssistantMessage,
  onDeleteMessage,
  onEditAssistantMessage,
  onEditUserMessage,
  onEditImageAttachment,
  onOpenCodeArtifact,
  onCycleMessageBranch,
  onToggleStar,
  onRename,
  projectMenu,
  onShare,
  shareActive = false,
  onExport,
  onExportMarkdown,
  onExportImage,
  onCopyMarkdown,
  onDelete,
  onQuoteSelection,
  readOnly = false,
  modelOptions = [],
  selectedPlatformModelName = "",
  markdownRender = true,
  showModelInfo = true,
  showLatency = true,
  showTokenUsage = true,
  showBillingCost = false,
  splitRightInset = false,
}: ChatAreaProps) {
  const t = useTranslations("chat");
  const tSelection = useTranslations("chat.selection");
  const { locale } = useAppLocale();
  const { getReaction, onReactAssistantMessage } = useMessageFeedback(messages);
  const { getBookmarked, onToggleMessageBookmark } = useMessageBookmark(messages);
  const messageSpeech = useMessageSpeech(locale);
  const stableOnRetryUserMessage = useStableEvent(onRetryUserMessage);
  const stableOnRetryAssistantMessage = useStableEvent(onRetryAssistantMessage);
  const stableOnContinueAssistantMessage = useStableEvent(onContinueAssistantMessage ?? (() => undefined));
  const stableOnDeleteMessage = useStableEvent(onDeleteMessage);
  const stableOnEditAssistantMessage = useStableEvent(onEditAssistantMessage);
  const stableOnEditUserMessage = useStableEvent(onEditUserMessage);
  const stableOnEditImageAttachment = useStableEvent((attachment: MessageAttachment, sourceModelName?: string) => {
    onEditImageAttachment?.(attachment, sourceModelName);
  });
  const stableOnCycleMessageBranch = useStableEvent(onCycleMessageBranch);
  const stableOnReactAssistantMessage = useStableEvent(onReactAssistantMessage);
  const stableOnToggleMessageBookmark = useStableEvent(onToggleMessageBookmark);
  const editImageAttachmentHandler = onEditImageAttachment ? stableOnEditImageAttachment : undefined;
  const shareLabel = shareActive ? t("manageShare") : t("shareConversation");
  const shareExportLabel = t("labelMenu.shareAndExport");
  const prefersReducedMotion = useReducedMotion();
  const [findOpen, setFindOpen] = React.useState(false);
  const [findQuery, setFindQuery] = React.useState("");
  const [activeFindIndex, setActiveFindIndex] = React.useState(-1);
  const [selectionToolbar, setSelectionToolbar] = React.useState<SelectionToolbarState | null>(null);
  const findMatches = React.useMemo(
    () => findConversationMatches(messages.map((item) => ({ key: item.key, content: item.content })), findQuery),
    [findQuery, messages],
  );
  const activeFindMatch = activeFindIndex >= 0 ? findMatches[activeFindIndex] : undefined;

  React.useEffect(() => {
    setActiveFindIndex(findMatches.length > 0 ? 0 : -1);
  }, [findMatches.length, findQuery]);

  React.useEffect(() => {
    if (!activeFindMatch || !messageContentRef.current) {
      return;
    }
    const selector = `[data-chat-message-key="${CSS.escape(activeFindMatch.messageKey)}"]`;
    const target = messageContentRef.current.querySelector<HTMLElement>(selector);
    target?.scrollIntoView({ block: "center", behavior: prefersReducedMotion ? "auto" : "smooth" });
  }, [activeFindMatch, messageContentRef, prefersReducedMotion]);

  const stepFindMatch = React.useCallback(
    (direction: "previous" | "next") => {
      setActiveFindIndex((current) => nextConversationMatchIndex(current, findMatches.length, direction));
    },
    [findMatches.length],
  );
  const loadOlderForFind = React.useCallback(() => {
    void onLoadOlderMessages?.();
  }, [onLoadOlderMessages]);

  const updateSelectionToolbar = React.useCallback(() => {
    if (typeof window === "undefined") {
      return;
    }
    const root = messageContentRef.current;
    const selection = window.getSelection();
    if (!root || !selection || selection.isCollapsed || selection.rangeCount === 0) {
      setSelectionToolbar(null);
      return;
    }

    const selectedText = selection.toString().trim();
    const range = selection.getRangeAt(0);
    const container =
      range.commonAncestorContainer.nodeType === Node.ELEMENT_NODE
        ? range.commonAncestorContainer
        : range.commonAncestorContainer.parentElement;
    if (!selectedText || !(container instanceof Node) || !root.contains(container)) {
      setSelectionToolbar(null);
      return;
    }

    const rect = range.getBoundingClientRect();
    if (!rect.width && !rect.height) {
      setSelectionToolbar(null);
      return;
    }

    const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0;
    const centerX = rect.left + rect.width / 2;
    setSelectionToolbar({
      text: selectedText,
      left: Math.min(Math.max(centerX, 96), Math.max(viewportWidth - 96, 96)),
      top: Math.max(rect.top - 52, 8),
    });
  }, [messageContentRef]);

  const scheduleSelectionToolbarUpdate = React.useCallback(() => {
    window.setTimeout(updateSelectionToolbar, 0);
  }, [updateSelectionToolbar]);

  const quoteSelection = React.useCallback(() => {
    if (!selectionToolbar) {
      return;
    }
    onQuoteSelection?.(selectionToolbar.text);
    setSelectionToolbar(null);
    window.getSelection()?.removeAllRanges();
  }, [onQuoteSelection, selectionToolbar]);

  const copySelection = React.useCallback(async () => {
    if (!selectionToolbar) {
      return;
    }
    try {
      await writeClipboardText(selectionToolbar.text);
      toast.success(tSelection("copied"));
      setSelectionToolbar(null);
    } catch {
      toast.error(tSelection("copyFailed"));
    }
  }, [selectionToolbar, tSelection]);

  return (
    <>
      <div className={cn("px-3 py-2.5 md:pl-0", splitRightInset ? "md:pr-4" : "md:pr-0")}>
        <div className="flex w-full items-center justify-between gap-3">
          {findOpen ? (
            <ConversationFindBar
              query={findQuery}
              matchCount={findMatches.length}
              activeIndex={activeFindIndex}
              hasOlderMessages={hasOlderMessages}
              loadingOlderMessages={loadingOlderMessages}
              onQueryChange={setFindQuery}
              onStep={stepFindMatch}
              onLoadOlder={loadOlderForFind}
              onClose={() => {
                setFindOpen(false);
                setFindQuery("");
              }}
            />
          ) : (
            <>
              <ChatLabel
                title={title}
                starred={starred}
                onToggleStar={canOperateConversation ? onToggleStar : undefined}
                onRename={canOperateConversation ? onRename : undefined}
                projectMenu={canOperateConversation ? projectMenu : undefined}
                onShare={canOperateConversation ? onShare : undefined}
                shareActive={shareActive}
                onExport={canOperateConversation ? onExport : undefined}
                onExportMarkdown={canOperateConversation ? onExportMarkdown : undefined}
                onExportImage={canOperateConversation ? onExportImage : undefined}
                onCopyMarkdown={canOperateConversation ? onCopyMarkdown : undefined}
                onDelete={canOperateConversation ? onDelete : undefined}
              />
              <div className="flex shrink-0 items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="relative size-8 rounded-full after:absolute after:-inset-1.5 after:content-[''] md:after:hidden"
                  aria-label={t("find.open")}
                  onClick={() => setFindOpen(true)}
                >
                  <Search className="size-4" strokeWidth={1.8} />
                </Button>
                {canOperateConversation ? (
                  <ConversationShareExportIconDropdown
                    label={shareExportLabel}
                    shareLabel={shareLabel}
                    exportLabel={t("labelMenu.exportJSON")}
                    exportMarkdownLabel={t("labelMenu.exportMarkdown")}
                    exportImageLabel={t("labelMenu.exportImage")}
                    copyMarkdownLabel={t("labelMenu.copyMarkdown")}
                    active={shareActive}
                    onShare={onShare}
                    onExport={onExport}
                    onExportMarkdown={onExportMarkdown}
                    onExportImage={onExportImage}
                    onCopyMarkdown={onCopyMarkdown}
                  />
                ) : null}
              </div>
            </>
          )}
        </div>
      </div>

      <div className="relative min-h-0 flex-1 overflow-hidden">
        <div
          ref={messageViewportRef}
          className="h-full min-h-0 overflow-y-auto px-3 pb-8 pt-2 [overflow-anchor:none] md:px-6"
          aria-busy={busy}
          aria-live="polite"
          aria-relevant="additions text"
          role="log"
          onScroll={onScroll}
          onKeyUp={scheduleSelectionToolbarUpdate}
          onMouseUp={scheduleSelectionToolbarUpdate}
          onTouchEnd={scheduleSelectionToolbarUpdate}
        >
          <div
            ref={messageContentRef}
            className="mx-auto w-full max-w-[760px]"
            style={{ fontFamily: "var(--font-chat)", fontWeight: "var(--font-chat-weight)" }}
          >
            {messages.map((item, index) => {
              const previousItem = index > 0 ? messages[index - 1] : null;
              const spacingClass =
                !previousItem
                  ? ""
                  : previousItem.role === "assistant" && item.role === "user"
                    ? "mt-6 md:mt-12"
                    : "mt-4";
              const shouldAnimateLayout = !prefersReducedMotion && !item.isPending && !item.isStreaming;

              const row = (
                <ChatMessageRow
                  item={item}
                  busy={busy}
                  reaction={getReaction(item)}
                  bookmarked={getBookmarked(item)}
                  onRetryUserMessage={stableOnRetryUserMessage}
                  onRetryAssistantMessage={stableOnRetryAssistantMessage}
                  onContinueAssistantMessage={onContinueAssistantMessage ? stableOnContinueAssistantMessage : undefined}
                  onDeleteMessage={stableOnDeleteMessage}
                  onEditAssistantMessage={stableOnEditAssistantMessage}
                  onEditUserMessage={stableOnEditUserMessage}
                  onEditImageAttachment={editImageAttachmentHandler}
                  onCycleMessageBranch={stableOnCycleMessageBranch}
                  onReactAssistantMessage={stableOnReactAssistantMessage}
                  onToggleMessageBookmark={stableOnToggleMessageBookmark}
                  onOpenCodeArtifact={onOpenCodeArtifact}
                  activeSpeechMessageKey={messageSpeech.activeMessageKey}
                  speechPaused={messageSpeech.paused}
                  speechSupported={messageSpeech.supported}
                  onToggleMessageSpeech={messageSpeech.toggleMessageSpeech}
                  retryModelOptions={modelOptions}
                  selectedPlatformModelName={selectedPlatformModelName}
                  markdownRender={markdownRender}
                  showModelInfo={showModelInfo}
                  showLatency={showLatency}
                  showTokenUsage={showTokenUsage}
                  showBillingCost={showBillingCost}
                  readOnly={readOnly}
                />
              );

              const compactDivider = item.compactDone ? (
                <CompactDivider summaryPreview={item.compactDone.summary_preview} />
              ) : null;

              if (!shouldAnimateLayout) {
                return (
                  <div
                    key={item.key}
                    data-chat-message-key={item.key}
                    data-find-active={activeFindMatch?.messageKey === item.key ? "true" : undefined}
                    className={cn(
                      spacingClass,
                      "rounded-xl [contain-intrinsic-size:1px_180px] [content-visibility:auto] transition-[background-color,box-shadow] duration-200 data-[find-active=true]:bg-primary/5 data-[find-active=true]:ring-2 data-[find-active=true]:ring-primary/20",
                    )}
                  >
                    {compactDivider}
                    {row}
                  </div>
                );
              }

              return (
                <motion.div
                  key={item.key}
                  data-chat-message-key={item.key}
                  data-find-active={activeFindMatch?.messageKey === item.key ? "true" : undefined}
                  layout="position"
                  className={cn(
                    spacingClass,
                    "rounded-xl [contain-intrinsic-size:1px_180px] [content-visibility:auto] transition-[background-color,box-shadow] duration-200 data-[find-active=true]:bg-primary/5 data-[find-active=true]:ring-2 data-[find-active=true]:ring-primary/20",
                  )}
                  transition={MESSAGE_SWITCH_TRANSITION}
                  style={{ willChange: "transform" }}
                >
                  {compactDivider}
                  {row}
                </motion.div>
              );
            })}
            <div ref={messageEndRef} aria-hidden="true" className="h-px" />
          </div>
        </div>

        {showScrollToLatestButton ? (
          <button
            type="button"
            className="absolute bottom-4 left-1/2 z-20 inline-flex size-8 -translate-x-1/2 items-center justify-center rounded-full border border-border/70 bg-background text-muted-foreground shadow-md transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 after:absolute after:-inset-1.5 after:content-[''] md:after:hidden"
            aria-label={t("messages.scrollToBottom")}
            title={t("messages.scrollToBottom")}
            onClick={onScrollToLatest}
          >
            <ArrowDownToLine className="size-4" strokeWidth={1.8} />
          </button>
        ) : null}

        {selectionToolbar ? (
          <SelectionQuoteToolbar
            state={selectionToolbar}
            onQuote={quoteSelection}
            onCopy={() => void copySelection()}
          />
        ) : null}
      </div>
    </>
  );
}

export function ChatAreaSkeleton() {
  return (
    <div aria-hidden="true" className="flex h-full min-h-0 flex-col">
      <div className="shrink-0 px-3 py-2.5 md:px-0">
        <div className="flex w-full items-center justify-between gap-3">
          <div className="inline-flex h-7 max-w-full items-center gap-1 rounded-lg">
            <Skeleton className="h-4 w-32 rounded-full bg-muted/35" />
            <Skeleton className="size-4 rounded-md bg-muted/35" />
          </div>
          <Skeleton className="size-8 shrink-0 rounded-full bg-muted/35" />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-hidden px-3 pb-8 pt-2 md:px-6">
        <div className="mx-auto w-full max-w-[760px] space-y-6">
          <ChatUserMessageSkeleton widthClassName="w-[min(26rem,70%)] max-sm:w-[88%]" />

          <ChatAssistantMessageSkeleton />

          <ChatUserMessageSkeleton widthClassName="w-[min(18rem,64%)] max-sm:w-[78%]" />
        </div>
      </div>
    </div>
  );
}

function ChatUserMessageSkeleton({ widthClassName }: { widthClassName: string }) {
  return (
    <div className="flex justify-end">
      <Skeleton className={cn("h-[54px] rounded-xl bg-muted/40", widthClassName)} />
    </div>
  );
}

function ChatAssistantMessageSkeleton() {
  return (
    <div className="flex w-full flex-col items-start gap-1.5">
      <AssistantMessageSkeleton />
      <div className="flex max-w-full flex-col items-start gap-1 md:flex-row md:items-center">
        <div className="flex items-center gap-1">
          <Skeleton className="size-6 rounded-md bg-muted/35" />
          <Skeleton className="size-6 rounded-md bg-muted/35" />
          <Skeleton className="size-6 rounded-md bg-muted/35" />
          <Skeleton className="size-6 rounded-md bg-muted/35" />
        </div>
        <div className="flex min-w-0 max-w-full flex-wrap items-center gap-1">
          <Skeleton className="h-5 w-24 rounded bg-muted/35" />
          <Skeleton className="h-5 w-36 rounded bg-muted/35" />
          <Skeleton className="h-5 w-14 rounded bg-muted/35" />
        </div>
      </div>
    </div>
  );
}

export function ChatAreaLoadError({
  onRefresh,
  onNewConversation,
}: {
  onRefresh: () => void | Promise<void>;
  onNewConversation?: () => void;
}) {
  const t = useTranslations("chat.loadError");
  return (
    <CenteredEmptyState
      className="flex-1 pb-20"
      title={t("title")}
      description={
        <>
          {t("prefix")}{" "}
          <button
            type="button"
            className="rounded-sm font-medium text-foreground underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            onClick={() => void onRefresh()}
          >
          {t("refresh")}
          </button>
          {onNewConversation ? (
            <>
              {" "}
              {t("or")}{" "}
              <button
                type="button"
                className="rounded-sm font-medium text-foreground underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={onNewConversation}
              >
                {t("newChat")}
              </button>
            </>
          ) : null}
        </>
      }
    />
  );
}
