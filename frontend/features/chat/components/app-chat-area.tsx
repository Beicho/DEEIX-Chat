"use client";

import * as React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { ChatArea, ChatAreaLoadError, ChatAreaSkeleton } from "@/features/chat/components/sections/chat-area";
import { ChatArtifactWorkspace } from "@/features/chat/components/sections/chat-artifact";
import { ChatEmptyState } from "@/features/chat/components/sections/chat-empty";
import { useChatSession } from "@/features/chat/context/chat-session-context";
import { useChatArtifacts } from "@/features/chat/hooks/use-chat-artifacts";
import { useChatAttachments } from "@/features/chat/hooks/use-chat-attachments";
import { useChatComposerState } from "@/features/chat/hooks/use-chat-composer-state";
import { useChatComposerSelection } from "@/features/chat/hooks/use-chat-composer-selection";
import type { ChatAreaMessage, MessageAttachment } from "@/features/chat/types/messages";
import { useChatModelOptions } from "@/features/chat/hooks/use-chat-model-options";
import { useChatRuntime } from "@/features/chat/hooks/use-chat-runtime";
import { useChatViewerProfile } from "@/features/chat/hooks/use-chat-viewer-profile";
import { useChatConversationExport, useConversationExportAction } from "@/features/chat/hooks/use-chat-conversation-export";
import { useChatScreenshot } from "@/features/chat/hooks/use-chat-screenshot";
import { useChatVisualPrompt } from "@/features/chat/hooks/use-chat-visual-prompt";

import { ChatInput } from "@/features/chat/components/sections/chat-input";
import { ChatScreenshotPreviewDialog } from "@/features/chat/components/sections/chat-screenshot-preview-dialog";
import { resolveChatContentWidthClassName } from "@/shared/model/chat-content-width";
import {
  ConversationShareDialog,
  sharePatchFromDTO,
} from "@/features/chat/components/sections/chat-share-dialog";
import { DeleteFilesOption } from "@/shared/components/delete-files-option";
import { useSettingsChatPreferences } from "@/features/settings/hooks/use-settings-chat-preferences";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  cloneConversationOptions,
  isConversationOptionsObject,
  sanitizeConversationOptions,
} from "@/features/chat/model/conversation-options";
import { useSidebarRecents } from "@/features/recent/context/sidebar-recents-context";
import { useChatData } from "@/features/chat/hooks/use-chat-data";
import { toPendingAttachment } from "@/features/chat/model/message-submit";
import { getConversation } from "@/shared/api/conversation";
import { getMCPToolPreference, listAvailableMCPTools, putMCPToolPreference } from "@/shared/api/mcp";
import { getUserSettings, patchUserSettings } from "@/shared/api/user-settings";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import type { ConversationDTO, ConversationOptions } from "@/shared/api/conversation.types";
import type { FileObjectDTO } from "@/shared/api/file.types";
import type { MCPToolDTO, MCPToolPreferenceDTO } from "@/shared/api/mcp.types";

import { useTheme } from "@/shared/components/theme-provider";
import { Button } from "@/components/ui/button";
import { isGlobalShortcutEvent, platformModifierLabel } from "@/shared/lib/platform-shortcuts";
import {
  estimateConversationTokens,
  resolveContextUsageRatio,
  resolveContextUsageTone,
} from "@/features/chat/model/context-usage";
import { buildQuotedDraft } from "@/features/chat/model/selection-quote";
import { cn } from "@/lib/utils";

const MODEL_OPTIONS_STORAGE_PREFIX = "deeix-chat:chat-model-options:";
const TOOL_SELECTION_STORAGE_PREFIX = "deeix-chat:tool-selection:v1:";
const DEFAULT_MCP_TOOLS_SETTING_KEY = "chat.default_mcp_tool_ids";

const EMPTY_CONVERSATION_OPTIONS: ConversationOptions = {};
const TOP_LOAD_OLDER_MESSAGES_THRESHOLD_PX = 48;
const SCREENSHOT_PREVIEW_CLOSE_DELAY_MS = 220;
function dragEventContainsFiles(event: React.DragEvent<HTMLElement>): boolean {
  return Array.from(event.dataTransfer.types ?? []).includes("Files");
}

function droppedFiles(event: React.DragEvent<HTMLElement>): File[] {
  return Array.from(event.dataTransfer.files ?? []).filter((file) => file.name.trim() || file.size > 0);
}

function modelOptionsStorageKey(platformModelName: string): string {
  return `${MODEL_OPTIONS_STORAGE_PREFIX}${encodeURIComponent(platformModelName)}`;
}

function readCachedModelOptions(platformModelName: string): ConversationOptions | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(modelOptionsStorageKey(platformModelName));
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as unknown;
    return isConversationOptionsObject(parsed) ? sanitizeConversationOptions(parsed) : null;
  } catch {
    return null;
  }
}

function writeCachedModelOptions(platformModelName: string, options: ConversationOptions): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(modelOptionsStorageKey(platformModelName), JSON.stringify(sanitizeConversationOptions(options)));
  } catch {
    // localStorage may be unavailable in private browsing or strict environments.
  }
}

function removeCachedModelOptions(platformModelName: string): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.removeItem(modelOptionsStorageKey(platformModelName));
  } catch {
    // localStorage may be unavailable in private browsing or strict environments.
  }
}

function toolSelectionStorageKey(conversationPublicID: string | null): string {
  const value = conversationPublicID?.trim() || "default";
  return `${TOOL_SELECTION_STORAGE_PREFIX}${encodeURIComponent(value)}`;
}

function normalizeToolPreference(
  value: Partial<MCPToolPreferenceDTO> | null | undefined,
  conversationPublicID: string | null,
): MCPToolPreferenceDTO {
  return {
    conversationPublicID: conversationPublicID?.trim() || "",
    selectedToolIDs: Array.isArray(value?.selectedToolIDs) ? value.selectedToolIDs.filter((id) => Number.isFinite(id) && id > 0) : [],
    confirmedToolIDs: Array.isArray(value?.confirmedToolIDs) ? value.confirmedToolIDs.filter((id) => Number.isFinite(id) && id > 0) : [],
    webSearchEnabled: Boolean(value?.webSearchEnabled),
    codeSandboxEnabled: Boolean(value?.codeSandboxEnabled),
    researchMaxLLMCalls: Number.isFinite(value?.researchMaxLLMCalls) ? Math.max(0, Math.floor(value?.researchMaxLLMCalls ?? 0)) : 0,
    researchMaxToolCalls: Number.isFinite(value?.researchMaxToolCalls) ? Math.max(0, Math.floor(value?.researchMaxToolCalls ?? 0)) : 0,
    updatedAt: value?.updatedAt ?? "",
  };
}

function readCachedToolPreference(conversationPublicID: string | null): MCPToolPreferenceDTO | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const raw = window.localStorage.getItem(toolSelectionStorageKey(conversationPublicID));
    if (!raw) {
      return null;
    }
    return normalizeToolPreference(JSON.parse(raw) as Partial<MCPToolPreferenceDTO>, conversationPublicID);
  } catch {
    return null;
  }
}

function writeCachedToolPreference(conversationPublicID: string | null, value: MCPToolPreferenceDTO): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.localStorage.setItem(toolSelectionStorageKey(conversationPublicID), JSON.stringify(normalizeToolPreference(value, conversationPublicID)));
  } catch {
    // localStorage may be unavailable in private browsing or strict environments.
  }
}

function ChatEmptySuggestions({
  suggestions,
  onSelectSuggestion,
}: {
  suggestions: { label: string; prompt: string }[];
  onSelectSuggestion: (prompt: string) => void;
}) {
  const t = useTranslations("chat");

  if (suggestions.length === 0) {
    return null;
  }

  return (
    <div className="mb-5 flex flex-col gap-3 md:mb-6">
      <div className="text-left text-[11px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
        {t("emptyStateSuggestionsTitle")}
      </div>
      <div className="grid gap-2 sm:grid-cols-2">
        {suggestions.map((suggestion) => (
          <Button
            key={suggestion.prompt}
            type="button"
            variant="outline"
            className={cn(
              "relative h-9 justify-start rounded-lg border-border/70 bg-background/70 px-3 text-left text-xs font-medium text-foreground shadow-none hover:border-border hover:bg-accent hover:text-accent-foreground",
              "after:absolute after:-inset-y-1 after:inset-x-0 after:content-[''] sm:after:hidden",
            )}
            onClick={() => onSelectSuggestion(suggestion.prompt)}
          >
            <span className="block min-w-0 truncate">{suggestion.label}</span>
          </Button>
        ))}
      </div>
    </div>
  );
}

