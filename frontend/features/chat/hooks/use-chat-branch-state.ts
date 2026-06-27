"use client";

import * as React from "react";
import { useTranslations } from "next-intl";

import type { ChatAreaMessage, MessageAttachment } from "@/features/chat/types/messages";
import type { PendingExchange } from "@/features/chat/types/chat-runtime";
import {
  applyBranchSelectionPath,
  buildVisibleMessages,
  mapServerMessage,
  reconcileBranchSelections,
  resolveBranchSelectionPath,
} from "@/features/chat/model/chat-thread";
import type { BranchSelectionPathItem } from "@/features/chat/model/chat-thread";
import type { MessageDTO } from "@/shared/api/conversation.types";
import type { UpstreamDebugInfo } from "@/shared/api/conversation.types";
import { ApiError } from "@/shared/api/http-client";
import { useLocalizedErrorMessage } from "@/i18n/use-localized-error";

const CHAT_BRANCH_SELECTION_STORAGE_KEY = "deeix-chat:branch-selections:v1";
const NEW_BRANCH_SELECTION_KEY = "__new__";

type PendingBranchSelectionInput = {
  parentPublicID: string | null;
  userPublicID?: string;
  assistantPublicID?: string;
  tempUserPublicID: string;
  tempAssistantPublicID: string;
};

function buildPendingMessages({
  conversationID,
  pendingExchange,
  serverTreeMessages,
  serverMessagePublicIDs,
}: {
  conversationID: string | null;
  pendingExchange: PendingExchange | null;
  serverTreeMessages: ChatAreaMessage[];
  serverMessagePublicIDs: Set<string>;
}) {
  const nextMessages = [...serverTreeMessages];
  const activePublicID = conversationID?.trim() || null;
  const pendingConversationPublicID = pendingExchange?.conversationPublicID?.trim() || null;
  if (!pendingExchange) {
    return nextMessages;
  }
  if (pendingConversationPublicID && pendingConversationPublicID !== activePublicID) {
    return nextMessages;
  }
  if (!pendingConversationPublicID && activePublicID) {
    return nextMessages;
  }
  const pendingRunID = pendingExchange.runID?.trim() || "";
  if (
    pendingRunID &&
    serverTreeMessages.some((item) => item.role === "assistant" && item.runID === pendingRunID)
  ) {
    return mergePendingAssistantState(nextMessages, pendingExchange);
  }

  const userPublicID = pendingExchange.userPublicID || pendingExchange.tempUserPublicID;
  const assistantPublicID = pendingExchange.assistantPublicID || pendingExchange.tempAssistantPublicID;

  if (!serverMessagePublicIDs.has(userPublicID)) {
    const pendingAttachments = pendingExchange.userAttachments;
    const attachments: MessageAttachment[] | undefined =
      pendingAttachments && pendingAttachments.length > 0
        ? pendingAttachments.map((att) => ({
            fileID: att.fileID,
            fileName: att.fileName,
            mimeType: att.mimeType,
            detectedMime: att.detectedMime,
            fileCategory: att.fileCategory,
            sizeBytes: att.sizeBytes,
            kind: att.kind ?? (att.mimeType.startsWith("image/") ? ("image" as const) : ("file" as const)),
            previewURL: att.previewURL,
            processingStatus: att.processingStatus,
            processingReady: att.processingReady,
            processingErrorCode: att.processingErrorCode,
            processingErrorMessage: att.processingErrorMessage,
            extractStatus: att.extractStatus,
            embedStatus: att.embedStatus,
            ragReady: att.ragReady,
            ragReason: att.ragReason,
            ocrUsed: att.ocrUsed,
          }))
        : undefined;
    nextMessages.push({
      key: `${pendingExchange.key}-user`,
      publicID: userPublicID,
      parentPublicID: pendingExchange.parentPublicID,
      sourcePublicID: pendingExchange.sourcePublicID,
      role: "user",
      content: pendingExchange.userContent,
      branchReason: pendingExchange.branchReason,
      status: pendingExchange.assistantPending ? "pending" : "success",
      runID: pendingExchange.runID,
      serverMessageID: pendingExchange.userServerMessageID,
      createdAt: pendingExchange.userCreatedAt,
      isPending: pendingExchange.assistantPending,
      attachments,
    });
  }

  if (
    pendingExchange.assistantPending ||
    pendingExchange.assistantText.length > 0 ||
    !serverMessagePublicIDs.has(assistantPublicID)
  ) {
    nextMessages.push({
      key: `${pendingExchange.key}-assistant`,
      publicID: assistantPublicID,
      parentPublicID: userPublicID,
      sourcePublicID: null,
      role: "assistant",
      contentType: pendingExchange.assistantContentType,
      content: pendingExchange.assistantText,
      branchReason: pendingExchange.branchReason,
      status: pendingExchange.assistantPending ? "pending" : pendingExchange.assistantStatus ?? "success",
      runID: pendingExchange.runID,
      platformModelName: pendingExchange.platformModelName,
      serverMessageID: pendingExchange.assistantServerMessageID,
      createdAt: pendingExchange.assistantCreatedAt,
      updatedAt: pendingExchange.assistantUpdatedAt,
      isPending: pendingExchange.assistantPending,
      isStreaming: pendingExchange.assistantStreaming,
      isFileProc: Boolean(pendingExchange.assistantFileProc && !pendingExchange.assistantText),
      activityLabel: pendingExchange.assistantActivityLabel,
      imageAspectRatio: pendingExchange.assistantImageAspectRatio,
      attachments: pendingExchange.assistantAttachments,
      processTrace: pendingExchange.assistantProcessTrace,
      inlineAlert: pendingExchange.assistantInlineAlert,
      inputTokens: pendingExchange.assistantInputTokens,
      outputTokens: pendingExchange.assistantOutputTokens,
      cacheReadTokens: pendingExchange.assistantCacheReadTokens,
      cacheWriteTokens: pendingExchange.assistantCacheWriteTokens,
      reasoningTokens: pendingExchange.assistantReasoningTokens,
      latencyMS: pendingExchange.assistantLatencyMS,
      compactDone: pendingExchange.compactDone,
    });
  }

  return nextMessages;
}

