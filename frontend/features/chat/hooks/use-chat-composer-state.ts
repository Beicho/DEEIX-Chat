"use client";

import * as React from "react";

import {
  deleteConversationDraft,
  getConversationDraft,
  upsertConversationDraft,
} from "@/shared/api/conversation";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import type { ConversationDraftDTO } from "@/shared/api/conversation.types";
import type { PendingAttachment } from "@/features/chat/types/chat-runtime";

const CHAT_COMPOSER_STORAGE_KEY = "deeix-chat:chat-composer:v1";
const NEW_CONVERSATION_COMPOSER_KEY = "__new__";
const TRANSIENT_COMPOSER_KEY = "__transient__";

type PersistedAttachment = Pick<
  PendingAttachment,
  | "fileID"
  | "fileName"
  | "mimeType"
  | "sizeBytes"
  | "detectedMime"
  | "fileCategory"
  | "processingStatus"
  | "processingReady"
  | "processingErrorCode"
  | "processingErrorMessage"
  | "extractStatus"
  | "embedStatus"
  | "ragReady"
  | "ragReason"
  | "ocrUsed"
>;

type PersistedComposerEntry = {
  draft: string;
  attachments: PersistedAttachment[];
  updatedAt: string;
};

type PersistedComposerStore = Record<string, PersistedComposerEntry>;

type ComposerState = {
  conversationKey: string;
  draft: string;
  attachments: PendingAttachment[];
  updatedAt: string | null;
};

const useIsomorphicLayoutEffect = typeof window === "undefined" ? React.useEffect : React.useLayoutEffect;

function sanitizeAttachments(items: PendingAttachment[]): PersistedAttachment[] {
  return items.map((item) => ({
    fileID: item.fileID,
    fileName: item.fileName,
    mimeType: item.mimeType,
    sizeBytes: item.sizeBytes,
    detectedMime: item.detectedMime,
    fileCategory: item.fileCategory,
    processingStatus: item.processingStatus,
    processingReady: item.processingReady,
    processingErrorCode: item.processingErrorCode,
    processingErrorMessage: item.processingErrorMessage,
    extractStatus: item.extractStatus,
    embedStatus: item.embedStatus,
    ragReady: item.ragReady,
    ragReason: item.ragReason,
    ocrUsed: item.ocrUsed,
  }));
}

function mergeAttachmentsByFileID<T extends Pick<PendingAttachment, "fileID">>(current: T[], incoming: T[]): T[] {
  if (incoming.length === 0) {
    return current;
  }
  const seen = new Set(current.map((item) => item.fileID));
  const next = [...current];
  for (const item of incoming) {
    if (seen.has(item.fileID)) {
      continue;
    }
    seen.add(item.fileID);
    next.push(item);
  }
  return next;
}

function restoreAttachments(items: PersistedAttachment[]): PendingAttachment[] {
  return items.map((item) => ({
    ...item,
    previewURL: undefined,
  }));
}

function isPersistedAttachment(value: unknown): value is PersistedAttachment {
  if (!value || typeof value !== "object") {
    return false;
  }

  const item = value as Record<string, unknown>;
  return (
    typeof item.fileID === "string" &&
    typeof item.fileName === "string" &&
    typeof item.mimeType === "string" &&
    typeof item.sizeBytes === "number"
  );
}

function readComposerStore(): PersistedComposerStore {
  if (typeof window === "undefined") {
    return {};
  }

  try {
    const raw = window.localStorage.getItem(CHAT_COMPOSER_STORAGE_KEY);
    if (!raw) {
      return {};
    }

    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object") {
      return {};
    }

    const nextStore: PersistedComposerStore = {};
    for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (!value || typeof value !== "object") {
        continue;
      }
      const entry = value as Record<string, unknown>;
      const draft = typeof entry.draft === "string" ? entry.draft : "";
      const attachments = Array.isArray(entry.attachments)
        ? entry.attachments.filter(isPersistedAttachment)
        : [];
      const updatedAt = typeof entry.updatedAt === "string" ? entry.updatedAt : new Date(0).toISOString();
      nextStore[key] = {
        draft,
        attachments,
        updatedAt,
      };
    }
    return nextStore;
  } catch {
    return {};
  }
}