function KeyboardShortcutsDialog({
  open,
  onOpenChange,
  sendShortcut,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sendShortcut: string;
}) {
  const t = useTranslations("chat.shortcuts");
  const modifierLabel = platformModifierLabel();
  const rows = React.useMemo(
    () => [
      {
        label: t("send"),
        keys: sendShortcut === "enter" ? ["Enter"] : [modifierLabel, "Enter"],
      },
      {
        label: t("newLine"),
        keys: ["Shift", "Enter"],
      },
      {
        label: t("stop"),
        keys: ["Esc"],
      },
      {
        label: t("help"),
        keys: [modifierLabel, "/"],
      },
      {
        label: t("search"),
        keys: [modifierLabel, "K"],
      },
      {
        label: t("newChat"),
        keys: [modifierLabel, "Shift", "O"],
      },
    ],
    [modifierLabel, sendShortcut, t],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("description")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-2">
          {rows.map((row) => (
            <div key={row.label} className="flex min-h-9 items-center justify-between gap-3 rounded-lg border border-border/60 bg-muted/20 px-3 py-1.5">
              <span className="text-sm font-medium text-foreground">{row.label}</span>
              <span className="flex shrink-0 items-center gap-1">
                {row.keys.map((key) => (
                  <kbd key={`${row.label}-${key}`} className="rounded-md border border-border bg-background px-1.5 py-1 text-[11px] font-medium leading-none text-muted-foreground">
                    {key}
                  </kbd>
                ))}
              </span>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function ContextUsageIndicator({
  estimatedTokens,
  ratio,
  tone,
}: {
  estimatedTokens: number;
  ratio: number;
  tone: "default" | "warning" | "danger";
}) {
  const t = useTranslations("chat.contextUsage");
  if (estimatedTokens <= 0) {
    return null;
  }

  const percent = Math.round(ratio * 100);
  return (
    <div
      className={cn(
        "mb-2 rounded-xl border px-3 py-2 text-xs shadow-xs",
        tone === "danger"
          ? "border-destructive/30 bg-destructive/10 text-destructive"
          : tone === "warning"
            ? "border-primary/25 bg-primary/10 text-foreground"
            : "border-border/70 bg-muted/25 text-muted-foreground",
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <span className="font-medium">{t("label")}</span>
        <span className="tabular-nums">{t("value", { tokens: estimatedTokens, percent })}</span>
      </div>
      <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-200",
            tone === "danger" ? "bg-destructive" : "bg-primary",
          )}
          style={{ width: `${percent}%` }}
        />
      </div>
      {tone !== "default" ? (
        <p className="mt-1.5 leading-5">
          {tone === "danger" ? t("dangerHint") : t("warningHint")}
        </p>
      ) : null}
    </div>
  );
}

type ChatWorkspaceAreaProps = {
  conversationIDOverride?: string | null;
  comparePaneLabel?: string;
  readOnly?: boolean;
};

function parseCompareConversationIDs(value: string): string[] {
  const seen = new Set<string>();
  return value
    .split(",")
    .map((item) => item.trim())
    .filter((item) => {
      if (!item || seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    })
    .slice(0, 2);
}

function CompareSplitView({ conversationIDs }: { conversationIDs: string[] }) {
  const t = useTranslations("chat");
  const panes = conversationIDs.slice(0, 2);
  const firstPane = panes[0] ?? "";

  return (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden bg-background">
      <div className="hidden h-full min-h-0 grid-cols-2 gap-px bg-border md:grid">
        {panes.map((conversationID, index) => {
          const label = t("compareView.column", { index: index + 1 });
          return (
            <section
              key={`${conversationID}-${index}`}
              className="flex min-h-0 min-w-0 flex-col overflow-hidden bg-background"
              aria-label={label}
            >
              <div className="shrink-0 border-b border-border/70 px-3 py-2">
                <div className="truncate text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground">
                  {label}
                </div>
                <div className="mt-0.5 truncate text-xs font-medium text-foreground">{conversationID}</div>
              </div>
              <ChatWorkspaceArea
                conversationIDOverride={conversationID}
                comparePaneLabel={label}
                readOnly
              />
            </section>
          );
        })}
      </div>

      <Tabs defaultValue={firstPane} className="flex h-full min-h-0 flex-1 flex-col md:hidden">
        <div className="shrink-0 border-b border-border/70 p-2">
          <TabsList className="grid min-h-9 w-full grid-cols-2">
            {panes.map((conversationID, index) => (
              <TabsTrigger key={`${conversationID}-trigger`} value={conversationID} className="min-h-8">
                {t("compareView.column", { index: index + 1 })}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>
        {panes.map((conversationID, index) => {
          const label = t("compareView.column", { index: index + 1 });
          return (
            <TabsContent
              key={`${conversationID}-content`}
              value={conversationID}
              className="min-h-0 flex-1 overflow-hidden data-[state=inactive]:hidden"
            >
              <ChatWorkspaceArea
                conversationIDOverride={conversationID}
                comparePaneLabel={label}
                readOnly
              />
            </TabsContent>
          );
        })}
      </Tabs>
    </div>
  );
}

function parseDefaultMCPToolIDs(raw: string | null | undefined): number[] {
  const value = raw?.trim();
  if (!value) {
    return [];
  }
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    const seen = new Set<number>();
    const result: number[] = [];
    for (const item of parsed) {
      const id = typeof item === "number" ? item : Number(item);
      if (Number.isSafeInteger(id) && id > 0 && !seen.has(id)) {
        seen.add(id);
        result.push(id);
      }
    }
    return result;
  } catch {
    return [];
  }
}

function normalizeAvailableMCPTools(tools: MCPToolDTO[]): MCPToolDTO[] {
  const seen = new Set<number>();
  return tools.filter((tool) => {
    if (!Number.isSafeInteger(tool.id) || tool.id <= 0 || seen.has(tool.id)) {
      return false;
    }
    const status = typeof tool.status === "string" ? tool.status.trim() : "";
    if (status && status !== "active") {
      return false;
    }
    seen.add(tool.id);
    return true;
  });
}

function filterAvailableMCPToolIDs(toolIDs: number[], tools: MCPToolDTO[], limit?: number): number[] {
  const availableIDs = new Set(tools.map((tool) => tool.id));
  const result = toolIDs.filter((id) => availableIDs.has(id));
  return typeof limit === "number" && limit >= 0 ? result.slice(0, limit) : result;

}

export function AppChatArea() {
  const searchParams = useSearchParams();
  const compareConversationIDs = React.useMemo(
    () => parseCompareConversationIDs(searchParams.get("compare")?.trim() || ""),
    [searchParams],
  );

  if (compareConversationIDs.length >= 2) {
    return <CompareSplitView conversationIDs={compareConversationIDs} />;
  }

  return <ChatWorkspaceArea />;
}

function ChatWorkspaceArea({
  conversationIDOverride,
  comparePaneLabel,
  readOnly = false,
}: ChatWorkspaceAreaProps) {
  const t = useTranslations("chat");
  const tRecent = useTranslations("recent");
  const tScreenshot = useTranslations("chat.screenshot");
  const router = useRouter();
  const searchParams = useSearchParams();
  const hasConversationOverride = conversationIDOverride !== undefined;
  const routeConversationID = hasConversationOverride
    ? conversationIDOverride?.trim() || null
    : searchParams.get("conversation_id")?.trim() || null;
  const routeProjectID = readOnly || hasConversationOverride ? null : searchParams.get("project_id")?.trim() || null;
  const { newConversationRevision, newConversationProjectID: requestedNewConversationProjectID, requestNewConversation } = useChatSession();
  const [locallyCreatedConversationID, setLocallyCreatedConversationID] = React.useState<string | null>(null);
  const [newConversationOverride, setNewConversationOverride] = React.useState<{
    ignoredConversationID: string | null;
  } | null>(null);
  const previousNewConversationRevisionRef = React.useRef(newConversationRevision);

  React.useEffect(() => {
    if (readOnly || hasConversationOverride) {
      return;
    }
    if (previousNewConversationRevisionRef.current === newConversationRevision) {
      return;
    }
    previousNewConversationRevisionRef.current = newConversationRevision;
    setLocallyCreatedConversationID(null);
    setNewConversationOverride({
      ignoredConversationID: routeConversationID,
    });
  }, [hasConversationOverride, newConversationRevision, readOnly, routeConversationID]);

  React.useEffect(() => {
    if (readOnly || hasConversationOverride) {
      return;
    }
    if (routeConversationID) {
      setLocallyCreatedConversationID(null);
    }
  }, [hasConversationOverride, readOnly, routeConversationID]);

  React.useEffect(() => {
    if (readOnly || hasConversationOverride) {
      return;
    }
    setNewConversationOverride((prev) =>
      prev && routeConversationID !== prev.ignoredConversationID ? null : prev,
    );
  }, [hasConversationOverride, readOnly, routeConversationID]);

  const resolvedRouteConversationID = routeConversationID ?? locallyCreatedConversationID;
  const conversationID =
    newConversationOverride && resolvedRouteConversationID === newConversationOverride.ignoredConversationID
      ? null
      : resolvedRouteConversationID;
  const onNewConversationFromLoadError = React.useCallback(() => {
    const projectID = routeProjectID ?? "";
    requestNewConversation({ projectID });
    router.push(projectID ? `/chat?project_id=${encodeURIComponent(projectID)}` : "/chat");
  }, [requestNewConversation, routeProjectID, router]);
  const activeGenerationRunsRef = React.useRef<Set<string>>(new Set());
  const failedGenerationRunsRef = React.useRef<Set<string>>(new Set());
  const {
    deleteFilesByDefault,
    loaded: chatPreferencesLoaded,
    reuseModelOptions,
  } = useSettingsChatPreferences();
  const {
    items,
    projects,
    prependNewConversation,
    touchByPublicID,
    renameByPublicID,
    regenerateTitleByPublicID,
    setStarByPublicID,
    setProjectByPublicID,
    deleteByPublicID,
  } = useSidebarRecents();
  const {
    cancelResumedGeneration,
    loading,
    loadingOlder,
    errorMsg,
    hasOlder,
    loadOlderMessages,
    loadAllOlderMessages,
    messages,
    reload,
    replaceMessage,
    resumingRunID,
  } = useChatData(conversationID, {
    activeGenerationRunsRef,
    failedGenerationRunsRef,
  });
  const { greetingTitle } = useChatViewerProfile();
  const [manualConversationTitle, setManualConversationTitle] = React.useState("");
  const [shareDialogOpen, setShareDialogOpen] = React.useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
  const [shortcutsDialogOpen, setShortcutsDialogOpen] = React.useState(false);
  const [deleteFiles, setDeleteFiles] = React.useState(false);
  const deleteFilesID = React.useId();
  const activeConversation = React.useMemo(() => {
    if (!conversationID) {
      return null;
    }
    return items.find((item) => item.publicID === conversationID) ?? null;
  }, [conversationID, items]);
  const [loadedConversation, setLoadedConversation] = React.useState<ConversationDTO | null>(null);
  React.useEffect(() => {
    const normalizedConversationID = conversationID?.trim() || "";
    if (!normalizedConversationID || activeConversation?.publicID === normalizedConversationID) {
      setLoadedConversation(null);
      return;
    }

    let cancelled = false;
    async function loadConversation() {
      const token = await resolveAccessToken();
      if (!token) {
        return;
      }
      const item = await getConversation(token, normalizedConversationID);
      if (cancelled) {
        return;
      }
      setLoadedConversation(item);
    }

    void loadConversation().catch(() => {
      if (!cancelled) {
        setLoadedConversation(null);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [activeConversation?.publicID, conversationID]);
  const currentConversation =
    activeConversation ?? (loadedConversation?.publicID === conversationID ? loadedConversation : null);
  const activeRouteProject = React.useMemo(() => {
    if (!routeProjectID || conversationID) {
      return null;
    }
    return projects.find((item) => item.publicID === routeProjectID) ?? null;
  }, [conversationID, projects, routeProjectID]);
  const newConversationProjectID = !conversationID ? routeProjectID ?? requestedNewConversationProjectID : "";
  const prependNewConversationInContext = React.useCallback(
    (platformModelName?: string) => prependNewConversation(platformModelName, newConversationProjectID || undefined),
    [newConversationProjectID, prependNewConversation],
  );

  const {
    modelOptions,
    refreshModelCatalog,
    refreshModelOption,
    modelsLoading,
    modelsErrorMsg,
    sendShortcut,
    restoreDraftOnFailure,
    preserveConversationDrafts,
    inputHeight,
    contentWidth,
    markdownRender,
    showModelInfo,
    showLatency,
    showTokenUsage,
    showBillingCost,
    billingDisplayCurrency,
    billingDisplayUsdToCnyRate,
    modelOptionPolicy,
    mcpMaxSelectedTools,
    selectedPlatformModelName,
    setSelectedPlatformModelName,
  } = useChatModelOptions({
    conversationPublicID: conversationID,
    conversationModel: currentConversation?.model ?? null,
    resetToken: newConversationRevision,
  });
  const {
    conversationKey,
    draft,
    attachments,
    setDraft,
    setAttachments,
    appendAttachmentsForKey,
  } = useChatComposerState(conversationID, {
    preserveDrafts: preserveConversationDrafts,
    resetToken: newConversationRevision,
  });
  const selectedModel = React.useMemo(
    () => modelOptions.find((item) => item.platformModelName === selectedPlatformModelName) ?? null,
    [modelOptions, selectedPlatformModelName],
  );
  const modelOptionPolicyDisabled = modelOptionPolicy?.mode?.trim() === "disabled";
  const refreshModelCatalogForComposer = React.useCallback(async () => {
    await refreshModelCatalog();
  }, [refreshModelCatalog]);
  const [options, setOptions] = React.useState<ConversationOptions>({});
  const [availableTools, setAvailableTools] = React.useState<MCPToolDTO[]>([]);
  const [toolsLoading, setToolsLoading] = React.useState(true);
  const {
    selectedToolIDs,
    selectedSkills,
    setSelectedToolIDs,
    setSelectedSkills,
  } = useChatComposerSelection({
    conversationKey,
    createdConversationID: locallyCreatedConversationID,
    resetToken: newConversationRevision,
    hasConversation: Boolean(conversationID),
  });
  const [confirmedToolIDs, setConfirmedToolIDs] = React.useState<number[]>([]);
  const [webSearchEnabled, setWebSearchEnabled] = React.useState(false);
  const [codeSandboxEnabled, setCodeSandboxEnabled] = React.useState(false);
  const [researchMaxLLMCalls, setResearchMaxLLMCalls] = React.useState(0);
  const [researchMaxToolCalls, setResearchMaxToolCalls] = React.useState(0);
  const [toolPreferenceReady, setToolPreferenceReady] = React.useState(false);
  const [defaultToolIDs, setDefaultToolIDs] = React.useState<number[]>([]);
  const defaultToolIDsRef = React.useRef<number[]>([]);
  const htmlVisualPrompt = useChatVisualPrompt();

  const { resolvedTheme } = useTheme();
  const initializedOptionsModelRef = React.useRef("");
  const selectedModelDefaultOptionsRef = React.useRef<ConversationOptions>({});
  const fileDragDepthRef = React.useRef(0);
  const [fileDragActive, setFileDragActive] = React.useState(false);

  React.useEffect(() => {
    setSelectedToolIDs((current) => {
      if (current.length <= mcpMaxSelectedTools) {
        return current;
      }
      return current.slice(0, mcpMaxSelectedTools);
    });
    setConfirmedToolIDs((current) => current.slice(0, mcpMaxSelectedTools));
  }, [mcpMaxSelectedTools, setSelectedToolIDs]);


  React.useEffect(() => {
    const platformModelName = selectedModel?.platformModelName.trim() || "";
    if (!platformModelName) {
      initializedOptionsModelRef.current = "";
      selectedModelDefaultOptionsRef.current = {};
      setOptions({});
      return;
    }
    if (!chatPreferencesLoaded) {
      return;
    }
    const nextDefaultOptions = cloneConversationOptions(selectedModel.defaultOptions);
    const previousDefaultOptions = selectedModelDefaultOptionsRef.current;
    if (initializedOptionsModelRef.current !== platformModelName) {
      initializedOptionsModelRef.current = platformModelName;
      selectedModelDefaultOptionsRef.current = nextDefaultOptions;
      const cachedOptions = reuseModelOptions ? readCachedModelOptions(platformModelName) : null;
      setOptions(cloneConversationOptions(cachedOptions ?? nextDefaultOptions));
      return;
    }
    selectedModelDefaultOptionsRef.current = nextDefaultOptions;
    const previousDefaultOptionsJSON = JSON.stringify(previousDefaultOptions);
    if (previousDefaultOptionsJSON === JSON.stringify(nextDefaultOptions)) {
      return;
    }
    setOptions((currentOptions) => {
      if (JSON.stringify(currentOptions) !== previousDefaultOptionsJSON) {
        return currentOptions;
      }
      removeCachedModelOptions(platformModelName);
      return cloneConversationOptions(nextDefaultOptions);
    });
  }, [chatPreferencesLoaded, reuseModelOptions, selectedModel]);

  const setModelOptions = React.useCallback(
    (action: React.SetStateAction<ConversationOptions>) => {
      setOptions((previous) => {
        const next = typeof action === "function" ? action(previous) : action;
        const normalized = isConversationOptionsObject(next) ? sanitizeConversationOptions(next) : {};
        const platformModelName = selectedModel?.platformModelName.trim() || "";
        if (platformModelName) {
          writeCachedModelOptions(platformModelName, normalized);
        }
        return normalized;
      });
    },
    [selectedModel?.platformModelName],
  );

  const resetModelOptions = React.useCallback((defaults?: ConversationOptions) => {
    const platformModelName = selectedModel?.platformModelName.trim() || "";
    const nextDefaults = cloneConversationOptions(defaults ?? selectedModel?.defaultOptions ?? {});
    if (platformModelName) {
      removeCachedModelOptions(platformModelName);
    }
    setOptions(nextDefaults);
  }, [selectedModel]);

  const restoreBackendDefaultModelOptions = React.useCallback(async () => {
    const platformModelName = selectedModel?.platformModelName.trim() || selectedPlatformModelName.trim();
    if (!platformModelName) {
      return null;
    }
    const refreshedModel = await refreshModelOption(platformModelName);
    return refreshedModel ? cloneConversationOptions(refreshedModel.defaultOptions) : null;
  }, [refreshModelOption, selectedModel?.platformModelName, selectedPlatformModelName]);

  React.useEffect(() => {
    let cancelled = false;

    async function loadTools() {
      if (readOnly) {
        setAvailableTools([]);
        setSelectedToolIDs([]);
        setConfirmedToolIDs([]);
        setWebSearchEnabled(false);
        setCodeSandboxEnabled(false);
        setResearchMaxLLMCalls(0);
        setResearchMaxToolCalls(0);
        setToolPreferenceReady(true);
        setToolsLoading(false);
        return;
      }
      setToolsLoading(true);
      setToolPreferenceReady(false);
      try {
        const token = await resolveAccessToken();
        if (!token) {
          if (!cancelled) {
            setAvailableTools([]);
            setSelectedToolIDs([]);
            setConfirmedToolIDs([]);
            setWebSearchEnabled(false);
            setCodeSandboxEnabled(false);
            setResearchMaxLLMCalls(0);
            setResearchMaxToolCalls(0);
          }
          return;
        }
        const [toolsResult, settingsResult, preferenceResult] = await Promise.allSettled([
          listAvailableMCPTools(token),
          getUserSettings(token).catch(() => ({} as Record<string, string>)),
          getMCPToolPreference(token, conversationID ?? ""),
        ]);
        if (cancelled) {
          return;
        }
        const toolsValue = toolsResult.status === "fulfilled" ? normalizeAvailableMCPTools(toolsResult.value) : [];
        const settings = settingsResult.status === "fulfilled" ? settingsResult.value : {};
        const userDefaultToolIDs = filterAvailableMCPToolIDs(
          parseDefaultMCPToolIDs(settings[DEFAULT_MCP_TOOLS_SETTING_KEY]),
          toolsValue,
        );
        const cachedPreference = readCachedToolPreference(conversationID);
        const serverPreference = preferenceResult.status === "fulfilled" ? preferenceResult.value : null;
        const preference = normalizeToolPreference(serverPreference ?? cachedPreference, conversationID);
        const availableIDs = new Set(toolsValue.map((item) => item.id));
        const toolDefaultIDs = toolsValue.filter((item) => item.defaultEnabled).map((item) => item.id);
        const fallbackDefaultToolIDs = userDefaultToolIDs.length > 0 ? userDefaultToolIDs : toolDefaultIDs;
        const selectedIDs = (preference.selectedToolIDs.length > 0 ? preference.selectedToolIDs : fallbackDefaultToolIDs)
          .filter((id) => availableIDs.has(id))
          .slice(0, mcpMaxSelectedTools);
        const selectedIDSet = new Set(selectedIDs);
        setAvailableTools(toolsValue);
        setDefaultToolIDs(userDefaultToolIDs);
        setSelectedToolIDs((previous) => {
          if (preference.selectedToolIDs.length > 0 || !conversationID) {
            return selectedIDs;
          }
          const retained = previous.filter((id) => availableIDs.has(id)).slice(0, mcpMaxSelectedTools);
          return retained.length > 0 ? retained : selectedIDs;
        });
        setConfirmedToolIDs(preference.confirmedToolIDs.filter((id) => selectedIDSet.has(id)));
        setWebSearchEnabled(preference.webSearchEnabled);
        setCodeSandboxEnabled(preference.codeSandboxEnabled);
        setResearchMaxLLMCalls(preference.researchMaxLLMCalls);
        setResearchMaxToolCalls(preference.researchMaxToolCalls);

      } catch {
        if (!cancelled) {
          setAvailableTools([]);
          setSelectedToolIDs([]);
          setConfirmedToolIDs([]);
        }
      } finally {
        if (!cancelled) {
          setToolPreferenceReady(true);
          setToolsLoading(false);
        }
      }
    }

    void loadTools();
    return () => {
      cancelled = true;
    };
  }, [conversationID, mcpMaxSelectedTools, readOnly, setSelectedToolIDs]);

  React.useEffect(() => {
    defaultToolIDsRef.current = defaultToolIDs;
  }, [defaultToolIDs]);

  React.useEffect(() => {
    if (readOnly || conversationID || !toolPreferenceReady) {
      return;
    }
    setSelectedToolIDs(filterAvailableMCPToolIDs(defaultToolIDsRef.current, availableTools, mcpMaxSelectedTools));
  }, [availableTools, conversationID, mcpMaxSelectedTools, newConversationRevision, readOnly, setSelectedToolIDs, toolPreferenceReady]);

  React.useEffect(() => {
    if (readOnly || !toolPreferenceReady) {
      return;
    }
    const selectedIDSet = new Set(selectedToolIDs);
    const preference = normalizeToolPreference(
      {
        conversationPublicID: conversationID ?? "",
        selectedToolIDs,
        confirmedToolIDs: confirmedToolIDs.filter((id) => selectedIDSet.has(id)),
        webSearchEnabled,
        codeSandboxEnabled,
        researchMaxLLMCalls,
        researchMaxToolCalls,
      },
      conversationID,
    );
    writeCachedToolPreference(conversationID, preference);

    let cancelled = false;
    const timer = window.setTimeout(() => {
      void (async () => {
        const token = await resolveAccessToken();
        if (!token || cancelled) {
          return;
        }
        await putMCPToolPreference(token, {
          conversationPublicID: preference.conversationPublicID,
          selectedToolIDs: preference.selectedToolIDs,
          confirmedToolIDs: preference.confirmedToolIDs,
          webSearchEnabled: preference.webSearchEnabled,
          codeSandboxEnabled: preference.codeSandboxEnabled,
          researchMaxLLMCalls: preference.researchMaxLLMCalls,
          researchMaxToolCalls: preference.researchMaxToolCalls,
        }).catch(() => {
          // Local cache keeps the composer usable when preference sync is unavailable.
        });
      })();
    }, 500);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [
    codeSandboxEnabled,
    confirmedToolIDs,
    conversationID,
    researchMaxLLMCalls,
    researchMaxToolCalls,
    selectedToolIDs,
    toolPreferenceReady,
    readOnly,
    webSearchEnabled,
  ]);

  const onDefaultToolIDsChange = React.useCallback(async (nextToolIDs: number[]) => {
    const nextDefaults = filterAvailableMCPToolIDs(nextToolIDs, availableTools, mcpMaxSelectedTools);
    const previousDefaults = defaultToolIDs;
    setDefaultToolIDs(nextDefaults);
    try {
      const token = await resolveAccessToken();
      if (!token) {
        throw new Error(t("composer.sessionExpired"));
      }
      await patchUserSettings(token, {
        [DEFAULT_MCP_TOOLS_SETTING_KEY]: JSON.stringify(nextDefaults),
      });
      toast.success(t("composer.defaultMCPToolsSaved"));
    } catch (error) {
      setDefaultToolIDs(previousDefaults);
      toast.error(t("composer.defaultMCPToolsSaveFailed"), {
        description: error instanceof Error ? error.message : t("composer.retryLater"),
      });
    }
  }, [availableTools, defaultToolIDs, mcpMaxSelectedTools, t]);


  const {
    uploading,
    uploadingAttachments,
    maxFilesPerMessage,
    fileMode,
    releaseAttachments,
    onRemoveAttachment,
    onUploadFiles,
    onCaptureScreenshot,
  } = useChatAttachments({
    conversationKey,
    attachments,
    setAttachments,
    appendAttachmentsForKey,
  });

  const {
    onCycleMessageBranch,
    onDeleteMessage,
    onEditAssistantMessage,
    onEditUserMessage,
    onContinueAssistantMessage,
    onRetryAssistantMessage,
    onRetryUserMessage,
    onSendMessage,
    onStopMessage,
    onDeleteQueuedMessage,
    onEditQueuedMessage,
    onGuideQueuedMessage,
    queuedMessages,
    sending,
    visibleMessageCount,
    visibleMessages,
    isConversationMode,
  } = useChatRuntime({
    conversationID,
    resetToken: newConversationRevision,
    messages,
    activeConversation: currentConversation,
    selectedPlatformModelName,
    modelOptions,
    selectedToolIDs,
confirmedToolIDs,
    webSearchEnabled,
    codeSandboxEnabled,
    researchMaxLLMCalls,
    researchMaxToolCalls,
    selectedSkills,

    htmlVisualPromptEnabled: htmlVisualPrompt.enabled,
    htmlVisualColorMode: resolvedTheme,
    options: modelOptionPolicyDisabled ? EMPTY_CONVERSATION_OPTIONS : options,
    draft,
    attachments,
    maxFilesPerMessage,
    uploading,
    restoreDraftOnFailure: readOnly ? false : restoreDraftOnFailure,
    prependNewConversation: prependNewConversationInContext,
    onConversationCreated: setLocallyCreatedConversationID,
    touchByPublicID,
    reload,
    replaceMessage,
    setDraft,
    setAttachments,
    releaseAttachments,
    activeGenerationRunsRef,
    failedGenerationRunsRef,
    resumingRunID,
  });
  const generating = readOnly ? false : sending || Boolean(resumingRunID);
  const uploadDropDisabled = readOnly || generating || loading || uploading;

  const onStopActiveMessage = React.useCallback(() => {
    if (sending) {
      onStopMessage();
      return;
    }
    void cancelResumedGeneration();
  }, [cancelResumedGeneration, onStopMessage, sending]);

  React.useEffect(() => {
    if (readOnly) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (!isGlobalShortcutEvent(event)) {
        return;
      }
      if (event.key === "Escape" && generating) {
        event.preventDefault();
        onStopActiveMessage();
        return;
      }
      if (event.key === "/") {
        event.preventDefault();
        setShortcutsDialogOpen(true);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [generating, onStopActiveMessage, readOnly]);

  const messageContentRef = React.useRef<HTMLDivElement | null>(null);
  const loadingOlderInFlightRef = React.useRef(false);
  const onScroll = React.useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const viewport = event.currentTarget;
      const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
      if (
        viewport.scrollTop > TOP_LOAD_OLDER_MESSAGES_THRESHOLD_PX ||
        distanceFromBottom <= TOP_LOAD_OLDER_MESSAGES_THRESHOLD_PX ||
        !hasOlder ||
        loadingOlder ||
        loadingOlderInFlightRef.current
      ) {
        return;
      }

      loadingOlderInFlightRef.current = true;
      Promise.resolve(loadOlderMessages())
        .catch(() => undefined)
        .finally(() => {
          loadingOlderInFlightRef.current = false;
        });
    },
    [hasOlder, loadOlderMessages, loadingOlder],
  );


  const onEditGeneratedImageAttachment = React.useCallback(
    (attachment: MessageAttachment, sourceModelName?: string) => {
      if (readOnly) {
        return;
      }
      const alreadyAttached = attachments.some((item) => item.fileID === attachment.fileID);
      if (!alreadyAttached && maxFilesPerMessage > 0 && attachments.length >= maxFilesPerMessage) {
        toast.error(t("attachments.limitReached"), {
          description: t("attachments.maxUploadFiles", { count: maxFilesPerMessage }),
        });
        return;
      }

      const pendingAttachment = toPendingAttachment(attachment);
      setAttachments((previous) => {
        if (previous.some((item) => item.fileID === pendingAttachment.fileID)) {
          return previous;
        }
        return [...previous, pendingAttachment];
      });

      const selectedSupportsImageEdit = selectedModel?.kinds.includes("image_edit") ?? false;
      if (!selectedSupportsImageEdit) {
        const normalizedSourceModelName = sourceModelName?.trim() || "";
        const sourceModel = modelOptions.find(
          (item) => item.platformModelName === normalizedSourceModelName && item.kinds.includes("image_edit"),
        );
        const fallbackModel = sourceModel ?? modelOptions.find((item) => item.kinds.includes("image_edit"));
        if (fallbackModel) {
          setSelectedPlatformModelName(fallbackModel.platformModelName);
        }
      }

    },
    [
      attachments,
      maxFilesPerMessage,
      modelOptions,
      selectedModel,
      setAttachments,
      setSelectedPlatformModelName,
      t,
      readOnly,
    ],
  );

  const onAttachExistingFile = React.useCallback(
    (file: FileObjectDTO) => {
      const alreadyAttached = attachments.some((item) => item.fileID === file.fileID);
      if (alreadyAttached) {
        return;
      }
      if (maxFilesPerMessage > 0 && attachments.length >= maxFilesPerMessage) {
        toast.error(t("attachments.limitReached"), {
          description: t("attachments.maxUploadFiles", { count: maxFilesPerMessage }),
        });
        return;
      }
      setAttachments((previous) => {
        if (previous.some((item) => item.fileID === file.fileID)) {
          return previous;
        }
        return [
          ...previous,
          {
            fileID: file.fileID,
            fileName: file.fileName,
            mimeType: file.mimeType,
            detectedMime: file.detectedMIME,
            fileCategory: file.fileCategory,
            sizeBytes: file.sizeBytes,
            processingStatus: file.processingStatus,
            processingReady: file.processingReady,
            processingErrorCode: file.processingErrorCode,
            processingErrorMessage: file.processingErrorMessage,
            extractStatus: file.extractStatus,
            embedStatus: file.embedStatus,
            ragReady: false,
            ragReason: "",
            ocrUsed: false,
            ragOptOut: file.ragOptOut,
          },
        ];
      });
    },
    [attachments, maxFilesPerMessage, setAttachments, t],
  );

  React.useEffect(() => {
    setManualConversationTitle("");
  }, [conversationID]);

  React.useEffect(() => {
    const nextTitle = currentConversation?.title?.trim();
    if (nextTitle) {
      setManualConversationTitle(nextTitle);
    }
  }, [currentConversation?.publicID, currentConversation?.title]);

  const actionConversationID = React.useMemo(() => (conversationID || "").trim(), [conversationID]);
  const canOperateConversation = !readOnly && actionConversationID.length > 0;
  const activeConversationTitle = React.useMemo(
    () => manualConversationTitle || currentConversation?.title?.trim() || t("untitledConversation"),
    [currentConversation?.title, manualConversationTitle, t],
  );
  const activeConversationStarred = Boolean(currentConversation?.isStarred);
  const activeConversationShared = currentConversation?.shareStatus === "active" && Boolean(currentConversation.shareID?.trim());
  const shareDefaultMessagePublicIDs = React.useMemo(
    () =>
      visibleMessages
        .filter((item) => !item.isPending && Boolean(item.serverMessageID) && item.publicID.trim())
        .map((item) => item.publicID.trim()),
    [visibleMessages],
  );

  const screenshotMessages = React.useMemo(
    () => ({
      emptySelection: tScreenshot("emptySelection"),
      generating: tScreenshot("generating"),
      ready: tScreenshot("ready"),
      failed: tScreenshot("failed"),
      loadLimitReached: tScreenshot("loadLimitReached"),
      tooLarge: tScreenshot("tooLarge"),
      downloaded: tScreenshot("downloaded"),
      copied: tScreenshot("copied"),
      copyFailed: tScreenshot("copyFailed"),
      copyUnsupported: tScreenshot("copyUnsupported"),
    }),
    [tScreenshot],
  );
  const screenshot = useChatScreenshot({
    conversationID: actionConversationID || null,
    messageContentRef,
    conversationTitle: activeConversationTitle,
    onLoadAllMessages: loadAllOlderMessages,
    messages: screenshotMessages,
  });
  const screenshotPreview = screenshot.preview;
  const closeScreenshotPreview = screenshot.closePreview;
  const [screenshotPreviewOpen, setScreenshotPreviewOpen] = React.useState(false);
  const screenshotPreviewCloseTimerRef = React.useRef<number | null>(null);

  const clearScreenshotPreviewCloseTimer = React.useCallback(() => {
    if (screenshotPreviewCloseTimerRef.current === null) {
      return;
    }
    window.clearTimeout(screenshotPreviewCloseTimerRef.current);
    screenshotPreviewCloseTimerRef.current = null;
  }, []);

  React.useEffect(() => {
    if (!screenshotPreview) {
      setScreenshotPreviewOpen(false);
      return;
    }
    clearScreenshotPreviewCloseTimer();
    setScreenshotPreviewOpen(true);
  }, [clearScreenshotPreviewCloseTimer, screenshotPreview]);

  React.useEffect(() => clearScreenshotPreviewCloseTimer, [clearScreenshotPreviewCloseTimer]);

  const closeScreenshotPreviewDialog = React.useCallback(() => {
    setScreenshotPreviewOpen(false);
    clearScreenshotPreviewCloseTimer();
    screenshotPreviewCloseTimerRef.current = window.setTimeout(() => {
      screenshotPreviewCloseTimerRef.current = null;
      closeScreenshotPreview();
    }, SCREENSHOT_PREVIEW_CLOSE_DELAY_MS);
  }, [clearScreenshotPreviewCloseTimer, closeScreenshotPreview]);

  const onToggleActiveConversationStar = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    await setStarByPublicID(actionConversationID, !activeConversationStarred);
  }, [actionConversationID, activeConversationStarred, canOperateConversation, setStarByPublicID]);

  const onRenameActiveConversation = React.useCallback(
    async (title: string) => {
      if (!canOperateConversation) {
        return;
      }
      const normalized = title.trim();
      if (!normalized) {
        return;
      }
      const updated = await renameByPublicID(actionConversationID, normalized);
      setManualConversationTitle(updated?.title?.trim() || normalized);
    },
    [actionConversationID, canOperateConversation, renameByPublicID],
  );

  const onAutoRenameActiveConversation = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    try {
      const updated = await regenerateTitleByPublicID(actionConversationID);
      if (updated?.title?.trim()) {
        setManualConversationTitle(updated.title.trim());
      }
    } catch (error) {
      toast.error(t("labelMenu.autoRenameFailed"));
      throw error;
    }
  }, [actionConversationID, canOperateConversation, regenerateTitleByPublicID, t]);

  const onRequestDeleteActiveConversation = React.useCallback(() => {
    if (!canOperateConversation) {
      return;
    }
    setDeleteFiles(deleteFilesByDefault);
    setDeleteDialogOpen(true);
  }, [canOperateConversation, deleteFilesByDefault]);

  const onConfirmDeleteActiveConversation = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    const ok = await deleteByPublicID(actionConversationID, { deleteFiles });
    if (ok) {
      setDeleteDialogOpen(false);
      setDeleteFiles(false);
      router.push("/chat");
    }
  }, [actionConversationID, canOperateConversation, deleteByPublicID, deleteFiles, router]);

  const onSetActiveConversationProject = React.useCallback(
    async (projectID?: string) => {
      if (!canOperateConversation) {
        return;
      }
      await setProjectByPublicID(actionConversationID, projectID);
    },
    [actionConversationID, canOperateConversation, setProjectByPublicID],
  );

  const onShareActiveConversation = React.useCallback(() => {
    if (!canOperateConversation) {
      return;
    }
    setShareDialogOpen(true);
  }, [canOperateConversation]);

  const exportActiveConversation = useChatConversationExport({
    successMessage: t("exportJSONSuccess"),
    failureMessage: t("exportJSONFailed"),
  });
  const exportActiveConversationMarkdown = useConversationExportAction({
    successMessage: t("exportMarkdownSuccess"),
    failureMessage: t("exportMarkdownFailed"),
    format: "markdown",
  });
  const copyActiveConversationMarkdown = useConversationExportAction({
    successMessage: t("copyMarkdownSuccess"),
    failureMessage: t("copyMarkdownFailed"),
    format: "markdown",
    action: "copy",
  });
  const exportActiveConversationImage = useConversationExportAction({
    successMessage: t("exportImageSuccess"),
    failureMessage: t("exportImageFailed"),
    format: "image",
    imageLabels: {
      titleFallback: t("untitledConversation"),
      exportedAt: t("imageExport.exportedAt"),
      conversationID: t("imageExport.conversationID"),
      roleAssistant: t("imageExport.roleAssistant"),
      roleSystem: t("imageExport.roleSystem"),
      roleUser: t("imageExport.roleUser"),
      roleMessage: t("imageExport.roleMessage"),
      model: t("model"),
      attachments: t("imageExport.attachments"),
      noTextContent: t("imageExport.noTextContent"),
      truncated: t("imageExport.truncated"),
      watermark: t("imageExport.watermark"),
    },
  });

  const onExportActiveConversation = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    await exportActiveConversation(actionConversationID);
  }, [actionConversationID, canOperateConversation, exportActiveConversation]);
  const onExportActiveConversationMarkdown = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    await exportActiveConversationMarkdown(actionConversationID);
  }, [actionConversationID, canOperateConversation, exportActiveConversationMarkdown]);
  const onCopyActiveConversationMarkdown = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    await copyActiveConversationMarkdown(actionConversationID);
  }, [actionConversationID, canOperateConversation, copyActiveConversationMarkdown]);
  const onExportActiveConversationImage = React.useCallback(async () => {
    if (!canOperateConversation) {
      return;
    }
    await exportActiveConversationImage(actionConversationID);
  }, [actionConversationID, canOperateConversation, exportActiveConversationImage]);

  const messagesWithInlineError = React.useMemo<ChatAreaMessage[]>(() => {
    const errors = [
      modelsErrorMsg.trim()
        ? {
            title: t("modelListLoadFailed"),
            message: modelsErrorMsg.trim(),
          }
        : null,
    ].filter((item): item is NonNullable<typeof item> => item !== null);

    if (errors.length === 0) {
      return visibleMessages;
    }

    return [
      ...visibleMessages,
      {
        key: `chat-inline-error-${conversationID ?? "current"}`,
        publicID: `chat-inline-error-${conversationID ?? "current"}`,
        parentPublicID: visibleMessages.at(-1)?.publicID ?? null,
        sourcePublicID: null,
        role: "system",
        content: "",
        branchReason: "default",
        isPending: false,
        isStreaming: false,
        inlineAlert: {
          title: errors.map((item) => item.title).join(" / "),
          message: errors.map((item) => item.message).join("\n"),
        },
      },
    ];
  }, [conversationID, modelsErrorMsg, t, visibleMessages]);

  const artifactWorkspace = useChatArtifacts({
    conversationID,
    messages: messagesWithInlineError,
  });
  const workspaceRef = React.useRef<HTMLDivElement | null>(null);
  const artifactResizeCleanupRef = React.useRef<(() => void) | null>(null);
  const [artifactResizing, setArtifactResizing] = React.useState(false);
  const hasInlineArtifact = Boolean(artifactWorkspace.activeArtifact && artifactWorkspace.isInlineViewport);
  const workspaceGridColumns = hasInlineArtifact
    ? `minmax(0, ${1 - artifactWorkspace.artifactRatio}fr) minmax(0, ${artifactWorkspace.artifactRatio}fr)`
    : "minmax(0, 1fr) minmax(0, 0fr)";

  React.useEffect(() => () => {
    artifactResizeCleanupRef.current?.();
  }, []);

  const onArtifactResizeStart = React.useCallback((event: React.PointerEvent<HTMLButtonElement>) => {
    const workspace = workspaceRef.current;
    if (!workspace || event.button !== 0) {
      return;
    }

    event.preventDefault();
    artifactResizeCleanupRef.current?.();
    setArtifactResizing(true);
    const resizeHandle = event.currentTarget;
    const pointerID = event.pointerId;
    const startClientX = event.clientX;
    const startRatio = artifactWorkspace.artifactRatio;

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    let stopped = false;
    const stopResize = () => {
      if (stopped) {
        return;
      }

      stopped = true;
      artifactResizeCleanupRef.current = null;
      setArtifactResizing(false);
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      if (resizeHandle.hasPointerCapture(pointerID)) {
        resizeHandle.releasePointerCapture(pointerID);
      }
      window.removeEventListener("pointermove", onPointerMove);
      window.removeEventListener("pointerup", stopResize);
      window.removeEventListener("pointercancel", stopResize);
      window.removeEventListener("blur", stopResize);
      document.removeEventListener("visibilitychange", stopResizeWhenHidden);
      resizeHandle.removeEventListener("lostpointercapture", stopResize);
    };
    const updateRatio = (clientX: number) => {
      const rect = workspace.getBoundingClientRect();
      if (rect.width <= 0) {
        stopResize();
        return;
      }

      const ratio = startRatio - ((clientX - startClientX) / rect.width);
      artifactWorkspace.setArtifactRatio(ratio);
    };
    const onPointerMove = (moveEvent: PointerEvent) => updateRatio(moveEvent.clientX);
    const stopResizeWhenHidden = () => {
      if (document.visibilityState === "hidden") {
        stopResize();
      }
    };

    resizeHandle.setPointerCapture(pointerID);
    artifactResizeCleanupRef.current = stopResize;
    window.addEventListener("pointermove", onPointerMove);
    window.addEventListener("pointerup", stopResize);
    window.addEventListener("pointercancel", stopResize);
    window.addEventListener("blur", stopResize);
    document.addEventListener("visibilitychange", stopResizeWhenHidden);
    resizeHandle.addEventListener("lostpointercapture", stopResize);
  }, [artifactWorkspace]);

  const effectiveOptions = modelOptionPolicyDisabled ? EMPTY_CONVERSATION_OPTIONS : options;
  const selectedModelDefaultOptions = modelOptionPolicyDisabled
    ? EMPTY_CONVERSATION_OPTIONS
    : (selectedModel?.defaultOptions ?? EMPTY_CONVERSATION_OPTIONS);
  const inputHistory = React.useMemo(
    () =>
      visibleMessages
        .filter((item) => item.role === "user" && !item.isPending && item.content.trim())
        .map((item) => item.content.trim())
        .slice(-20),
    [visibleMessages],
  );
  const estimatedContextTokens = React.useMemo(
    () => estimateConversationTokens([...visibleMessages.map((item) => item.content), draft]),
    [draft, visibleMessages],
  );
  const contextUsageRatio = resolveContextUsageRatio(estimatedContextTokens);
  const contextUsageTone = resolveContextUsageTone(contextUsageRatio);
  const showContextUsageIndicator = contextUsageRatio >= 0.25;
  const resetFileDragState = React.useCallback(() => {
    fileDragDepthRef.current = 0;
    setFileDragActive(false);
  }, []);
  const onFileDragEnter = React.useCallback((event: React.DragEvent<HTMLDivElement>) => {
    if (!dragEventContainsFiles(event)) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    if (uploadDropDisabled) {
      return;
    }
    fileDragDepthRef.current += 1;
    setFileDragActive(true);
  }, [uploadDropDisabled]);
  const onFileDragOver = React.useCallback((event: React.DragEvent<HTMLDivElement>) => {
    if (!dragEventContainsFiles(event)) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = uploadDropDisabled ? "none" : "copy";
  }, [uploadDropDisabled]);
  const onFileDragLeave = React.useCallback((event: React.DragEvent<HTMLDivElement>) => {
    if (!dragEventContainsFiles(event)) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    fileDragDepthRef.current = Math.max(0, fileDragDepthRef.current - 1);
    if (fileDragDepthRef.current === 0) {
      setFileDragActive(false);
    }
  }, []);
  const onFileDrop = React.useCallback((event: React.DragEvent<HTMLDivElement>) => {
    if (!dragEventContainsFiles(event)) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    const files = droppedFiles(event);
    resetFileDragState();
    if (uploadDropDisabled || files.length === 0) {
      return;
    }
    void onUploadFiles(files);
  }, [onUploadFiles, resetFileDragState, uploadDropDisabled]);
  React.useEffect(() => {
    if (uploadDropDisabled) {
      resetFileDragState();
    }
  }, [resetFileDragState, uploadDropDisabled]);

  const onQuoteSelection = React.useCallback((text: string) => {
    if (readOnly) {
      return;
    }
    setDraft((currentDraft) => buildQuotedDraft(currentDraft, text));
  }, [readOnly, setDraft]);
  const onSelectedToolsChange = React.useCallback((nextToolIDs: number[]) => {
    setSelectedToolIDs(nextToolIDs);
    const nextSet = new Set(nextToolIDs);
    setConfirmedToolIDs((current) => current.filter((id) => nextSet.has(id)));
  }, []);

  const chatInputProps = {
    draft,
    loading,
    sending: generating,
    uploading,
    isConversationMode,
    maxFilesPerMessage,
    fileMode,
    sendShortcut,
    inputHeight,
    attachments,
    uploadingAttachments,
    inputHistory,
    modelOptions,
    billingDisplayCurrency,
    billingDisplayUsdToCnyRate,
    selectedPlatformModelName,
    availableTools,
    selectedToolIDs,
confirmedToolIDs,
    webSearchEnabled,
    codeSandboxEnabled,
    researchMaxLLMCalls,
    researchMaxToolCalls,
    selectedSkills,
    defaultToolIDs,
    queuedMessages,

    htmlVisualPromptEnabled: htmlVisualPrompt.enabled,
    maxSelectedTools: mcpMaxSelectedTools,
    toolsLoading,
    options: effectiveOptions,
    defaultOptions: selectedModelDefaultOptions,
    modelOptionPolicy,
    modelLoading: modelsLoading,
    dropActive: fileDragActive,
    onDraftChange: readOnly ? () => undefined : setDraft,
    onModelChange: setSelectedPlatformModelName,
onModelCatalogRefresh: refreshModelCatalogForComposer,
    onSelectedToolsChange,
    onConfirmedToolsChange: setConfirmedToolIDs,
    onWebSearchEnabledChange: setWebSearchEnabled,
    onCodeSandboxEnabledChange: setCodeSandboxEnabled,
    onResearchMaxLLMCallsChange: setResearchMaxLLMCalls,
    onResearchMaxToolCallsChange: setResearchMaxToolCalls,
    maxSelectedSkills: mcpMaxSelectedTools,
    onSelectedSkillsChange: setSelectedSkills,
    onDefaultToolsChange: onDefaultToolIDsChange,

    onHTMLVisualPromptChange: htmlVisualPrompt.setEnabled,
    onOptionsChange: setModelOptions,
    onOptionsReset: resetModelOptions,
    onOptionsDefaultRestore: restoreBackendDefaultModelOptions,
    onAttachExistingFile: readOnly ? () => undefined : onAttachExistingFile,
    onUploadFiles: readOnly ? () => undefined : onUploadFiles,
    onCaptureScreenshot: readOnly ? () => undefined : onCaptureScreenshot,
    onRemoveAttachment: readOnly ? () => undefined : onRemoveAttachment,
    onSendMessage: readOnly ? () => undefined : onSendMessage,
    onStopMessage: readOnly ? () => undefined : onStopActiveMessage,
    onDeleteQueuedMessage,
    onEditQueuedMessage,
    onGuideQueuedMessage,
  };
  const chatContentWidthClassName = resolveChatContentWidthClassName(contentWidth);
  const emptyStateSuggestions = React.useMemo(
    () => [
      {
        label: t("emptyStateSuggestions.summarizeLabel"),
        prompt: t("emptyStateSuggestions.summarizePrompt"),
      },
      {
        label: t("emptyStateSuggestions.planLabel"),
        prompt: t("emptyStateSuggestions.planPrompt"),
      },
      {
        label: t("emptyStateSuggestions.rewriteLabel"),
        prompt: t("emptyStateSuggestions.rewritePrompt"),
      },
      {
        label: t("emptyStateSuggestions.brainstormLabel"),
        prompt: t("emptyStateSuggestions.brainstormPrompt"),
      },
    ],
    [t],
  );
  const showEmptySuggestions = !readOnly && chatInputProps.draft.trim().length === 0;

  const isConversationLoading = Boolean(conversationID) && loading && visibleMessageCount === 0 && messagesWithInlineError.length === 0;
  const isConversationLoadFailed = Boolean(conversationID) && !loading && errorMsg.trim().length > 0 && visibleMessageCount === 0;
  const shouldUseCenteredComposer =
    !isConversationLoading && !isConversationLoadFailed && !isConversationMode && messagesWithInlineError.length === 0;

  return (
    <div
      className="relative flex h-full min-h-0 w-full flex-1 flex-col overflow-hidden md:overflow-visible"
      onDragEnter={readOnly ? undefined : onFileDragEnter}
      onDragOver={readOnly ? undefined : onFileDragOver}
      onDragLeave={readOnly ? undefined : onFileDragLeave}
      onDrop={readOnly ? undefined : onFileDrop}
    >
      {shouldUseCenteredComposer ? (
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <ChatEmptyState
            greetingTitle={activeRouteProject?.name || greetingTitle}
            badgeLabel={activeRouteProject ? t("projectMode") : undefined}
            badgeTooltip={activeRouteProject ? t("projectModeTooltip") : undefined}
            contentWidthClassName={chatContentWidthClassName}
          >
            {showEmptySuggestions ? (
              <ChatEmptySuggestions
                suggestions={emptyStateSuggestions}
                onSelectSuggestion={(prompt) => {
                  chatInputProps.onDraftChange(prompt);
                }}
              />
            ) : null}
            {showContextUsageIndicator ? (
              <ContextUsageIndicator
                estimatedTokens={estimatedContextTokens}
                ratio={contextUsageRatio}
                tone={contextUsageTone}
              />
            ) : null}
            {!readOnly ? (
              <div className="w-full">
                <ChatInput {...chatInputProps} />
              </div>
            ) : null}
          </ChatEmptyState>
        </div>
      ) : (
        <div
          ref={workspaceRef}
          className={cn(
            "relative grid min-h-0 flex-1 overflow-hidden",
            artifactResizing
              ? "transition-none"
              : "transition-[grid-template-columns] duration-500 ease-[cubic-bezier(0.16,1,0.3,1)]",
            hasInlineArtifact && "md:overflow-visible",
          )}
          style={{ gridTemplateColumns: workspaceGridColumns }}
        >
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
            <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
              {isConversationLoading ? (
                <ChatAreaSkeleton contentWidthClassName={chatContentWidthClassName} />
              ) : isConversationLoadFailed ? (
                <ChatAreaLoadError
                  onRefresh={reload}
                  onNewConversation={readOnly ? undefined : onNewConversationFromLoadError}
                />
              ) : (
                <ChatArea
                  title={comparePaneLabel || activeConversationTitle}
                  starred={activeConversationStarred}
                  canOperateConversation={canOperateConversation}
                  readOnly={readOnly}
                  messages={messagesWithInlineError}
                  busy={generating}
                  messageContentRef={messageContentRef}
                  onScroll={onScroll}

                  onRetryUserMessage={onRetryUserMessage}
                  onRetryAssistantMessage={onRetryAssistantMessage}
                  onContinueAssistantMessage={onContinueAssistantMessage}
                  onDeleteMessage={onDeleteMessage}
                  onEditAssistantMessage={onEditAssistantMessage}
                  onEditUserMessage={onEditUserMessage}
                  modelOptions={modelOptions}
                  selectedPlatformModelName={selectedPlatformModelName}
                  onModelChange={setSelectedPlatformModelName}
                  onModelCatalogRefresh={refreshModelCatalogForComposer}
                  onEditImageAttachment={onEditGeneratedImageAttachment}
                  onOpenCodeArtifact={artifactWorkspace.openArtifact}
                  onCycleMessageBranch={onCycleMessageBranch}
                  onToggleStar={onToggleActiveConversationStar}
                  onRename={onRenameActiveConversation}
                  onAutoRename={onAutoRenameActiveConversation}
                  projectMenu={{
                    label: t("labelMenu.moveToProject"),
                    unassignedLabel: t("labelMenu.unassignedProject"),
                    currentProjectID: currentConversation?.projectID,
                    projects,
                    onSelect: onSetActiveConversationProject,
                  }}
                  onShare={onShareActiveConversation}
                  shareActive={activeConversationShared}
                  onExport={onExportActiveConversation}
                  onExportMarkdown={onExportActiveConversationMarkdown}
                  onExportImage={onExportActiveConversationImage}
                  onCopyMarkdown={onCopyActiveConversationMarkdown}
                  onDelete={onRequestDeleteActiveConversation}
                  onQuoteSelection={readOnly ? undefined : onQuoteSelection}
                  markdownRender={markdownRender}
                  showModelInfo={showModelInfo}
                  showLatency={showLatency}
                  showTokenUsage={showTokenUsage}
                  showBillingCost={showBillingCost}
                  billingDisplayCurrency={billingDisplayCurrency}
                  billingDisplayUsdToCnyRate={billingDisplayUsdToCnyRate}
                  splitRightInset={hasInlineArtifact}
                  contentWidthClassName={chatContentWidthClassName}
                  onScreenshotFull={screenshot.captureFullConversation}
                  onScreenshotSelect={screenshot.startSelectionScreenshot}
                  screenshot={{
                    selectionMode: screenshot.selectionMode,
                    selectedIDs: screenshot.selectedIDs,
                    selectedCount: screenshot.selectedCount,
                    capturing: screenshot.capturing,
                    onToggleSelection: screenshot.toggleSelection,
                    onSelectAll: screenshot.selectMany,
                    onClearSelection: screenshot.clearSelection,
                    onPruneSelection: screenshot.pruneSelection,
                    onCapture: screenshot.captureSelectedMessages,
                    onExit: screenshot.exitSelectionMode,
                  }}
                />
              )}
            </div>

{!readOnly && !isConversationLoadFailed ? (
              <div className="relative z-10 shrink-0 px-3 pb-3 md:px-6">
                <div className={cn("mx-auto w-full", chatContentWidthClassName)}>
                  {showContextUsageIndicator ? (
                    <ContextUsageIndicator
                      estimatedTokens={estimatedContextTokens}
                      ratio={contextUsageRatio}
                      tone={contextUsageTone}
                    />
                  ) : null}

                  <ChatInput {...chatInputProps} />
                </div>
              </div>
            ) : null}
          </div>

          <ChatArtifactWorkspace
            artifact={artifactWorkspace.activeArtifact}
            artifacts={artifactWorkspace.artifacts}
            isInlineViewport={artifactWorkspace.isInlineViewport}
            onArtifactChange={artifactWorkspace.selectArtifact}
            onClose={artifactWorkspace.closeArtifact}
            onResizeReset={artifactWorkspace.resetArtifactRatio}
            onResizeStart={onArtifactResizeStart}
          />
        </div>
      )}

      {!readOnly ? (
        <KeyboardShortcutsDialog
          open={shortcutsDialogOpen}
          onOpenChange={setShortcutsDialogOpen}
          sendShortcut={sendShortcut}
        />
      ) : null}

      <ChatScreenshotPreviewDialog
        open={screenshotPreviewOpen}
        onOpenChange={(open) => {
          if (!open) {
            closeScreenshotPreviewDialog();
          }
        }}
        previewURL={screenshotPreview?.url ?? null}
        clipboardSupported={screenshot.clipboardSupported}
        onDownload={screenshot.downloadPreview}
        onCopy={screenshot.copyPreviewToClipboard}
      />


      {canOperateConversation ? (
        <>
          <ConversationShareDialog
            open={shareDialogOpen}
            onOpenChange={setShareDialogOpen}
            conversationPublicID={actionConversationID}
            conversationTitle={activeConversationTitle}
            defaultMessagePublicIDs={shareDefaultMessagePublicIDs}
            onExportImage={onExportActiveConversationImage}
            onShareChange={(share) => {
              touchByPublicID(actionConversationID, sharePatchFromDTO(share));
            }}
          />

          <AlertDialog
            open={deleteDialogOpen}
            onOpenChange={(open) => {
              setDeleteDialogOpen(open);
              if (!open) {
                setDeleteFiles(false);
              }
            }}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{tRecent("dialogs.deleteTitle")}</AlertDialogTitle>
                <AlertDialogDescription>
                  {tRecent("dialogs.deleteDescription", {
                    label: tRecent("deleteConversationLabel", { title: activeConversationTitle }),
                  })}
                </AlertDialogDescription>
                <DeleteFilesOption
                  id={deleteFilesID}
                  checked={deleteFiles}
                  onCheckedChange={setDeleteFiles}
                />
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{tRecent("dialogs.cancel")}</AlertDialogCancel>
                <AlertDialogAction variant="destructive" onClick={() => void onConfirmDeleteActiveConversation()}>
                  {tRecent("dialogs.delete")}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </>
      ) : null}
    </div>
  );
}