function mergePendingAssistantState(messages: ChatAreaMessage[], pendingExchange: PendingExchange) {
  const pendingAlert = pendingExchange.assistantInlineAlert;
  const pendingRunID = pendingExchange.runID?.trim() || "";
  const pendingAssistantID = pendingExchange.assistantPublicID || pendingExchange.tempAssistantPublicID;
  const pendingText = pendingExchange.assistantText;
  return messages.map((item) => {
    const sameAssistant =
      item.role === "assistant" &&
      ((pendingRunID && item.runID === pendingRunID) || item.publicID === pendingAssistantID);
    if (!sameAssistant) {
      return item;
    }
    const serverStatus = item.status?.trim().toLowerCase() || "success";
    const serverHasTerminalState = !item.isPending && !item.isStreaming && serverStatus !== "pending";
    const existingAlert = item.inlineAlert;
    const nextAlert = pendingAlert
      ? {
          title: existingAlert?.title || pendingAlert.title,
          message: existingAlert?.message || pendingAlert.message,
          details: existingAlert?.details?.request?.body ? existingAlert.details : pendingAlert.details,
        }
      : existingAlert;
    return {
      ...item,
      content: serverHasTerminalState && item.content ? item.content : pendingText ? pendingText : item.content,
      contentType: pendingExchange.assistantContentType ?? item.contentType,
      isPending: pendingExchange.assistantPending,
      isStreaming: pendingExchange.assistantStreaming,
      isFileProc: Boolean(pendingExchange.assistantFileProc && !pendingText),
      activityLabel: pendingExchange.assistantActivityLabel ?? item.activityLabel,
      imageAspectRatio: pendingExchange.assistantImageAspectRatio ?? item.imageAspectRatio,
      attachments: pendingExchange.assistantAttachments ?? item.attachments,
      processTrace: pendingExchange.assistantProcessTrace ?? item.processTrace,
      inlineAlert: nextAlert,
      inputTokens: pendingExchange.assistantInputTokens ?? item.inputTokens,
      outputTokens: pendingExchange.assistantOutputTokens ?? item.outputTokens,
      cacheReadTokens: pendingExchange.assistantCacheReadTokens ?? item.cacheReadTokens,
      cacheWriteTokens: pendingExchange.assistantCacheWriteTokens ?? item.cacheWriteTokens,
      reasoningTokens: pendingExchange.assistantReasoningTokens ?? item.reasoningTokens,
      latencyMS: pendingExchange.assistantLatencyMS ?? item.latencyMS,
      compactDone: pendingExchange.compactDone ?? item.compactDone,
      platformModelName: pendingExchange.platformModelName ?? item.platformModelName,
      status: pendingExchange.assistantPending
        ? "pending"
        : serverHasTerminalState
          ? item.status
          : pendingExchange.assistantStatus ?? item.status,
    };
  });
}

