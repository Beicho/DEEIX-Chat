"use client";

import * as React from "react";
import dynamic from "next/dynamic";
import { BookOpen, Camera, Image, ImageOff, ImagePlus, Save, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { AudioLines } from "@/components/animate-ui/icons/audio-lines";
import { Blocks } from "@/components/animate-ui/icons/blocks";
import { Pause } from "@/components/animate-ui/icons/pause";
import { Plus } from "@/components/animate-ui/icons/plus";
import { Send } from "@/components/animate-ui/icons/send";
import { Link as LinkIcon } from "@/components/animate-ui/icons/link";
import { Crop } from "@/components/animate-ui/icons/crop";
import { X as XIcon } from "@/components/animate-ui/icons/x";
import type {
  ChatModelOption,
  PendingAttachment,
  UploadingAttachment,
} from "@/features/chat/types/chat-runtime";
import { useSpeechInput } from "@/features/chat/hooks/use-speech-input";
import { ChatMCP } from "@/features/chat/components/sections/chat-mcp";
import { ChatModelPicker } from "@/features/chat/components/sections/chat-model-picker";
import { ChatModelConfig } from "@/features/chat/components/sections/chat-model-config";
import { formatBytes, resolveFileIcon } from "@/features/files/utils/file-display";
import type { ChatSubmitDecision } from "@/features/chat/model/chat-task";
import { isMediaSubmitTask, resolveChatSubmitDecision } from "@/features/chat/model/chat-task";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupTextarea,
} from "@/components/ui/input-group";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { resolveFileProcessingBadge, resolveFileProcessingToneClass } from "@/shared/lib/file-processing";
import { cn } from "@/lib/utils";
import type { ConversationOptions } from "@/shared/api/conversation.types";
import type { MCPToolDTO } from "@/shared/api/mcp.types";
import type { ModelOptionPolicy } from "@/shared/lib/model-option-policy";
import type { SendShortcut } from "@/features/settings/types/settings";
import { isSendShortcutEvent } from "@/shared/lib/platform-shortcuts";
import { useIsMobile } from "@/shared/hooks/use-mobile";
import {
  createPastedTextFile,
  shouldConvertPasteToTextFile,
} from "@/features/chat/model/paste-to-file";
import {
  buildDraftFromPromptTemplate,
  filterPromptTemplates,
  normalizePromptTemplate,
  recordPromptTemplateUsage,
  type PromptTemplate,
} from "@/features/chat/model/prompt-templates";

const FilePreviewDialog = dynamic(
  () => import("@/features/files/components/preview/file-preview-dialog").then((module) => module.FilePreviewDialog),
  { ssr: false },
);

const PROMPT_TEMPLATE_STORAGE_KEY = "deeix-chat:prompt-templates:v1";
const PROMPT_TEMPLATE_RECENT_STORAGE_KEY = "deeix-chat:recent-prompt-templates:v1";
const PROMPT_TEMPLATE_RECENT_LIMIT = 5;

type ChatInputProps = {
  draft: string;
  loading: boolean;
  sending: boolean;
  uploading: boolean;
  isConversationMode: boolean;
  maxFilesPerMessage: number;
  fileMode?: "auto" | "full_context" | "rag";
  sendShortcut?: SendShortcut;
  inputHeight?: "compact" | "standard" | "loose";
  attachments: PendingAttachment[];
  uploadingAttachments: UploadingAttachment[];
  inputHistory?: string[];
  modelOptions: ChatModelOption[];
  selectedPlatformModelName: string;
  availableTools: MCPToolDTO[];
  selectedToolIDs: number[];
  confirmedToolIDs: number[];
  webSearchEnabled: boolean;
  codeSandboxEnabled: boolean;
  researchMaxLLMCalls: number;
  researchMaxToolCalls: number;
  htmlVisualPromptEnabled: boolean;
  maxSelectedTools: number;
  toolsLoading: boolean;
  options: ConversationOptions;
  defaultOptions: ConversationOptions;
  modelOptionPolicy: ModelOptionPolicy | null;
  modelLoading: boolean;
  modelDisabled?: boolean;
  dropActive?: boolean;
  onDraftChange: (value: string) => void;
  onModelChange: (platformModelName: string) => void;
  onSelectedToolsChange: (toolIDs: number[]) => void;
  onConfirmedToolsChange: (toolIDs: number[]) => void;
  onWebSearchEnabledChange: (enabled: boolean) => void;
  onCodeSandboxEnabledChange: (enabled: boolean) => void;
  onResearchMaxLLMCallsChange: (value: number) => void;
  onResearchMaxToolCallsChange: (value: number) => void;
  onHTMLVisualPromptChange: (enabled: boolean) => void;
  onOptionsChange: React.Dispatch<React.SetStateAction<ConversationOptions>>;
  onOptionsReset: (defaults?: ConversationOptions) => void;
  onOptionsDefaultRestore: () => Promise<ConversationOptions | null>;
  onUploadFiles: (files: File[]) => void | Promise<void>;
  onCaptureScreenshot: () => void | Promise<void>;
  onRemoveAttachment: (fileID: string) => void;
  onSendMessage: () => void | Promise<void>;
  onStopMessage: () => void;
};

