"use client";

import * as React from "react";
import { Check, ChevronDown, Globe2, Info, SlidersHorizontal, Terminal } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Unplug } from "@/components/animate-ui/icons/unplug";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { InputGroupButton } from "@/components/ui/input-group";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { MCPToolDTO } from "@/shared/api/mcp.types";
import { useIsMobile } from "@/shared/hooks/use-mobile";

const DEFAULT_MCP_TOOL_SELECTION_LIMIT = 32;
const MAX_MCP_TOOL_SELECTION_LIMIT = 128;

type MCPToolGroup = {
  key: string;
  serverName: string;
  tools: MCPToolDTO[];
};

type FilteredMCPToolGroup = MCPToolGroup & {
  visibleTools: MCPToolDTO[];
};

type ChatMCPProps = {
  availableTools: MCPToolDTO[];
  selectedToolIDs: number[];
  confirmedToolIDs: number[];
  webSearchEnabled: boolean;
  codeSandboxEnabled: boolean;
  researchMaxLLMCalls: number;
  researchMaxToolCalls: number;
  maxSelectedTools: number;
  disabled: boolean;
  onSelectedToolsChange: (toolIDs: number[]) => void;
  onConfirmedToolsChange: (toolIDs: number[]) => void;
  onWebSearchEnabledChange: (enabled: boolean) => void;
  onCodeSandboxEnabledChange: (enabled: boolean) => void;
  onResearchMaxLLMCallsChange: (value: number) => void;
  onResearchMaxToolCallsChange: (value: number) => void;
};

function resolveMCPToolLabel(tool: MCPToolDTO, fallback: string): string {
  return tool.displayName.trim() || tool.name.trim() || fallback;
}

function resolveMCPToolServerName(tool: MCPToolDTO): string {
  return tool.serverName?.trim() ?? "";
}

function resolveMCPToolServerKey(tool: MCPToolDTO): string {
  if (Number.isFinite(tool.serverID) && tool.serverID > 0) {
    return `server:${tool.serverID}`;
  }
  const serverName = resolveMCPToolServerName(tool);
  return serverName ? `server-name:${serverName}` : "server:unknown";
}

function buildMCPToolGroups(tools: MCPToolDTO[], fallbackServerName: string): MCPToolGroup[] {
  const groups = new Map<string, MCPToolGroup>();
  for (const tool of tools) {
    const key = resolveMCPToolServerKey(tool);
    const serverName = resolveMCPToolServerName(tool) || fallbackServerName;
    const existing = groups.get(key);
    if (existing) {
      existing.tools.push(tool);
      continue;
    }
    groups.set(key, { key, serverName, tools: [tool] });
  }
  return [...groups.values()];
}

function matchesMCPToolSearch(tool: MCPToolDTO, query: string): boolean {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  if (!normalizedQuery) {
    return true;
  }
  return [
    resolveMCPToolLabel(tool, String(tool.id)),
    resolveMCPToolServerName(tool),
    tool.name,
    tool.description,
  ]
    .join(" ")
    .toLocaleLowerCase()
    .includes(normalizedQuery);
}

function matchesMCPServerSearch(group: MCPToolGroup, query: string): boolean {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  if (!normalizedQuery) {
    return true;
  }
  return group.serverName.toLocaleLowerCase().includes(normalizedQuery);
}

function filterMCPToolGroups(groups: MCPToolGroup[], query: string): FilteredMCPToolGroup[] {
  const normalizedQuery = query.trim();
  if (!normalizedQuery) {
    return groups.map((group) => ({ ...group, visibleTools: group.tools }));
  }
  return groups.flatMap((group) => {
    if (matchesMCPServerSearch(group, normalizedQuery)) {
      return [{ ...group, visibleTools: group.tools }];
    }
    const visibleTools = group.tools.filter((tool) => matchesMCPToolSearch(tool, normalizedQuery));
    return visibleTools.length > 0 ? [{ ...group, visibleTools }] : [];
  });
}

function resolveToolSelectionLimit(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return DEFAULT_MCP_TOOL_SELECTION_LIMIT;
  }
  return Math.min(Math.floor(value), MAX_MCP_TOOL_SELECTION_LIMIT);
}