function resolvePendingBranchSelectionPath(
  messages: ChatAreaMessage[],
  pendingExchange: PendingBranchSelectionInput | null,
): BranchSelectionPathItem[] {
  if (!pendingExchange) {
    return [];
  }

  const assistantPublicID = pendingExchange.assistantPublicID || pendingExchange.tempAssistantPublicID;
  const resolvedPath = resolveBranchSelectionPath(messages, assistantPublicID);
  if (resolvedPath.length > 0) {
    return resolvedPath;
  }

  const userPublicID = pendingExchange.userPublicID || pendingExchange.tempUserPublicID;
  return [
    { parentPublicID: pendingExchange.parentPublicID, publicID: userPublicID },
    { parentPublicID: userPublicID, publicID: assistantPublicID },
  ];
}

function serializeBranchSelectionPath(path: BranchSelectionPathItem[]): string {
  return path.map((item) => `${item.parentPublicID?.trim() || ""}>${item.publicID?.trim() || ""}`).join("|");
}

function branchSelectionPathResolvedByServer(
  path: BranchSelectionPathItem[],
  serverMessagePublicIDs: Set<string>,
): boolean {
  if (path.length === 0) {
    return false;
  }
  return path.every((item) => {
    const publicID = item.publicID?.trim() || "";
    return Boolean(publicID && serverMessagePublicIDs.has(publicID));
  });
}

function resolveBranchStorageKey(conversationID: string | null): string {
  return conversationID?.trim() || NEW_BRANCH_SELECTION_KEY;
}

function readBranchSelectionStore(): Record<string, Record<string, string>> {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const raw = window.localStorage.getItem(CHAT_BRANCH_SELECTION_STORAGE_KEY);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    const store: Record<string, Record<string, string>> = {};
    for (const [conversationKey, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (!value || typeof value !== "object" || Array.isArray(value)) {
        continue;
      }
      const selections: Record<string, string> = {};
      for (const [parentKey, publicID] of Object.entries(value as Record<string, unknown>)) {
        if (typeof publicID === "string" && parentKey.trim() && publicID.trim()) {
          selections[parentKey] = publicID;
        }
      }
      if (Object.keys(selections).length > 0) {
        store[conversationKey] = selections;
      }
    }
    return store;
  } catch {
    return {};
  }
}