type ComposerModeIndicator = {
  label: string;
  intro: string;
  description: string;
  icon: React.ComponentType<{ className?: string; strokeWidth?: number }>;
  tone: "default" | "warning";
};

function resolveComposerModeIndicator(
  decision: ChatSubmitDecision,
  t: (key: string) => string,
): ComposerModeIndicator | null {
  if (decision.blockedReason === "image_task_rejects_non_image_attachments") {
    return {
      label: t("mediaMode.invalidFile"),
      intro: t("mediaMode.invalidFileIntro"),
      description: t(`mediaMode.blockedDescriptions.${decision.blockedReason}`),
      icon: ImageOff,
      tone: "warning",
    };
  }
  if (decision.task === "image_generation") {
    return {
      label: t("mediaMode.imageGeneration"),
      intro: t("mediaMode.imageGenerationIntro"),
      description: decision.blockedReason
        ? t(`mediaMode.blockedDescriptions.${decision.blockedReason}`)
        : t("mediaMode.imageGenerationDescription"),
      icon: Image,
      tone: "default",
    };
  }
  if (decision.task === "image_edit") {
    return {
      label: t("mediaMode.imageEdit"),
      intro: t("mediaMode.imageEditIntro"),
      description: decision.blockedReason
        ? t(`mediaMode.blockedDescriptions.${decision.blockedReason}`)
        : t("mediaMode.imageEditDescription"),
      icon: ImagePlus,
      tone: "default",
    };
  }
  return null;
}

function clipboardFilesFromPaste(event: React.ClipboardEvent<HTMLTextAreaElement>): File[] {
  const itemFiles = Array.from(event.clipboardData.items ?? [])
    .filter((item) => item.kind === "file")
    .map((item) => item.getAsFile())
    .filter((file): file is File => file !== null);
  const sourceFiles = itemFiles.length > 0 ? itemFiles : Array.from(event.clipboardData.files ?? []);
  const pastedAt = Date.now();

  return sourceFiles.map((file, index) => {
    if (file.name.trim()) {
      return file;
    }
    const extension = file.type.startsWith("image/") ? ".png" : "";
    const prefix = file.type.startsWith("image/") ? "pasted-image" : "pasted-file";
    return new File([file], `${prefix}-${pastedAt}-${index + 1}${extension}`, {
      type: file.type,
      lastModified: file.lastModified,
    });
  });
}