export function ChatMCP({
  availableTools,
  selectedToolIDs,
  confirmedToolIDs,
  webSearchEnabled,
  codeSandboxEnabled,
  researchMaxLLMCalls,
  researchMaxToolCalls,
  maxSelectedTools,
  disabled,
  onSelectedToolsChange,
  onConfirmedToolsChange,
  onWebSearchEnabledChange,
  onCodeSandboxEnabledChange,
  onResearchMaxLLMCallsChange,
  onResearchMaxToolCallsChange,
}: ChatMCPProps) {
  const tComposer = useTranslations("chat.composer");
  const isMobile = useIsMobile();
  const [hovered, setHovered] = React.useState(false);
  const [open, setOpen] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const [expandedServerKeys, setExpandedServerKeys] = React.useState<Set<string>>(() => new Set());
  const selectedToolIDSet = React.useMemo(() => new Set(selectedToolIDs), [selectedToolIDs]);
  const confirmedToolIDSet = React.useMemo(() => new Set(confirmedToolIDs), [confirmedToolIDs]);
  const selectedToolCount = selectedToolIDs.length + (webSearchEnabled ? 1 : 0) + (codeSandboxEnabled ? 1 : 0);
  const selectionLimit = resolveToolSelectionLimit(maxSelectedTools);
  const toolGroups = React.useMemo(
    () => buildMCPToolGroups(availableTools, tComposer("mcpUnknownServer")),
    [availableTools, tComposer],
  );
  const filteredToolGroups = React.useMemo(
    () => filterMCPToolGroups(toolGroups, search),
    [toolGroups, search],
  );
  const hasSearch = search.trim().length > 0;

  const showToolLimitToast = React.useCallback(() => {
    toast.error(tComposer("mcpToolLimitTitle"), {
      description: tComposer("mcpToolLimitDescription", { limit: selectionLimit }),
    });
  }, [selectionLimit, tComposer]);

  const toggleTool = React.useCallback(
    (toolID: number, checked: boolean) => {
      if (checked) {
        if (selectedToolIDSet.has(toolID)) {
          return;
        }
        if (selectedToolIDs.length >= selectionLimit) {
          showToolLimitToast();
          return;
        }
        onSelectedToolsChange([...selectedToolIDs, toolID]);
        return;
      }
      onSelectedToolsChange(selectedToolIDs.filter((item) => item !== toolID));
    },
    [onSelectedToolsChange, selectedToolIDs, selectedToolIDSet, selectionLimit, showToolLimitToast],
  );

  const toggleToolGroup = React.useCallback(
    (tools: MCPToolDTO[], checked: boolean) => {
      const toolIDs = tools.map((tool) => tool.id);
      if (!checked) {
        const removeSet = new Set(toolIDs);
        onSelectedToolsChange(selectedToolIDs.filter((id) => !removeSet.has(id)));
        return;
      }
      const selectedSet = new Set(selectedToolIDs);
      const missingIDs = toolIDs.filter((id) => !selectedSet.has(id));
      if (selectedSet.size + missingIDs.length > selectionLimit) {
        showToolLimitToast();
        return;
      }
      onSelectedToolsChange([...selectedToolIDs, ...missingIDs]);
    },
    [onSelectedToolsChange, selectedToolIDs, selectionLimit, showToolLimitToast],
  );

  const toggleToolConfirmation = React.useCallback(
    (toolID: number, checked: boolean) => {
      if (checked) {
        if (confirmedToolIDSet.has(toolID)) {
          return;
        }
        onConfirmedToolsChange([...confirmedToolIDs, toolID]);
        return;
      }
      onConfirmedToolsChange(confirmedToolIDs.filter((item) => item !== toolID));
    },
    [confirmedToolIDs, confirmedToolIDSet, onConfirmedToolsChange],
  );

  const onResearchLimitChange = React.useCallback((kind: "llm" | "tool", rawValue: string) => {
    const parsed = Number.parseInt(rawValue, 10);
    const value = Number.isFinite(parsed) ? Math.max(0, parsed) : 0;
    if (kind === "llm") {
      onResearchMaxLLMCallsChange(value);
      return;
    }
    onResearchMaxToolCallsChange(value);
  }, [onResearchMaxLLMCallsChange, onResearchMaxToolCallsChange]);

  const toggleServerExpanded = React.useCallback((serverKey: string) => {
    setExpandedServerKeys((current) => {
      const next = new Set(current);
      if (next.has(serverKey)) {
        next.delete(serverKey);
      } else {
        next.add(serverKey);
      }
      return next;
    });
  }, []);

  const toolSelectionState = React.useCallback(
    (tools: MCPToolDTO[]) => {
      const selectedCount = tools.filter((tool) => selectedToolIDSet.has(tool.id)).length;
      return {
        selectedCount,
        allSelected: tools.length > 0 && selectedCount === tools.length,
        partiallySelected: selectedCount > 0 && selectedCount < tools.length,
      };
    },
    [selectedToolIDSet],
  );

  const trigger = (
    <InputGroupButton
      type="button"
      variant="ghost"
      size="icon-sm"
      className="relative size-8 rounded-md text-muted-foreground hover:text-foreground after:absolute after:-inset-1.5 after:content-[''] md:after:hidden"
      disabled={disabled}
      aria-label={tComposer("mcpTools")}
      title={selectedToolCount > 0 ? tComposer("mcpToolsSelected", { count: selectedToolCount }) : tComposer("mcpTools")}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Unplug
        size={20}
        strokeWidth={1.4}
        animate={hovered ? "default" : undefined}
      />
      {selectedToolCount > 0 ? (
        <span className="absolute -right-0.5 -top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-primary px-1 text-[9px] font-medium leading-none text-primary-foreground">
          {selectedToolCount}
        </span>
      ) : null}
    </InputGroupButton>
  );

  const content = (
    <>
      <div className="flex items-center justify-between gap-3 px-2 pb-1.5 text-[11px] font-medium">
        <span>{tComposer("mcpTools")}</span>
        {selectedToolCount > 0 ? (
          <button
            type="button"
            className="text-[11px] text-muted-foreground transition-colors hover:text-foreground"
            onClick={() => {
              onSelectedToolsChange([]);
              onConfirmedToolsChange([]);
              onWebSearchEnabledChange(false);
              onCodeSandboxEnabledChange(false);
            }}
          >
            {tComposer("clear")}
          </button>
        ) : null}
      </div>
      <div
        className="px-0.5 py-1"
        onPointerDown={(event) => event.stopPropagation()}
        onMouseDown={(event) => event.stopPropagation()}
        onClick={(event) => event.stopPropagation()}
      >
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          onKeyDown={(event) => event.stopPropagation()}
          className="bg-background"
          placeholder={tComposer("searchToolsPlaceholder")}
        />
      </div>
      <div className="grid gap-1.5 px-0.5 py-1.5">
        <button
          type="button"
          className={cn(
            "flex min-h-9 items-center gap-2 rounded-md px-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
            webSearchEnabled && "bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
          )}
          onClick={() => onWebSearchEnabledChange(!webSearchEnabled)}
          aria-pressed={webSearchEnabled}
        >
          <Globe2 className="size-3.5 shrink-0" strokeWidth={1.8} />
          <span className="min-w-0 flex-1 truncate">{tComposer("webSearch")}</span>
          <span className="text-[10px] text-current">{webSearchEnabled ? tComposer("enabled") : tComposer("disabled")}</span>
        </button>
        <button
          type="button"
          className={cn(
            "flex min-h-9 items-center gap-2 rounded-md px-2 text-left text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
            codeSandboxEnabled && "bg-primary/10 text-primary hover:bg-primary/10 hover:text-primary",
          )}
          onClick={() => onCodeSandboxEnabledChange(!codeSandboxEnabled)}
          aria-pressed={codeSandboxEnabled}
        >
          <Terminal className="size-3.5 shrink-0" strokeWidth={1.8} />
          <span className="min-w-0 flex-1 truncate">{tComposer("codeRun")}</span>
          <span className="text-[10px] text-current">{codeSandboxEnabled ? tComposer("enabled") : tComposer("disabled")}</span>
        </button>
        <div className="rounded-md border border-border/70 bg-muted/20 p-2">
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
            <SlidersHorizontal className="size-3.5" strokeWidth={1.8} />
            <span>{tComposer("researchLimits")}</span>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1 text-[10px] font-medium text-muted-foreground">
              <span>{tComposer("researchModelCalls")}</span>
              <Input
                type="number"
                min={0}
                max={32}
                inputMode="numeric"
                value={researchMaxLLMCalls || ""}
                onChange={(event) => onResearchLimitChange("llm", event.target.value)}
                onKeyDown={(event) => event.stopPropagation()}
                className="h-8 bg-background text-xs"
                placeholder={tComposer("researchDefault")}
              />
            </label>
            <label className="grid gap-1 text-[10px] font-medium text-muted-foreground">
              <span>{tComposer("researchToolCalls")}</span>
              <Input
                type="number"
                min={0}
                max={64}
                inputMode="numeric"
                value={researchMaxToolCalls || ""}
                onChange={(event) => onResearchLimitChange("tool", event.target.value)}
                onKeyDown={(event) => event.stopPropagation()}
                className="h-8 bg-background text-xs"
                placeholder={tComposer("researchDefault")}
              />
            </label>
          </div>
        </div>
      </div>
      <div className="max-h-[min(54dvh,18rem)] overflow-y-auto px-0.5 pt-1 md:max-h-72">
        {filteredToolGroups.map((group) => {
          const groupState = toolSelectionState(group.tools);
          const expanded = hasSearch || expandedServerKeys.has(group.key);
          const overLimit = group.tools.length > selectionLimit;
          return (
            <div key={group.key} className="mb-1">
              <div
                data-selected={groupState.selectedCount > 0}
                className="group/server flex min-h-9 items-center gap-1 rounded-md text-[11px] font-medium text-muted-foreground data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
              >
                <button
                  type="button"
                  className="relative flex min-h-9 min-w-0 flex-1 items-center gap-2 rounded-md px-1.5 text-left outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground after:absolute after:-inset-y-1 after:inset-x-0 after:content-[''] md:after:hidden"
                  onClick={() => toggleServerExpanded(group.key)}
                >
                  <ChevronDown
                    className={cn("size-3.5 shrink-0 transition-transform duration-200", expanded && "rotate-180")}
                    strokeWidth={1.7}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-xs text-current">{group.serverName}</span>
                    <span className="mt-0.5 flex min-w-0 items-center gap-1.5 text-[10px] leading-none text-muted-foreground transition-colors group-data-[selected=true]/server:text-current">
                      <span className="shrink-0">
                        {tComposer("mcpServerToolCount", { selected: groupState.selectedCount, total: group.tools.length })}
                      </span>
                      {overLimit ? (
                        <span className="min-w-0 truncate text-primary">
                          {tComposer("mcpServerLimitHint", { limit: selectionLimit })}
                        </span>
                      ) : null}
                    </span>
                  </span>
                </button>
                <button
                  type="button"
                  className="relative h-7 shrink-0 rounded-md px-1.5 text-[11px] text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground group-data-[selected=true]/server:text-current after:absolute after:-inset-y-1.5 after:inset-x-0 after:content-[''] md:after:hidden"
                  onClick={() => toggleToolGroup(group.tools, !groupState.allSelected)}
                >
                  {groupState.allSelected ? tComposer("mcpClearServerTools") : tComposer("mcpSelectServerTools")}
                </button>
                <Checkbox
                  checked={groupState.allSelected ? true : groupState.partiallySelected ? "indeterminate" : false}
                  className="mr-1"
                  aria-label={tComposer("mcpToggleServerTools", { server: group.serverName })}
                  onCheckedChange={(nextChecked) => toggleToolGroup(group.tools, nextChecked === true)}
                />
              </div>
              <AnimatePresence initial={false}>
                {expanded ? (
                  <motion.div
                    key={`${group.key}-tools`}
                    initial={{ height: 0, opacity: 0, y: -4 }}
                    animate={{ height: "auto", opacity: 1, y: 0 }}
                    exit={{ height: 0, opacity: 0, y: -4 }}
                    transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
                    className="overflow-hidden"
                  >
                    <div className="ml-4 mt-1 space-y-1 pl-2">
                      {group.visibleTools.map((tool) => {
                        const checked = selectedToolIDSet.has(tool.id);
                        const confirmationRequired = Boolean(tool.requiresConfirmation);
                        const confirmed = confirmedToolIDSet.has(tool.id);
                        const label = resolveMCPToolLabel(tool, tComposer("tool", { id: tool.id }));
                        const description = (tool.description ?? "").trim() || tComposer("noToolDescription");
                        return (
                          <div
                            key={tool.id}
                            data-selected={checked}
                            className="group/tool flex h-7 items-center justify-between rounded-md text-[11px] font-medium text-muted-foreground data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground"
                          >
                            <button
                              type="button"
                              className="relative flex h-full min-w-0 flex-1 items-center gap-1.5 rounded-md px-1.5 text-left outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground after:absolute after:-inset-y-1 after:inset-x-0 after:content-[''] md:after:hidden"
                              onClick={() => toggleTool(tool.id, !checked)}
                            >
                              <span className="min-w-0 truncate text-xs text-current">{label}</span>
                              {confirmationRequired ? (
                                <span className="shrink-0 rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground group-data-[selected=true]/tool:text-current">
                                  {confirmed ? tComposer("toolConfirmed") : tComposer("toolNeedsConfirmation")}
                                </span>
                              ) : null}
                            </button>
                            {confirmationRequired && checked ? (
                              <Checkbox
                                checked={confirmed}
                                className="mx-1"
                                aria-label={tComposer("confirmToolExecution", { tool: label })}
                                onCheckedChange={(nextChecked) => toggleToolConfirmation(tool.id, nextChecked === true)}
                              />
                            ) : null}
                            <span className="flex size-3 shrink-0 items-center justify-center text-current">
                              {checked ? <Check className="size-3 text-current" strokeWidth={1.7} /> : null}
                            </span>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  aria-label={tComposer("viewToolDescription")}
                                  className="relative ml-1 flex size-6 shrink-0 items-center justify-center rounded-md text-current outline-none transition-colors after:absolute after:-inset-2 after:content-[''] hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground md:after:hidden"
                                >
                                  <Info className="size-3.5" strokeWidth={1.8} />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent
                                side="right"
                                align="center"
                                sideOffset={8}
                                className="max-w-72 whitespace-normal text-left text-xs leading-5 [text-wrap:auto]"
                              >
                                {description}
                              </TooltipContent>
                            </Tooltip>
                          </div>
                        );
                      })}
                    </div>
                  </motion.div>
                ) : null}
              </AnimatePresence>
            </div>
          );
        })}
        {filteredToolGroups.length === 0 ? (
          <div className="px-2 py-6 text-center text-xs text-muted-foreground">
            {tComposer("noMatchingTools")}
          </div>
        ) : null}
      </div>
    </>
  );

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={setOpen} direction="bottom">
        <DrawerTrigger asChild>
          {trigger}
        </DrawerTrigger>
        <DrawerContent className="flex max-h-[min(58dvh,24rem)] flex-col overflow-hidden rounded-t-xl border-border bg-popover text-popover-foreground">
          <DrawerHeader className="px-3 pb-1.5 pt-2.5 text-left">
            <DrawerTitle className="text-sm">{tComposer("mcpTools")}</DrawerTitle>
          </DrawerHeader>
          <div className="min-h-0 flex-1 overflow-hidden px-2 pb-[calc(env(safe-area-inset-bottom)+0.5rem)]">
            {content}
          </div>
        </DrawerContent>
      </Drawer>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {trigger}
      </PopoverTrigger>
      <PopoverContent
        side="bottom"
        align="start"
        sideOffset={8}
        data-mcp-tools-popover-content
        className="w-[22rem] p-1.5"
        onPointerDownOutside={(event) => {
          const target = event.target as HTMLElement | null;
          if (target?.closest("[data-mcp-tools-popover-content]")) {
            event.preventDefault();
          }
        }}
        onFocusOutside={(event) => {
          const target = event.target as HTMLElement | null;
          if (target?.closest("[data-mcp-tools-popover-content]")) {
            event.preventDefault();
          }
        }}
      >
        {content}
      </PopoverContent>
    </Popover>
  );
}