function writeBranchSelectionStore(store: Record<string, Record<string, string>>) {
  if (typeof window === "undefined") {
    return;
  }
  try {
    if (Object.keys(store).length === 0) {
      window.localStorage.removeItem(CHAT_BRANCH_SELECTION_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(CHAT_BRANCH_SELECTION_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Keep runtime branch switching usable when localStorage is unavailable.
  }
}

function persistBranchSelections(conversationKey: string, selections: Record<string, string>) {
  const store = readBranchSelectionStore();
  if (Object.keys(selections).length === 0) {
    delete store[conversationKey];
  } else {
    store[conversationKey] = selections;
  }
  writeBranchSelectionStore(store);
}

function readPersistedBranchSelections(conversationKey: string): Record<string, string> {
  return readBranchSelectionStore()[conversationKey] ?? {};
}

function sanitizeBranchSelectionsForPersistence(
  selections: Record<string, string>,
  serverMessagePublicIDs: ReadonlySet<string>,
): Record<string, string> {
  const sanitized: Record<string, string> = {};
  for (const [parentKey, selectedPublicID] of Object.entries(selections)) {
    const selected = selectedPublicID.trim();
    if (!selected || !serverMessagePublicIDs.has(selected)) {
      continue;
    }
    if (parentKey !== "__root__" && !serverMessagePublicIDs.has(parentKey)) {
      continue;
    }
    sanitized[parentKey] = selected;
  }
  return sanitized;
}

export function useChatBranchState({
  conversationID,
  resetToken,
  messages,
  pendingExchange,
  liveRunIDs,
}: {
  conversationID: string | null;
  resetToken: number;
  messages: MessageDTO[];
  pendingExchange: PendingExchange | null;
  liveRunIDs?: ReadonlySet<string>;
}) {
  const t = useTranslations("chat.messages");
  const resolveErrorMessage = useLocalizedErrorMessage();
  const [branchSelections, setBranchSelections] = React.useState<Record<string, string>>({});
  const [submittedBranchSelectionPath, setSubmittedBranchSelectionPath] = React.useState<BranchSelectionPathItem[]>([]);
  const [hydratedBranchStorageKey, setHydratedBranchStorageKey] = React.useState<string | null>(null);
  const branchStorageKey = React.useMemo(() => resolveBranchStorageKey(conversationID), [conversationID]);

  React.useEffect(() => {
    const nextSelections = resetToken > 0 && !conversationID
      ? {}
      : readPersistedBranchSelections(branchStorageKey);
    if (resetToken > 0 && !conversationID) {
      persistBranchSelections(branchStorageKey, {});
    }
    setBranchSelections(nextSelections);
    setSubmittedBranchSelectionPath([]);
    setHydratedBranchStorageKey(branchStorageKey);
  }, [branchStorageKey, conversationID, resetToken]);

  const serverTreeMessages = React.useMemo(
    () =>
      messages.map((item) =>
        mapServerMessage(
          item,
          {
            generationInterrupted: t("generationInterrupted"),
            streamInterrupted: t("streamInterrupted"),
            imageRunning: t("imageRunning"),
            resolveErrorMessage: (errorCode: string, fallback: string, details?: UpstreamDebugInfo) =>
              resolveErrorMessage(new ApiError(fallback, 502, details, errorCode), fallback),
          },
          { liveRunIDs },
        ),
      ),
    [liveRunIDs, messages, resolveErrorMessage, t],
  );
  const serverMessagePublicIDs = React.useMemo(
    () => new Set(serverTreeMessages.map((item) => item.publicID).filter(Boolean)),
    [serverTreeMessages],
  );

  const combinedMessages = React.useMemo(
    () =>
      buildPendingMessages({
        conversationID,
        pendingExchange,
        serverTreeMessages,
        serverMessagePublicIDs,
      }),
    [conversationID, pendingExchange, serverMessagePublicIDs, serverTreeMessages],
  );
  const combinedMessagesRef = React.useRef(combinedMessages);
  React.useEffect(() => {
    combinedMessagesRef.current = combinedMessages;
  }, [combinedMessages]);
  const pendingParentPublicID = pendingExchange?.parentPublicID ?? null;
  const pendingUserPublicID = pendingExchange?.userPublicID;
  const pendingAssistantPublicID = pendingExchange?.assistantPublicID;
  const pendingTempUserPublicID = pendingExchange?.tempUserPublicID;
  const pendingTempAssistantPublicID = pendingExchange?.tempAssistantPublicID;
  const pendingBranchSelectionInput = React.useMemo<PendingBranchSelectionInput | null>(
    () =>
      pendingTempUserPublicID && pendingTempAssistantPublicID
        ? {
            parentPublicID: pendingParentPublicID,
            userPublicID: pendingUserPublicID,
            assistantPublicID: pendingAssistantPublicID,
            tempUserPublicID: pendingTempUserPublicID,
            tempAssistantPublicID: pendingTempAssistantPublicID,
          }
        : null,
    [
      pendingAssistantPublicID,
      pendingParentPublicID,
      pendingTempAssistantPublicID,
      pendingTempUserPublicID,
      pendingUserPublicID,
    ],
  );
  const pendingBranchSelectionPath = React.useMemo(
    () => resolvePendingBranchSelectionPath(combinedMessages, pendingBranchSelectionInput),
    [combinedMessages, pendingBranchSelectionInput],
  );
  const pendingBranchSelectionKey = React.useMemo(
    () => serializeBranchSelectionPath(pendingBranchSelectionPath),
    [pendingBranchSelectionPath],
  );
  React.useEffect(() => {
    if (pendingBranchSelectionPath.length === 0) {
      return;
    }
    setSubmittedBranchSelectionPath((prev) =>
      serializeBranchSelectionPath(prev) === pendingBranchSelectionKey ? prev : pendingBranchSelectionPath,
    );
  }, [pendingBranchSelectionKey, pendingBranchSelectionPath]);
  const submittedBranchSelectionKey = React.useMemo(
    () => serializeBranchSelectionPath(submittedBranchSelectionPath),
    [submittedBranchSelectionPath],
  );
  const activeBranchSelectionPath = pendingBranchSelectionPath.length > 0
    ? pendingBranchSelectionPath
    : submittedBranchSelectionPath;
  const activeBranchSelectionKey = pendingBranchSelectionPath.length > 0
    ? pendingBranchSelectionKey
    : submittedBranchSelectionKey;
  const pendingBranchSelectionPathRef = React.useRef(pendingBranchSelectionPath);
  React.useEffect(() => {
    pendingBranchSelectionPathRef.current = activeBranchSelectionPath;
  }, [activeBranchSelectionPath, activeBranchSelectionKey]);
  const pendingObsoletePublicIDs = React.useMemo(
    () => [pendingTempUserPublicID, pendingTempAssistantPublicID],
    [pendingTempAssistantPublicID, pendingTempUserPublicID],
  );
  const pendingObsoletePublicIDKey = React.useMemo(
    () => pendingObsoletePublicIDs.map((item) => item?.trim() || "").join("|"),
    [pendingObsoletePublicIDs],
  );
  const pendingObsoletePublicIDsRef = React.useRef(pendingObsoletePublicIDs);
  React.useEffect(() => {
    pendingObsoletePublicIDsRef.current = pendingObsoletePublicIDs;
  }, [pendingObsoletePublicIDs]);
  const messageStructureKey = React.useMemo(
    () =>
      combinedMessages
        .map((item) => `${item.publicID}:${item.parentPublicID ?? ""}:${item.role}`)
        .join("|"),
    [combinedMessages],
  );

  React.useEffect(() => {
    setBranchSelections((prev) => {
      const reconciled = reconcileBranchSelections(combinedMessagesRef.current, prev);
      const targetPath = pendingBranchSelectionPathRef.current;
      if (targetPath.length === 0) {
        return reconciled;
      }
      return applyBranchSelectionPath(reconciled, targetPath, pendingObsoletePublicIDsRef.current);
    });
  }, [activeBranchSelectionKey, messageStructureKey, pendingObsoletePublicIDKey]);

  React.useEffect(() => {
    if (hydratedBranchStorageKey !== branchStorageKey) {
      return;
    }
    if (serverMessagePublicIDs.size === 0 && Object.keys(branchSelections).length > 0) {
      return;
    }
    persistBranchSelections(
      branchStorageKey,
      sanitizeBranchSelectionsForPersistence(branchSelections, serverMessagePublicIDs),
    );
  }, [branchSelections, branchStorageKey, hydratedBranchStorageKey, serverMessagePublicIDs]);

  React.useEffect(() => {
    if (pendingBranchSelectionPath.length > 0 || submittedBranchSelectionPath.length === 0) {
      return;
    }
    if (!branchSelectionPathResolvedByServer(submittedBranchSelectionPath, serverMessagePublicIDs)) {
      return;
    }
    setBranchSelections((prev) => applyBranchSelectionPath(prev, submittedBranchSelectionPath));
    setSubmittedBranchSelectionPath([]);
  }, [pendingBranchSelectionPath.length, serverMessagePublicIDs, submittedBranchSelectionKey, submittedBranchSelectionPath]);

  const visibleMessages = React.useMemo(
    () => buildVisibleMessages(combinedMessages, branchSelections),
    [branchSelections, combinedMessages],
  );

  const visibleMessageCount = visibleMessages.length;
  const currentLeafMessage = visibleMessages.at(-1) ?? null;
  const showPendingAssistant = Boolean(
    pendingExchange &&
      visibleMessages.some(
        (item) =>
          item.role === "assistant" &&
          ((pendingExchange.runID && item.runID === pendingExchange.runID) ||
            item.publicID === (pendingExchange.assistantPublicID || pendingExchange.tempAssistantPublicID)) &&
          item.isStreaming,
      ),
  );

  return {
    branchSelections,
    setBranchSelections,
    combinedMessages,
    currentLeafMessage,
    serverMessagePublicIDs,
    showPendingAssistant,
    visibleMessageCount,
    visibleMessages,
  };
}