function readPromptTemplateArray(key: string): unknown[] {
  if (typeof window === "undefined") {
    return [];
  }
  try {
    const raw = window.localStorage.getItem(key);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function readCustomPromptTemplates(): PromptTemplate[] {
  return readPromptTemplateArray(PROMPT_TEMPLATE_STORAGE_KEY)
    .map((item) => {
      if (!item || typeof item !== "object") {
        return null;
      }
      const record = item as Record<string, unknown>;
      if (typeof record.id !== "string" || typeof record.title !== "string" || typeof record.body !== "string") {
        return null;
      }
      return normalizePromptTemplate({
        id: record.id,
        title: record.title,
        body: record.body,
        category: typeof record.category === "string" ? record.category : "custom",
      });
    })
    .filter((item): item is PromptTemplate => Boolean(item?.id && item.title && item.body));
}

function readRecentPromptTemplateIDs(): string[] {
  return readPromptTemplateArray(PROMPT_TEMPLATE_RECENT_STORAGE_KEY)
    .filter((item): item is string => typeof item === "string" && item.trim().length > 0)
    .slice(0, PROMPT_TEMPLATE_RECENT_LIMIT);
}

function writePromptTemplateStorage(key: string, value: unknown) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(key, JSON.stringify(value));
}

function ChatInputComponent({
  draft,
  loading,
  sending,
  uploading,
  fileMode,
  sendShortcut = "enter",
  inputHeight = "standard",
  attachments,
  uploadingAttachments,
  inputHistory = [],
  modelOptions,
  selectedPlatformModelName,
  availableTools,
  selectedToolIDs,
  confirmedToolIDs,
  webSearchEnabled,
  codeSandboxEnabled,
  researchMaxLLMCalls,
  researchMaxToolCalls,
  htmlVisualPromptEnabled,
  maxSelectedTools,
  toolsLoading,
  options,
  defaultOptions,
  modelOptionPolicy,
  modelLoading,
  modelDisabled = false,
  dropActive = false,
  onDraftChange,
  onModelChange,
  onSelectedToolsChange,
  onConfirmedToolsChange,
  onWebSearchEnabledChange,
  onCodeSandboxEnabledChange,
  onResearchMaxLLMCallsChange,
  onResearchMaxToolCallsChange,
  onHTMLVisualPromptChange,
  onOptionsChange,
  onOptionsReset,
  onOptionsDefaultRestore,
  onUploadFiles,
  onCaptureScreenshot,
  onRemoveAttachment,
  onSendMessage,
  onStopMessage,
}: ChatInputProps) {
  const tChat = useTranslations("chat");
  const tComposer = useTranslations("chat.composer");
  const tFileStatus = useTranslations("files.status");
  const [isPlusHovered, setIsPlusHovered] = React.useState(false);
  const [isBlocksHovered, setIsBlocksHovered] = React.useState(false);
  const [isVoiceHovered, setIsVoiceHovered] = React.useState(false);
  const speechInput = useSpeechInput({
    draft,
    idlePlaceholder: tChat("placeholder"),
    listeningPlaceholder: tComposer("listening"),
    onDraftChange,
  });
  const isMobile = useIsMobile();
  const [hoveredTool, setHoveredTool] = React.useState<"upload" | "camera" | "gallery" | "screenshot" | null>(null);
  const [ragWarnDismissed, setRagWarnDismissed] = React.useState(false);
  const [previewAttachment, setPreviewAttachment] = React.useState<PendingAttachment | null>(null);
  const [customPromptTemplates, setCustomPromptTemplates] = React.useState<PromptTemplate[]>([]);
  const [recentPromptTemplateIDs, setRecentPromptTemplateIDs] = React.useState<string[]>([]);
  const fileInputRef = React.useRef<HTMLInputElement | null>(null);
  const cameraInputRef = React.useRef<HTMLInputElement | null>(null);
  const galleryInputRef = React.useRef<HTMLInputElement | null>(null);
  const composingRef = React.useRef(false);
  const historyIndexRef = React.useRef<number | null>(null);
  const hasDraftText = draft.trim().length > 0;
  const hasSendableContent = hasDraftText || attachments.length > 0;
  const canSend = hasSendableContent && !sending && !loading && !uploading;
  const inputHeightClassName =
    inputHeight === "compact" ? "max-h-32" : inputHeight === "loose" ? "max-h-64" : "max-h-44";

  // Only relevant in RAG mode: all document attachments opted out of RAG.
  const docAttachments = attachments.filter((a) => a.fileCategory !== "image");
  const allRagOptOut =
    fileMode === "rag" &&
    docAttachments.length > 0 &&
    docAttachments.every((a) => a.ragOptOut === true);
  const showRagWarn = allRagOptOut && !ragWarnDismissed;

  const closePreviewDialog = React.useCallback((open: boolean) => {
    if (!open) {
      setPreviewAttachment(null);
    }
  }, []);

  React.useEffect(() => {
    setCustomPromptTemplates(readCustomPromptTemplates());
    setRecentPromptTemplateIDs(readRecentPromptTemplateIDs());
  }, []);

  const builtInPromptTemplates = React.useMemo(
    () => [
      normalizePromptTemplate({
        id: "summarize",
        title: tComposer("promptTemplates.defaults.summarize.title"),
        body: tComposer("promptTemplates.defaults.summarize.body"),
        category: tComposer("promptTemplates.categories.work"),
      }),
      normalizePromptTemplate({
        id: "rewrite",
        title: tComposer("promptTemplates.defaults.rewrite.title"),
        body: tComposer("promptTemplates.defaults.rewrite.body"),
        category: tComposer("promptTemplates.categories.writing"),
      }),
      normalizePromptTemplate({
        id: "plan",
        title: tComposer("promptTemplates.defaults.plan.title"),
        body: tComposer("promptTemplates.defaults.plan.body"),
        category: tComposer("promptTemplates.categories.work"),
      }),
      normalizePromptTemplate({
        id: "compare",
        title: tComposer("promptTemplates.defaults.compare.title"),
        body: tComposer("promptTemplates.defaults.compare.body"),
        category: tComposer("promptTemplates.categories.analysis"),
      }),
    ],
    [tComposer],
  );

  const promptTemplateQuery = draft.trimStart().startsWith("/") ? draft.trimStart() : "";
  const promptTemplates = React.useMemo(() => {
    const recentRank = new Map(recentPromptTemplateIDs.map((id, index) => [id, index]));
    return [...customPromptTemplates, ...builtInPromptTemplates].sort((a, b) => {
      const aRank = recentRank.get(a.id) ?? Number.POSITIVE_INFINITY;
      const bRank = recentRank.get(b.id) ?? Number.POSITIVE_INFINITY;
      return aRank - bRank || a.title.localeCompare(b.title);
    });
  }, [builtInPromptTemplates, customPromptTemplates, recentPromptTemplateIDs]);
  const filteredPromptTemplates = React.useMemo(
    () => filterPromptTemplates(promptTemplates, promptTemplateQuery).slice(0, 8),
    [promptTemplateQuery, promptTemplates],
  );
  const showPromptTemplatePanel = promptTemplateQuery.length > 0;

  const selectedModel = React.useMemo(
    () => modelOptions.find((item) => item.platformModelName === selectedPlatformModelName) ?? null,
    [modelOptions, selectedPlatformModelName],
  );
  const selectedProtocol = selectedModel?.protocols[0]?.trim() ?? "";
  const selectedModelName = selectedModel?.platformModelName || selectedPlatformModelName;
  const screenshotSupported = typeof navigator !== "undefined" && Boolean(navigator.mediaDevices?.getDisplayMedia);
  const submitDecision = resolveChatSubmitDecision(selectedModel, attachments);
  const submitTask = submitDecision.task;
  const isMediaMode = isMediaSubmitTask(submitTask);
  const composerModeIndicator = resolveComposerModeIndicator(submitDecision, tComposer);
  const ComposerModeIcon = composerModeIndicator?.icon;
  const modelOptionPolicyDisabled = modelOptionPolicy?.mode?.trim() === "disabled";
  const showMCPToolsButton = !isMediaMode;
  const showHTMLVisualPromptButton = !isMediaMode;
  const onSelectUploadTool = React.useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const onSelectCameraTool = React.useCallback(() => {
    cameraInputRef.current?.click();
  }, []);

  const onSelectGalleryTool = React.useCallback(() => {
    galleryInputRef.current?.click();
  }, []);

  const onSelectScreenshotTool = React.useCallback(() => {
    void onCaptureScreenshot();
  }, [onCaptureScreenshot]);

  const applyPromptTemplate = React.useCallback(
    (template: PromptTemplate) => {
      onDraftChange(buildDraftFromPromptTemplate(draft, template));
      const nextRecentIDs = recordPromptTemplateUsage(recentPromptTemplateIDs, template.id, PROMPT_TEMPLATE_RECENT_LIMIT);
      setRecentPromptTemplateIDs(nextRecentIDs);
      writePromptTemplateStorage(PROMPT_TEMPLATE_RECENT_STORAGE_KEY, nextRecentIDs);
    },
    [draft, onDraftChange, recentPromptTemplateIDs],
  );

  const saveCurrentDraftAsPromptTemplate = React.useCallback(() => {
    const body = draft.replace(/^\/\S*\s*/, "").trim();
    if (!body) {
      toast.error(tComposer("promptTemplates.emptyDraft"));
      return;
    }
    const title = body.split("\n")[0]?.slice(0, 32).trim() || tComposer("promptTemplates.customFallbackTitle");
    const nextTemplate = normalizePromptTemplate({
      id: `custom-${Date.now()}`,
      title,
      body,
      category: tComposer("promptTemplates.categories.mine"),
    });
    const nextTemplates = [nextTemplate, ...customPromptTemplates].slice(0, 30);
    setCustomPromptTemplates(nextTemplates);
    writePromptTemplateStorage(PROMPT_TEMPLATE_STORAGE_KEY, nextTemplates);
    toast.success(tComposer("promptTemplates.saved"));
  }, [customPromptTemplates, draft, tComposer]);

  const deleteCustomPromptTemplate = React.useCallback(
    (templateID: string) => {
      const nextTemplates = customPromptTemplates.filter((template) => template.id !== templateID);
      setCustomPromptTemplates(nextTemplates);
      writePromptTemplateStorage(PROMPT_TEMPLATE_STORAGE_KEY, nextTemplates);
    },
    [customPromptTemplates],
  );

  return (
    <div className="w-full pb-[env(safe-area-inset-bottom)] md:pb-0">
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="sr-only "
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          if (files.length > 0) {
            void onUploadFiles(files);
          }
          event.currentTarget.value = "";
        }}
      />
      <input
        ref={cameraInputRef}
        type="file"
        accept="image/*"
        capture="environment"
        className="sr-only"
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          if (files.length > 0) {
            void onUploadFiles(files);
          }
          event.currentTarget.value = "";
        }}
      />
      <input
        ref={galleryInputRef}
        type="file"
        accept="image/*"
        multiple
        className="sr-only"
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          if (files.length > 0) {
            void onUploadFiles(files);
          }
          event.currentTarget.value = "";
        }}
      />

      <InputGroup
        className={cn(
          "bg-pure rounded-3xl border-[0.5px] border-border/70 shadow-xs has-[[data-slot=input-group-control]:focus-visible]:ring-0 has-[[data-slot=input-group-control]:focus-visible]:border-border",
          dropActive && "border-dashed border-foreground/30 bg-muted/20 shadow-none",
        )}
      >
        {attachments.length > 0 || uploadingAttachments.length > 0 ? (
          <div className="w-full space-y-2 px-2.5 pt-2">
            {showRagWarn ? (
              <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/70 px-3 py-2 text-[11px] text-foreground">
                <ImageOff className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.6} />
                <span className="flex-1">{tComposer("ragAllDisabled")}</span>
                <button
                  type="button"
                  className="relative inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground after:absolute after:-inset-2 after:content-[''] md:after:hidden"
                  onClick={() => setRagWarnDismissed(true)}
                  aria-label={tComposer("closeHint")}
                >
                  <XIcon size={15} strokeWidth={1.8} />
                </button>
              </div>
            ) : null}
            <div className="w-full overflow-hidden sm:overflow-x-auto">
              <div className="flex max-h-[196px] w-full flex-col gap-2 overflow-y-auto pb-1 pl-1.5 pr-2 pt-2 sm:max-h-none sm:w-max sm:flex-row sm:overflow-y-visible sm:pr-1.5">
                {attachments.map((item) => (
                  <div
                    key={item.fileID}
                    className="bg-pure group relative flex h-14 w-full shrink-0 items-center gap-1.5 rounded-lg border border-border/50 bg-background/95 px-2 text-left shadow-xs transition-colors hover:border-border hover:bg-accent/30 sm:w-[228px] sm:px-2.5"
                  >
                    <button
                      type="button"
                      className="relative flex h-full min-w-0 flex-1 items-center gap-2 rounded-md py-1 text-left outline-none transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/35 after:absolute after:-inset-y-1 after:inset-x-0 after:content-['']"
                      onClick={() => setPreviewAttachment(item)}
                      aria-label={tComposer("previewAttachment", { name: item.fileName })}
                    >
                    {(() => {
                      const badge = resolveFileProcessingBadge(item, (key, values) => tFileStatus(key, values));
                      const FileIcon = resolveFileIcon(item);
                      const imagePreview = item.fileCategory === "image" && item.previewURL ? item.previewURL : "";
                      return (
                        <>
                          {imagePreview ? (
                            <div className="size-10 shrink-0 overflow-hidden rounded-md border border-border/60 bg-muted">
                              {/* eslint-disable-next-line @next/next/no-img-element -- Composer previews are local/user-uploaded attachment URLs. */}
                              <img src={imagePreview} alt={item.fileName} loading="lazy" decoding="async" className="size-full object-cover" />
                            </div>
                          ) : (
                            <div className="flex size-6 shrink-0 items-center justify-center">
                              <FileIcon className="size-5 text-muted-foreground" strokeWidth={1.6} />
                            </div>
                          )}
                          <div className="flex min-w-0 flex-1 flex-col justify-center">
                            <p className="truncate text-[12px] font-medium leading-4 text-foreground/90" title={item.fileName}>
                              {item.fileName}
                            </p>
                            <div className="mt-1 flex min-w-0 items-center gap-1.5">
                              <span className="min-w-0 shrink truncate text-[10px] leading-none text-muted-foreground">
                                {formatBytes(item.sizeBytes)}
                              </span>
                              <span
                                className={cn(
                                  "inline-flex max-w-[82px] shrink-0 items-center rounded-md px-1.5 py-0.5 text-[10px] font-medium leading-none",
                                  resolveFileProcessingToneClass(badge.tone),
                                )}
                                title={badge.detail}
                              >
                                <span className="truncate">{badge.label}</span>
                              </span>
                              {item.ragOptOut && item.fileCategory !== "image" ? (
                                <span
                                  className="shrink-0 rounded-md bg-muted/60 px-1.5 py-0.5 text-[10px] font-medium leading-none text-muted-foreground/65"
                                  title={tComposer("ragDisabledTitle")}
                                >
                                  {tComposer("ragOff")}
                                </span>
                              ) : null}
                            </div>
                          </div>
                        </>
                      );
                    })()}
                    </button>
                    <button
                      type="button"
                      className="relative inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/35 after:absolute after:-inset-2 after:content-[''] md:after:hidden"
                      onClick={() => onRemoveAttachment(item.fileID)}
                      aria-label={tComposer("removeAttachment", { name: item.fileName })}
                    >
                      <XIcon size={15} strokeWidth={1.8} animateOnHover="default" />
                    </button>
                  </div>
                ))}
                {uploadingAttachments.map((item) => (
                  <div
                    key={item.tempID}
                    className="bg-pure relative flex h-14 w-full shrink-0 items-center gap-2.5 rounded-lg border border-border/50 bg-background/95 px-2.5 sm:w-[228px]"
                    aria-label={tComposer("uploadingAttachment", { name: item.fileName })}
                  >
                    <Skeleton className="size-5 shrink-0 rounded-sm" />
                    <div className="min-w-0 flex-1 space-y-2">
                      <Skeleton className="h-3 w-[78%]" />
                      <div className="flex items-center gap-1.5">
                        <Skeleton className="h-2.5 w-10" />
                        <Skeleton className="h-4 w-12 rounded-md" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            {previewAttachment ? (
              <FilePreviewDialog
                file={previewAttachment}
                open={previewAttachment !== null}
                onOpenChange={closePreviewDialog}
              />
            ) : null}
          </div>
        ) : null}

        {showPromptTemplatePanel ? (
          <div
            className="mx-2.5 mb-2 rounded-2xl border border-border/70 bg-popover p-2 text-popover-foreground shadow-xs"
            onMouseDown={(event) => event.preventDefault()}
          >
            <div className="mb-1 flex items-center justify-between gap-2 px-1">
              <div className="flex min-w-0 items-center gap-1.5 text-xs font-medium text-muted-foreground">
                <BookOpen className="size-3.5" strokeWidth={1.8} />
                <span>{tComposer("promptTemplates.title")}</span>
              </div>
              <button
                type="button"
                className="inline-flex min-h-9 items-center gap-1 rounded-md px-2 text-[11px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground md:min-h-7"
                onClick={saveCurrentDraftAsPromptTemplate}
              >
                <Save className="size-3.5" strokeWidth={1.7} />
                {tComposer("promptTemplates.save")}
              </button>
            </div>
            <div className="max-h-64 overflow-y-auto">
              {filteredPromptTemplates.length > 0 ? (
                filteredPromptTemplates.map((template) => {
                  const custom = template.id.startsWith("custom-");
                  return (
                    <div key={template.id} className="group/template flex items-start gap-1 rounded-xl hover:bg-accent/60">
                      <button
                        type="button"
                        className="relative flex min-h-9 min-w-0 flex-1 flex-col rounded-xl px-2.5 py-1.5 text-left outline-none transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/35 after:absolute after:-inset-y-1 after:inset-x-0 after:content-['']"
                        onClick={() => applyPromptTemplate(template)}
                      >
                        <span className="flex w-full min-w-0 items-center gap-2">
                          <span className="truncate text-xs font-medium text-foreground">{template.title}</span>
                          <span className="shrink-0 rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                            {template.category}
                          </span>
                        </span>
                        <span className="mt-0.5 line-clamp-2 text-[11px] leading-4 text-muted-foreground">
                          {template.body}
                        </span>
                      </button>
                      {custom ? (
                        <button
                          type="button"
                          className="mr-1 mt-1 inline-flex size-9 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-destructive md:size-7"
                          aria-label={tComposer("promptTemplates.delete", { title: template.title })}
                          onClick={() => deleteCustomPromptTemplate(template.id)}
                        >
                          <Trash2 className="size-3.5" strokeWidth={1.8} />
                        </button>
                      ) : null}
                    </div>
                  );
                })
              ) : (
                <div className="px-3 py-4 text-center text-xs text-muted-foreground">
                  {tComposer("promptTemplates.empty")}
                </div>
              )}
            </div>
          </div>
        ) : null}

        <InputGroupTextarea
          value={draft}
          disabled={sending || loading || uploading}
          readOnly={speechInput.active}
          placeholder={dropActive ? tChat("attachments.dropTitle") : speechInput.placeholder}
          rows={1}
          style={{ fontFamily: "var(--font-chat)", fontWeight: "var(--font-chat-weight)" }}
          className={cn(
            "rounded-3xl min-h-12 overflow-y-auto px-5 pt-4 text-[15px] leading-6 placeholder:text-muted-foreground placeholder:font-[inherit] placeholder:leading-[inherit]",
            inputHeightClassName,
            speechInput.active ? "placeholder:font-normal placeholder:text-muted-foreground" : "",
          )}
          onChange={(event) => onDraftChange(event.target.value)}
          onPaste={(event) => {
            const files = clipboardFilesFromPaste(event);
            if (files.length === 0) {
              const pastedText = event.clipboardData.getData("text/plain");
              if (shouldConvertPasteToTextFile(pastedText)) {
                event.preventDefault();
                void onUploadFiles([createPastedTextFile(pastedText)]);
                toast.success(tComposer("longPasteAttached"));
              }
              return;
            }
            if (!event.clipboardData.getData("text/plain")) {
              event.preventDefault();
            }
            void onUploadFiles(files);
          }}
          onCompositionStart={() => {
            composingRef.current = true;
          }}
          onCompositionEnd={() => {
            composingRef.current = false;
          }}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing || composingRef.current || event.key === "Process" || event.keyCode === 229) {
              return;
            }
            if (
              (event.key === "ArrowUp" || event.key === "ArrowDown") &&
              inputHistory.length > 0 &&
              event.currentTarget.selectionStart === event.currentTarget.selectionEnd
            ) {
              const atStart = event.currentTarget.selectionStart === 0;
              const atEnd = event.currentTarget.selectionStart === event.currentTarget.value.length;
              const currentIndex = historyIndexRef.current;
              if (event.key === "ArrowUp" && atStart && (!draft.trim() || currentIndex !== null)) {
                event.preventDefault();
                const nextIndex = currentIndex === null ? inputHistory.length - 1 : Math.max(0, currentIndex - 1);
                historyIndexRef.current = nextIndex;
                onDraftChange(inputHistory[nextIndex] ?? "");
                return;
              }
              if (event.key === "ArrowDown" && atEnd && currentIndex !== null) {
                event.preventDefault();
                const nextIndex = currentIndex + 1;
                if (nextIndex >= inputHistory.length) {
                  historyIndexRef.current = null;
                  onDraftChange("");
                } else {
                  historyIndexRef.current = nextIndex;
                  onDraftChange(inputHistory[nextIndex] ?? "");
                }
                return;
              }
            } else if (event.key !== "ArrowUp" && event.key !== "ArrowDown") {
              historyIndexRef.current = null;
            }
            const shouldSend = isSendShortcutEvent(sendShortcut, event);

            if (shouldSend) {
              event.preventDefault();
              if (canSend) {
                void onSendMessage();
              }
            }
          }}
        />

        <InputGroupAddon align="block-end" className="items-center justify-between pt-2">
          <div className="flex shrink-0 items-center gap-0.5 sm:gap-1">
            <DropdownMenu modal={false}>
              <DropdownMenuTrigger asChild>
                <InputGroupButton
                  id="chat-tools-menu-trigger"
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="relative size-8 rounded-md text-muted-foreground hover:text-foreground after:absolute after:-inset-1.5 after:content-[''] md:after:hidden"
                  disabled={sending || loading || uploading}
                  aria-label={tComposer("openTools")}
                  onMouseEnter={() => setIsPlusHovered(true)}
                  onMouseLeave={() => setIsPlusHovered(false)}
                >
                  <Plus
                    size={20}
                    strokeWidth={1.4}
                    animate={isPlusHovered ? "default" : undefined}
                  />
                </InputGroupButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent side="bottom" align="start" sideOffset={8} className="w-36">
                {isMobile ? (
                  <>
                    <DropdownMenuItem
                      onMouseEnter={() => setHoveredTool("camera")}
                      onMouseLeave={() => setHoveredTool((prev) => (prev === "camera" ? null : prev))}
                      onSelect={(event) => {
                        event.preventDefault();
                        onSelectCameraTool();
                      }}
                    >
                      <Camera size={12} strokeWidth={1.5} />
                      {tComposer("takePhoto")}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onMouseEnter={() => setHoveredTool("gallery")}
                      onMouseLeave={() => setHoveredTool((prev) => (prev === "gallery" ? null : prev))}
                      onSelect={(event) => {
                        event.preventDefault();
                        onSelectGalleryTool();
                      }}
                    >
                      <ImagePlus size={12} strokeWidth={1.5} />
                      {tComposer("chooseFromGallery")}
                    </DropdownMenuItem>
                  </>
                ) : null}
                <DropdownMenuItem
                  onMouseEnter={() => setHoveredTool("upload")}
                  onMouseLeave={() => setHoveredTool((prev) => (prev === "upload" ? null : prev))}
                  onSelect={(event) => {
                    event.preventDefault();
                    onSelectUploadTool();
                  }}
                >
                  <LinkIcon size={12} strokeWidth={1.5} animate={hoveredTool === "upload" ? "default" : undefined} />
                  {tComposer("uploadFile")}
                </DropdownMenuItem>
                {screenshotSupported && !isMobile ? (
                  <DropdownMenuItem
                    onMouseEnter={() => setHoveredTool("screenshot")}
                    onMouseLeave={() => setHoveredTool((prev) => (prev === "screenshot" ? null : prev))}
                    onSelect={(event) => {
                      event.preventDefault();
                      onSelectScreenshotTool();
                    }}
                  >
                    <Crop size={12} strokeWidth={1.5} animate={hoveredTool === "screenshot" ? "default" : undefined} />
                    {tComposer("screenshot")}
                  </DropdownMenuItem>
                ) : null}
              </DropdownMenuContent>
            </DropdownMenu>

            {!modelOptionPolicyDisabled ? (
              <ChatModelConfig
                disabled={sending || loading || uploading || modelLoading}
                options={options}
                defaultOptions={defaultOptions}
                optionControls={selectedModel?.optionControls ?? []}
                nativeToolKeys={selectedModel?.nativeToolKeys ?? []}
                nativeTools={selectedModel?.nativeTools ?? []}
                modelOptionPolicy={modelOptionPolicy}
                selectedProtocol={selectedProtocol}
                selectedModelName={selectedModelName}
                onOptionsChange={onOptionsChange}
                onOptionsReset={onOptionsReset}
                onDefaultOptionsRestore={onOptionsDefaultRestore}
              />
            ) : null}

            {showMCPToolsButton ? (
              <ChatMCP
                availableTools={availableTools}
                selectedToolIDs={selectedToolIDs}
                confirmedToolIDs={confirmedToolIDs}
                webSearchEnabled={webSearchEnabled}
                codeSandboxEnabled={codeSandboxEnabled}
                researchMaxLLMCalls={researchMaxLLMCalls}
                researchMaxToolCalls={researchMaxToolCalls}
                maxSelectedTools={maxSelectedTools}
                disabled={sending || loading || uploading || toolsLoading}
                onSelectedToolsChange={onSelectedToolsChange}
                onConfirmedToolsChange={onConfirmedToolsChange}
                onWebSearchEnabledChange={onWebSearchEnabledChange}
                onCodeSandboxEnabledChange={onCodeSandboxEnabledChange}
                onResearchMaxLLMCallsChange={onResearchMaxLLMCallsChange}
                onResearchMaxToolCallsChange={onResearchMaxToolCallsChange}
              />
            ) : null}

            {showHTMLVisualPromptButton ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <InputGroupButton
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className={cn(
                      "relative size-8 rounded-md text-muted-foreground hover:text-foreground after:absolute after:-inset-1.5 after:content-[''] md:after:hidden",
                      htmlVisualPromptEnabled && "bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
                    )}
                    disabled={sending || loading || uploading}
                    aria-label={tComposer("htmlVisualPrompt")}
                    aria-pressed={htmlVisualPromptEnabled}
                    onClick={() => onHTMLVisualPromptChange(!htmlVisualPromptEnabled)}
                    onMouseEnter={() => setIsBlocksHovered(true)}
                    onMouseLeave={() => setIsBlocksHovered(false)}
                  >
                    <Blocks
                      size={20}
                      strokeWidth={1.4}
                      animate={htmlVisualPromptEnabled ? "default" : isBlocksHovered ? "default" : undefined}
                    />
                  </InputGroupButton>
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-72 text-xs leading-5">
                  {htmlVisualPromptEnabled
                    ? tComposer("htmlVisualPromptEnabled")
                    : tComposer("htmlVisualPromptDisabled")}
                </TooltipContent>
              </Tooltip>
            ) : null}
          </div>

          <div className="flex min-w-0 flex-1 items-center justify-end gap-1 overflow-hidden sm:gap-1.5">
            {composerModeIndicator && ComposerModeIcon ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span
                    className={cn(
                      "inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg px-2 text-[11px] font-medium transition-colors",
                      composerModeIndicator.tone === "warning"
                        ? "bg-destructive/10 text-destructive"
                        : "bg-muted/60 text-muted-foreground",
                    )}
                  >
                    <ComposerModeIcon className="size-3.5" strokeWidth={1.7} />
                    <span className="hidden sm:inline">{composerModeIndicator.label}</span>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top" align="end" className="max-w-72 text-xs leading-5">
                  {composerModeIndicator.intro} {composerModeIndicator.description}
                </TooltipContent>
              </Tooltip>
            ) : null}
            <ChatModelPicker
              modelOptions={modelOptions}
              selectedPlatformModelName={selectedPlatformModelName}
              loading={modelLoading}
              disabled={modelDisabled}
              onModelChange={onModelChange}
            />

            <InputGroupButton
              type="button"
              variant="ghost"
              size="icon-sm"
              className="relative size-8 rounded-md text-muted-foreground hover:text-foreground after:absolute after:-inset-1.5 after:content-[''] md:after:hidden"
              disabled={loading || uploading || (!sending && !hasSendableContent && !speechInput.supported)}
              onClick={sending ? onStopMessage : hasSendableContent ? onSendMessage : speechInput.toggle}
              onMouseEnter={() => setIsVoiceHovered(true)}
              onMouseLeave={() => setIsVoiceHovered(false)}
              aria-label={sending ? tComposer("pauseGeneration") : hasSendableContent ? tChat("send") : speechInput.active ? tComposer("cancelVoiceInput") : tComposer("voiceInput")}
              title={sending ? tComposer("pauseGeneration") : hasSendableContent ? tChat("send") : speechInput.supported ? (speechInput.active ? tComposer("cancelVoiceInput") : tComposer("voiceInput")) : tComposer("voiceUnsupported")}
            >
              {sending ? (
                <Pause
                  size={20}
                  strokeWidth={1.4}
                  animate="default-loop"
                />
              ) : speechInput.active ? (
                <AudioLines
                  size={20}
                  strokeWidth={1.4}
                  animate="default"
                />
              ) : hasSendableContent ? (
                <Send
                  size={20}
                  strokeWidth={1.4}
                  animate={isVoiceHovered ? "default" : undefined}
                />
              ) : (
                <AudioLines
                  size={20}
                  strokeWidth={1.4}
                  animate={isVoiceHovered ? "default" : undefined}
                />
              )}
            </InputGroupButton>
          </div>
        </InputGroupAddon>
      </InputGroup>
    </div>
  );
}

export const ChatInput = React.memo(ChatInputComponent);
ChatInput.displayName = "ChatInput";