function writeComposerStore(store: PersistedComposerStore) {
  if (typeof window === "undefined") {
    return;
  }

  try {
    if (Object.keys(store).length === 0) {
      window.localStorage.removeItem(CHAT_COMPOSER_STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(CHAT_COMPOSER_STORAGE_KEY, JSON.stringify(store));
  } catch {
    // Ignore storage quota / serialization issues and keep runtime state usable.
  }
}

function createEmptyComposerState(conversationKey: string): ComposerState {
  return {
    conversationKey,
    draft: "",
    attachments: [],
    updatedAt: null,
  };
}

function hasComposerContent(state: Pick<ComposerState, "draft" | "attachments">): boolean {
  return state.draft.trim().length > 0 || state.attachments.length > 0;
}

function composerFingerprint(state: Pick<ComposerState, "draft" | "attachments">): string {
  return JSON.stringify({
    draft: state.draft,
    attachments: sanitizeAttachments(state.attachments),
  });
}

function isRemoteDraftNewer(remoteUpdatedAt: string | null, localUpdatedAt: string | null): boolean {
  if (!remoteUpdatedAt) {
    return false;
  }
  if (!localUpdatedAt) {
    return true;
  }
  const remoteTime = Date.parse(remoteUpdatedAt);
  const localTime = Date.parse(localUpdatedAt);
  return Number.isFinite(remoteTime) && (!Number.isFinite(localTime) || remoteTime > localTime);
}

function composerStateFromDraftDTO(conversationKey: string, dto: ConversationDraftDTO): ComposerState {
  return {
    conversationKey,
    draft: dto.draft ?? "",
    attachments: restoreAttachments(dto.attachments ?? []),
    updatedAt: dto.updatedAt ?? null,
  };
}

// ComposerStorageOps centralizes localStorage access and avoids repeated readComposerStore() calls.
const ComposerStorageOps = {
  readEntry(conversationKey: string): ComposerState {
    const entry = readComposerStore()[conversationKey];
    return {
      conversationKey,
      draft: entry?.draft ?? "",
      attachments: restoreAttachments(entry?.attachments ?? []),
      updatedAt: entry?.updatedAt ?? null,
    };
  },

  writeEntry(conversationKey: string, draft: string, attachments: PendingAttachment[]) {
    const store = readComposerStore();
    const normalizedAttachments = sanitizeAttachments(attachments);

    if (!draft.trim() && normalizedAttachments.length === 0) {
      delete store[conversationKey];
    } else {
      store[conversationKey] = {
        draft,
        attachments: normalizedAttachments,
        updatedAt: new Date().toISOString(),
      };
    }
    writeComposerStore(store);
  },

  removeEntry(conversationKey: string) {
    const store = readComposerStore();
    delete store[conversationKey];
    writeComposerStore(store);
  },

  appendAttachments(conversationKey: string, items: PendingAttachment[]) {
    if (items.length === 0) {
      return;
    }
    const store = readComposerStore();
    const existing = store[conversationKey];
    const attachments = mergeAttachmentsByFileID(existing?.attachments ?? [], sanitizeAttachments(items));
    store[conversationKey] = {
      draft: existing?.draft ?? "",
      attachments,
      updatedAt: new Date().toISOString(),
    };
    writeComposerStore(store);
  },
};

export function resolveConversationComposerKey(conversationID: string | null): string {
  return conversationID?.trim() || NEW_CONVERSATION_COMPOSER_KEY;
}

export function useChatComposerState(
  conversationID: string | null,
  {
    preserveDrafts = true,
    resetToken = 0,
    transient = false,
  }: {
    preserveDrafts?: boolean;
    resetToken?: number;
    transient?: boolean;
  } = {},
) {
  const conversationKey = React.useMemo(
    () => transient ? TRANSIENT_COMPOSER_KEY : resolveConversationComposerKey(conversationID),
    [conversationID, transient],
  );
  const persistenceEnabled = preserveDrafts && !transient;
  const [state, setState] = React.useState<ComposerState>(() => createEmptyComposerState(conversationKey));
  const [hydratedConversationKey, setHydratedConversationKey] = React.useState<string | null>(null);
  const [serverHydratedConversationKey, setServerHydratedConversationKey] = React.useState<string | null>(null);
  const serverHydrateSeqRef = React.useRef(0);
  const serverSyncTimerRef = React.useRef<number | null>(null);

  React.useEffect(() => {
    if (resetToken <= 0 || conversationID) {
      return;
    }
    if (!transient) {
      ComposerStorageOps.removeEntry(conversationKey);
    }
    setHydratedConversationKey(conversationKey);
    setServerHydratedConversationKey(conversationKey);
    setState(createEmptyComposerState(conversationKey));
  }, [conversationID, conversationKey, resetToken, transient]);

  useIsomorphicLayoutEffect(() => {
    if (!persistenceEnabled) {
      if (!transient) {
        ComposerStorageOps.removeEntry(conversationKey);
      }
      setState((prev) => (prev.conversationKey === conversationKey ? prev : createEmptyComposerState(conversationKey)));
      setHydratedConversationKey(conversationKey);
      setServerHydratedConversationKey(conversationKey);
      return;
    }

    const nextState = ComposerStorageOps.readEntry(conversationKey);
    setState((prev) => {
      const nextHasContent = hasComposerContent(nextState);
      const prevMatchesConversation = prev.conversationKey === conversationKey;
      const prevHasContent = prevMatchesConversation && hasComposerContent(prev);

      if (!nextHasContent && !prevHasContent) {
        return prevMatchesConversation ? prev : createEmptyComposerState(conversationKey);
      }

      if (
        prevMatchesConversation &&
        prev.draft === nextState.draft &&
        prev.attachments.length === nextState.attachments.length &&
        prev.attachments.every(
          (item, index) =>
            item.fileID === nextState.attachments[index]?.fileID &&
            item.fileName === nextState.attachments[index]?.fileName &&
            item.mimeType === nextState.attachments[index]?.mimeType &&
            item.sizeBytes === nextState.attachments[index]?.sizeBytes &&
            item.processingStatus === nextState.attachments[index]?.processingStatus &&
            item.processingReady === nextState.attachments[index]?.processingReady,
        )
      ) {
        return prev;
      }

      return nextHasContent ? nextState : createEmptyComposerState(conversationKey);
    });
    setHydratedConversationKey(conversationKey);
  }, [conversationKey, persistenceEnabled, transient]);

  React.useEffect(() => {
    if (hydratedConversationKey !== state.conversationKey || state.conversationKey !== conversationKey) {
      return;
    }
    if (!persistenceEnabled) {
      if (!transient) {
        ComposerStorageOps.removeEntry(state.conversationKey);
      }
      return;
    }
    ComposerStorageOps.writeEntry(state.conversationKey, state.draft, state.attachments);
  }, [conversationKey, hydratedConversationKey, persistenceEnabled, state, transient]);

  React.useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    if (serverSyncTimerRef.current) {
      clearTimeout(serverSyncTimerRef.current);
      serverSyncTimerRef.current = null;
    }
    if (
      hydratedConversationKey !== state.conversationKey ||
      serverHydratedConversationKey !== state.conversationKey ||
      !preserveDrafts
    ) {
      return;
    }

    const snapshot = {
      conversationKey: state.conversationKey,
      draft: state.draft,
      attachments: sanitizeAttachments(state.attachments),
    };

    serverSyncTimerRef.current = window.setTimeout(() => {
      void resolveAccessToken().then((accessToken) => {
        if (!accessToken) {
          return undefined;
        }
        if (!snapshot.draft.trim() && snapshot.attachments.length === 0) {
          return deleteConversationDraft(accessToken, snapshot.conversationKey).catch(() => undefined);
        }
        return upsertConversationDraft(accessToken, snapshot.conversationKey, {
          draft: snapshot.draft,
          attachments: snapshot.attachments,
        }).catch(() => undefined);
      });
    }, 800);

    return () => {
      if (serverSyncTimerRef.current) {
        clearTimeout(serverSyncTimerRef.current);
        serverSyncTimerRef.current = null;
      }
    };
  }, [
    hydratedConversationKey,
    preserveDrafts,
    serverHydratedConversationKey,
    state.attachments,
    state.conversationKey,
    state.draft,
  ]);

  const visibleState = state.conversationKey === conversationKey ? state : createEmptyComposerState(conversationKey);

  const setDraft = React.useCallback((value: React.SetStateAction<string>) => {
    setHydratedConversationKey(conversationKey);
    setState((prev) => ({
      ...(prev.conversationKey === conversationKey ? prev : createEmptyComposerState(conversationKey)),
      draft:
        typeof value === "function"
          ? value(prev.conversationKey === conversationKey ? prev.draft : "")
          : value,
      updatedAt: new Date().toISOString(),
    }));
  }, [conversationKey]);

  const setAttachments = React.useCallback((value: React.SetStateAction<PendingAttachment[]>) => {
    setHydratedConversationKey(conversationKey);
    setState((prev) => ({
      ...(prev.conversationKey === conversationKey ? prev : createEmptyComposerState(conversationKey)),
      attachments:
        typeof value === "function"
          ? value(prev.conversationKey === conversationKey ? prev.attachments : [])
          : value,
      updatedAt: new Date().toISOString(),
    }));
  }, [conversationKey]);

  const appendAttachmentsForKey = React.useCallback((targetConversationKey: string, items: PendingAttachment[]) => {
    if (items.length === 0) {
      return;
    }

    if (conversationKey === targetConversationKey) {
      setHydratedConversationKey(targetConversationKey);
      setState((prev) => ({
        ...(prev.conversationKey === targetConversationKey ? prev : createEmptyComposerState(targetConversationKey)),
        attachments: mergeAttachmentsByFileID(
          prev.conversationKey === targetConversationKey ? prev.attachments : [],
          items,
        ),
        updatedAt: new Date().toISOString(),
      }));
      return;
    }

    if (!persistenceEnabled) {
      return;
    }

    ComposerStorageOps.appendAttachments(targetConversationKey, items);
  }, [conversationKey, persistenceEnabled]);

  return {
    conversationKey: visibleState.conversationKey,
    draft: visibleState.draft,
    attachments: visibleState.attachments,
    setDraft,
    setAttachments,
    appendAttachmentsForKey,
  };
}
