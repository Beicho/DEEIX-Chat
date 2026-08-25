"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown, ChevronLeft, ChevronRight, CircleDollarSign, Search, Clock3, TicketSlash } from "lucide-react";
import { useTranslations } from "next-intl";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "@/components/ui/drawer";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Input } from "@/components/ui/input";
import { InputGroupButton } from "@/components/ui/input-group";
import type { ChatModelOption } from "@/features/chat/types/chat-runtime";
import {
  resolveDesktopMenuListMaxHeight,
  resolveDesktopModelMenuListMaxHeight,
} from "./chat-model-picker-layout";
import { useIsMobile } from "@/shared/hooks/use-mobile";
import { ModelIcon } from "@/shared/components/model-icon";
import {
  cacheWritePricingLabel,
  cacheWritePricingNote,
  formatBillingDisplayUnitPriceFromUSD,
  resolveCacheWritePricingUSD,
} from "@/shared/lib/billing-display";
import type { BillingDisplayCurrency, BillingDisplayLabels, BillingDisplayOptions } from "@/shared/lib/billing-display";
import { resolveModelIconURL, resolveModelIdentity } from "@/shared/lib/model-identity";
import { resolveModelPresentationGroup } from "@/shared/lib/model-presentation";
import { cn } from "@/lib/utils";
import {
  filterModelPickerGroups,
  MODEL_PICKER_RECENT_LIMIT,
  recordRecentModelSelection,
  readRecentModelNames,
  resolveRecentModelOptions,
  writeRecentModelNames,
} from "@/features/chat/model/model-picker-utils";

const MODEL_MENU_MAX_HEIGHT = 400;
const MODEL_MENU_VENDOR_ROW_HEIGHT = 28;
const MODEL_MENU_MODEL_ROW_HEIGHT = 28;
const MODEL_MENU_TOUCH_ROW_HEIGHT = 32;
const MODEL_MENU_ROW_GAP = 2;
const MODEL_MENU_LIST_PADDING_BOTTOM = 4;
const MODEL_MENU_MODEL_PANEL_CHROME_HEIGHT = 12;
const MODEL_MENU_TEXT_WIDTH_UNIT = 7;
const MODEL_MENU_CONTENT_GAP_WIDTH = 56;
const MODEL_MENU_VIEWPORT_GUTTER = 24;
const MODEL_MENU_PANEL_GAP = 8;
const MODEL_MENU_COLLISION_GUTTER = 12;
const MODEL_MENU_SCROLL_MORE_THRESHOLD = 8;
const PRICING_TOOLTIP_TITLE_CLASS = "font-sans text-xs font-medium leading-4 text-background";
const PRICING_TOOLTIP_BODY_CLASS = "font-sans text-[11px] leading-4 text-background/80";

type FloatingModelPanelLayout = {
  key: number;
  x: number;
  y: number;
  width: number;
  listMaxHeight: number;
};

type ChatModelPickerProps = {
  modelOptions: ChatModelOption[];
  billingDisplayCurrency: BillingDisplayCurrency;
  billingDisplayUsdToCnyRate: number | null;
  selectedPlatformModelName: string;
  loading: boolean;
  disabled: boolean;
  onModelCatalogRefresh?: () => void | Promise<void>;
  onModelChange: (platformModelName: string) => void;
};

const MODEL_MENU_COLLISION_PADDING = 24;
const DESKTOP_MODEL_MENU_WIDTH = 224;
const DESKTOP_MODEL_SUBMENU_GAP = 8;
const DESKTOP_MODEL_MENU_SIDE_OFFSET = 8;
const DESKTOP_MODEL_MENU_MIN_SCROLL_HEIGHT = 96;
/** Group panel non-list chrome: p-1.5 × 2 + header h-7. */
const DESKTOP_GROUP_MENU_VERTICAL_CHROME = 40;
/** Model submenu non-list chrome: p-1.5 × 2. */
const DESKTOP_SUBMENU_VERTICAL_CHROME = 12;

function resolveModelGroups(modelOptions: ChatModelOption[]) {
  const groupMap = new Map<string, { label: string; icon: string; items: ChatModelOption[] }>();
  for (const item of modelOptions) {
    const presentation = resolveModelPresentationGroup(item);
    const group = groupMap.get(presentation.key);
    if (group) {
      group.items.push(item);
      continue;
    }
    groupMap.set(presentation.key, {
      label: presentation.label,
      icon: presentation.icon,
      items: [item],
    });
  }
  return `min(${contentHeight}px, calc(100vh - ${96 + chromeHeight}px))`;
}

  return Array.from(groupMap.entries()).map(([key, group]) => ({ key, ...group }));
}

function ChatModelIdentity({
  model,
  density = "default",
}: {
  model: ChatModelOption;
  density?: "default" | "compact";
}) {
  const platformModelName = model.platformModelName.trim();
  const identity = React.useMemo(
    () =>
      resolveModelIdentity({
        code: model.platformModelName,
        vendor: model.vendor,
        icon: model.icon,
      }),
    [model.icon, model.platformModelName, model.vendor],
  );
  const iconURL = React.useMemo(() => resolveModelIconURL(identity.modelIcon), [identity.modelIcon]);
  const compact = density === "compact";

  return (
    <div className={cn("flex min-w-0 items-center", compact ? "gap-2" : "gap-2.5")}>
      <ModelIcon iconUrl={iconURL} label={platformModelName} />
      <div className="min-w-0 flex-1 overflow-hidden">
        <div className={cn("flex items-center", compact ? "gap-1" : "gap-1.5")}>
          <p
            className={cn(
              "truncate font-medium text-foreground",
              compact ? "text-[12.5px] leading-4" : "text-[13px] leading-4.5",
            )}
          >
            {platformModelName}
          </p>
        </div>
      </div>
    </div>
  );
}

function ChatModelTriggerSkeleton() {
  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <Skeleton className="size-4 shrink-0 rounded-full bg-muted/55" />
      <Skeleton className="h-3.5 w-20 rounded-full bg-muted/50" />
    </div>
  );
}

function ModelMenuScrollContainer({
  maxHeight,
  children,
  maxHeight,
  onScroll,
}: {
  maxHeight: string;
  children: React.ReactNode;
  maxHeight?: number;
  onScroll?: () => void;
}) {
  const viewportRef = React.useRef<HTMLDivElement | null>(null);
  const [hasMoreAbove, setHasMoreAbove] = React.useState(false);
  const [hasMoreBelow, setHasMoreBelow] = React.useState(false);
  const resolvedMaxHeight = Number.isFinite(maxHeight) ? Math.max(0, maxHeight ?? 0) : undefined;

  const updateScrollHints = React.useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) {
      setHasMoreAbove(false);
      setHasMoreBelow(false);
      return;
    }
    const remaining = viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop;
    setHasMoreAbove(viewport.scrollTop > MODEL_MENU_SCROLL_MORE_THRESHOLD);
    setHasMoreBelow(remaining > MODEL_MENU_SCROLL_MORE_THRESHOLD);
  }, []);

  React.useLayoutEffect(() => {
    updateScrollHints();
    const viewport = viewportRef.current;
    if (!viewport || typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver(updateScrollHints);
    observer.observe(viewport);
    if (viewport.firstElementChild) {
      observer.observe(viewport.firstElementChild);
    }
    return () => observer.disconnect();
  }, [children, resolvedMaxHeight, updateScrollHints]);

  const handleScroll = React.useCallback(() => {
    updateScrollHints();
    onScroll?.();
  }, [onScroll, updateScrollHints]);

  return (
    <div className="relative">
      <div
        ref={viewportRef}
        style={resolvedMaxHeight === undefined ? undefined : { maxHeight: resolvedMaxHeight }}
        className={cn(
          "overflow-y-auto overscroll-contain pr-0 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
          resolvedMaxHeight === undefined
            ? "max-h-[min(20rem,var(--model-menu-scroll-max-height,var(--radix-popover-content-available-height)))]"
            : null,
        )}
        onScroll={handleScroll}
      >
        <div className="pb-1">
          {children}
        </div>
      </div>
      {hasMoreAbove ? (
        <div className="pointer-events-none absolute inset-x-0 top-0 flex h-4 items-start justify-center rounded-t-lg bg-gradient-to-b from-popover via-popover/80 to-transparent pt-px">
          <ChevronDown className="size-3 rotate-180 text-muted-foreground/75" strokeWidth={1.8} />
        </div>
      ) : null}
      {hasMoreBelow ? (
        <div className="pointer-events-none absolute inset-x-0 bottom-0 flex h-4 items-end justify-center rounded-b-lg bg-gradient-to-t from-popover via-popover/80 to-transparent pb-px">
          <ChevronDown className="size-3 text-muted-foreground/75" strokeWidth={1.8} />
        </div>
      ) : null}
    </div>
  );
}

function ModelPricingTooltipContent({
  platformModelName,
  protocols,
  pricing,
  billingDisplay,
  labels,
}: {
  platformModelName: string;
  protocols: readonly string[];
  pricing: NonNullable<ChatModelOption["pricing"]>;
  billingDisplay: BillingDisplayOptions;
  labels: {
    freeModel: string;
    freeModelDescription: string;
    tieredPricing: string;
    callPricing: string;
    durationPricing: string;
    tokenPricing: string;
    input: string;
    output: string;
    cacheRead: string;
    perCall: string;
    perSecond: string;
    callUnit: string;
    secondUnit: string;
    billingDisplay: BillingDisplayLabels;
  };
}) {
  const cacheWriteLabel = cacheWritePricingLabel(protocols, labels.billingDisplay);
  const cacheWriteNote = cacheWritePricingNote(protocols, labels.billingDisplay);
  if (pricing.isFree) {
    return (
      <div className="flex flex-col gap-1">
        <span className={PRICING_TOOLTIP_TITLE_CLASS}>{labels.freeModel}</span>
        <span className={PRICING_TOOLTIP_BODY_CLASS}>{labels.freeModelDescription}</span>
      </div>
    );
  }

  if (pricing.mode === "tiered") {
    return (
      <PricingTable
        platformModelName={platformModelName}
        title={labels.tieredPricing}
        footerNote={cacheWriteNote}
        headerRow={["", ...pricing.tiers.map((tier) => formatTokenRange(tier.fromTokens, tier.upToTokens))]}
        bodyRows={[
          [labels.input, ...pricing.tiers.map((tier) => formatPricingUnitUSD(tier.inputUSDPerMTokens, billingDisplay))],
          [labels.output, ...pricing.tiers.map((tier) => formatPricingUnitUSD(tier.outputUSDPerMTokens, billingDisplay))],
          [labels.cacheRead, ...pricing.tiers.map((tier) => formatPricingUnitUSD(tier.cacheReadUSDPerMTokens, billingDisplay))],
          [cacheWriteLabel, ...pricing.tiers.map((tier) => formatPricingUnitUSD(resolveCacheWritePricingUSD(protocols, tier.cacheWriteUSDPerMTokens), billingDisplay))],
        ]}
      />
    );
  }

  if (pricing.mode === "call") {
    return (
      <div className="flex flex-col gap-1">
        <span className={PRICING_TOOLTIP_TITLE_CLASS}>{labels.callPricing}</span>
        <PricingTooltipRow label={labels.perCall} value={`${formatPricingUnitUSD(pricing.callUSDPerCall, billingDisplay)} / ${labels.callUnit}`} />
      </div>
    );
  }

  if (pricing.mode === "duration") {
    return (
      <div className="flex flex-col gap-1">
        <span className={PRICING_TOOLTIP_TITLE_CLASS}>{labels.durationPricing}</span>
        <PricingTooltipRow label={labels.perSecond} value={`${formatPricingUnitUSD(pricing.durationUSDPerSecond, billingDisplay)} / ${labels.secondUnit}`} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      <span className={PRICING_TOOLTIP_TITLE_CLASS}>{labels.tokenPricing}</span>
      <PricingTooltipRow label={labels.input} value={`${formatPricingUnitUSD(pricing.inputUSDPerMTokens, billingDisplay)} / 1M tokens`} />
      <PricingTooltipRow label={labels.output} value={`${formatPricingUnitUSD(pricing.outputUSDPerMTokens, billingDisplay)} / 1M tokens`} />
      <PricingTooltipRow label={labels.cacheRead} value={`${formatPricingUnitUSD(pricing.cacheReadUSDPerMTokens, billingDisplay)} / 1M tokens`} />
      <PricingTooltipRow label={cacheWriteLabel} value={`${formatPricingUnitUSD(resolveCacheWritePricingUSD(protocols, pricing.cacheWriteUSDPerMTokens), billingDisplay)} / 1M tokens`} />
      {cacheWriteNote ? <span className={cn(PRICING_TOOLTIP_BODY_CLASS, "block max-w-72 text-background/70")}>{cacheWriteNote}</span> : null}
    </div>
  );
}

function PricingTooltipRow({ label, value }: { label: string; value: string }) {
  return (
    <div className={cn("grid grid-cols-[minmax(5.5rem,max-content)_auto] items-baseline gap-5", PRICING_TOOLTIP_BODY_CLASS)}>
      <span className="whitespace-nowrap text-left">{label}</span>
      <span className="whitespace-nowrap text-right tabular-nums">{value}</span>
    </div>
  );
}

function PricingTable({
  platformModelName,
  title,
  footerNote,
  headerRow,
  bodyRows,
}: {
  platformModelName: string;
  title: string;
  footerNote?: string | null;
  headerRow: string[];
  bodyRows: string[][];
}) {
  return (
    <div className="flex max-w-[560px] flex-col gap-2 overflow-x-auto">
      <span className={PRICING_TOOLTIP_TITLE_CLASS}>{title}</span>
      <table className={cn("border-collapse text-left tabular-nums", PRICING_TOOLTIP_BODY_CLASS)}>
        <thead>
          <tr className="border-b border-background/20">
            {headerRow.map((cell, index) => (
              <th
                key={`${platformModelName}-pricing-head-${index}`}
                scope="col"
                className={cn(
                  "whitespace-nowrap px-2 pb-1 font-medium text-background/70 first:pl-0 last:pr-0",
                  index > 0 ? "text-right" : null,
                )}
              >
                {cell}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {bodyRows.map((row, rowIndex) => (
            <tr key={`${platformModelName}-pricing-row-${rowIndex}`} className="border-b border-background/10 last:border-0">
              {row.map((cell, cellIndex) => (
                <td
                  key={`${platformModelName}-pricing-cell-${rowIndex}-${cellIndex}`}
                  className={cn(
                    "whitespace-nowrap px-2 py-1 first:pl-0 last:pr-0",
                    cellIndex === 0 ? "font-medium text-background/90" : "text-right",
                  )}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {footerNote ? <span className={cn(PRICING_TOOLTIP_BODY_CLASS, "block text-background/70")}>{footerNote}</span> : null}
    </div>
  );
}

function formatPricingUnitUSD(value: number, billingDisplay: BillingDisplayOptions): string {
  return formatBillingDisplayUnitPriceFromUSD(value, billingDisplay);
}

function formatTokenRange(fromTokens: number, upToTokens: number | null): string {
  if (!upToTokens || upToTokens <= 0) {
    return `${formatTokenQuantity(fromTokens)}～∞`;
  }
  return `${formatTokenQuantity(fromTokens)}～${formatTokenQuantity(upToTokens)}`;
}

function formatTokenQuantity(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "0";
  }
  if (value >= 1000000 && value % 1000000 === 0) {
    return `${value / 1000000}M`;
  }
  if (value >= 1000 && value % 1000 === 0) {
    return `${value / 1000}K`;
  }
  return String(value);
}

function ChatModelMenuItem({
  model,
  selected,
  onSelect,
  billingDisplay,
  pricingLabels,
  viewPricingLabel,
  pricingTooltipSide,
  buttonRef,
}: {
  model: ChatModelOption;
  selected: boolean;
  onSelect: () => void;
  billingDisplay: BillingDisplayOptions;
  pricingLabels: React.ComponentProps<typeof ModelPricingTooltipContent>["labels"];
  viewPricingLabel: string;
  pricingTooltipSide: "right";
  buttonRef?: React.Ref<HTMLButtonElement>;
}) {
  const platformModelName = model.platformModelName.trim();
  const identity = React.useMemo(
    () =>
      resolveModelIdentity({
        code: model.platformModelName,
        vendor: model.vendor,
        icon: model.icon,
      }),
    [model.icon, model.platformModelName, model.vendor],
  );
  const iconURL = React.useMemo(() => resolveModelIconURL(identity.modelIcon), [identity.modelIcon]);

  return (
    <div
      data-selected={selected}
      className={cn(
        "group flex items-center rounded-md font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-within:bg-accent focus-within:text-accent-foreground data-[selected=true]:bg-accent data-[selected=true]:text-accent-foreground",
        touch ? "h-8 text-[11px]" : "h-7 text-[11px]",
      )}
    >
      <button
        ref={buttonRef}
        type="button"
        className={cn(
          "flex min-w-0 flex-1 items-center gap-2 rounded-md bg-transparent text-left font-medium leading-none text-inherit outline-none",
          touch ? "relative h-8 py-0 pl-2 pr-1 after:absolute after:-inset-y-1.5 after:inset-x-0 after:content-['']" : "h-7 py-0 pl-2 pr-1",
        )}
        onClick={onSelect}
      >
        <ModelIcon iconUrl={iconURL} label={platformModelName} />
        <span className="min-w-0 flex-1 truncate leading-4">
          {platformModelName}
        </span>
        <span className={cn("flex shrink-0 items-center justify-center", touch ? "size-4" : "size-3")}>
          {selected ? <Check className="size-3 text-current" strokeWidth={1.7} /> : null}
        </span>
      </button>
      {model.pricing ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className={cn(
                "flex shrink-0 items-center justify-center rounded-md text-muted-foreground/70 transition-colors hover:text-current focus-visible:text-current focus-visible:outline-none group-hover:text-current group-focus-within:text-current group-data-[selected=true]:text-current",
                touch ? "relative h-8 w-8 after:absolute after:-inset-y-1.5 after:inset-x-0 after:content-['']" : "h-7 w-7",
              )}
              aria-label={viewPricingLabel}
            >
              {model.pricing.isFree ? (
                <TicketSlash className="size-3.5" strokeWidth={1.8} />
              ) : (
                <CircleDollarSign className="size-3.5" strokeWidth={1.8} />
              )}
            </button>
          </TooltipTrigger>
          <TooltipContent
            side={pricingTooltipSide}
            align="center"
            sideOffset={8}
            className="z-[80] max-w-[min(92vw,35rem)] text-left font-medium tabular-nums"
          >
            <ModelPricingTooltipContent
              platformModelName={model.platformModelName}
              protocols={model.protocols}
              pricing={model.pricing}
              billingDisplay={billingDisplay}
              labels={pricingLabels}
            />
          </TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  );
}

function ModelPickerModelList({
  items,
  density,
  selectedPlatformModelName,
  billingDisplay,
  pricingLabels,
  viewPricingLabel,
  onSelectModel,
}: {
  items: readonly ChatModelOption[];
  density: "compact" | "touch";
  selectedPlatformModelName: string;
  billingDisplay: BillingDisplayOptions;
  pricingLabels: React.ComponentProps<typeof ModelPricingTooltipContent>["labels"];
  viewPricingLabel: string;
  onSelectModel: (platformModelName: string) => void;
}) {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-0.5">
      {items.map((item) => (
        <ChatModelMenuItem
          key={item.platformModelName}
          model={item}
          selected={item.platformModelName === selectedPlatformModelName}
          onSelect={() => onSelectModel(item.platformModelName)}
          billingDisplay={billingDisplay}
          pricingLabels={pricingLabels}
          viewPricingLabel={viewPricingLabel}
          pricingTooltipSide="right"
          density={density}
        />
      ))}
    </div>
  );
}

function ModelPickerGroupedModelList({
  groups,
  density,
  selectedPlatformModelName,
  billingDisplay,
  pricingLabels,
  viewPricingLabel,
  onSelectModel,
}: {
  groups: readonly {
    vendor: string;
    label: string;
    icon: string;
    items: ChatModelOption[];
  }[];
  density: "compact" | "touch";
  selectedPlatformModelName: string;
  billingDisplay: BillingDisplayOptions;
  pricingLabels: React.ComponentProps<typeof ModelPricingTooltipContent>["labels"];
  viewPricingLabel: string;
  onSelectModel: (platformModelName: string) => void;
}) {
  if (groups.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {groups.map((group) => {
        const iconURL = resolveLobeHubIconURL(group.icon);
        return (
          <div key={group.vendor} className="flex flex-col gap-0.5">
            <div className="flex items-center gap-2 px-2 pt-1 text-[10px] font-medium text-muted-foreground">
              <LobeHubIcon iconUrl={iconURL} label={group.label} />
              <span className="min-w-0 flex-1 truncate">{group.label}</span>
              <span className="shrink-0 tabular-nums">{group.items.length}</span>
            </div>
            <ModelPickerModelList
              items={group.items}
              density={density}
              selectedPlatformModelName={selectedPlatformModelName}
              billingDisplay={billingDisplay}
              pricingLabels={pricingLabels}
              viewPricingLabel={viewPricingLabel}
              onSelectModel={onSelectModel}
            />
          </div>
        );
      })}
    </div>
  );
}

export function ChatModelPicker({
  modelOptions,
  billingDisplayCurrency,
  billingDisplayUsdToCnyRate,
  selectedPlatformModelName,
  loading,
  disabled,
  onModelCatalogRefresh,
  onModelChange,
}: ChatModelPickerProps) {
  const t = useTranslations("chat.modelPicker");
  const isMobile = useIsMobile();
  const [open, setOpen] = React.useState(false);
  const [activeGroupKey, setActiveGroupKey] = React.useState("");
  const [mobileGroupKey, setMobileGroupKey] = React.useState<string | null>(null);
  const [desktopSubmenuSide, setDesktopSubmenuSide] = React.useState<"right" | "left">("right");
  const [desktopSubmenuTop, setDesktopSubmenuTop] = React.useState(0);
  const [desktopSubmenuWidth, setDesktopSubmenuWidth] = React.useState(DESKTOP_MODEL_MENU_WIDTH);
  const [desktopGroupListMaxHeight, setDesktopGroupListMaxHeight] = React.useState(320);
  const [desktopSubmenuListMaxHeight, setDesktopSubmenuListMaxHeight] = React.useState(320);
  const desktopMenuRootRef = React.useRef<HTMLDivElement | null>(null);
  const desktopGroupMenuRef = React.useRef<HTMLDivElement | null>(null);
  const desktopSubmenuRef = React.useRef<HTMLDivElement | null>(null);
  const selectedModelButtonRef = React.useRef<HTMLButtonElement | null>(null);
  const desktopGroupItemRefs = React.useRef(new Map<string, HTMLButtonElement>());
  const selectedModel = React.useMemo(
    () => modelOptions.find((item) => item.platformModelName === selectedPlatformModelName) ?? null,
    [modelOptions, selectedPlatformModelName],
  );
  const selectedGroupKey = React.useMemo(() => {
    if (!selectedModel) {
      return "";
    }
    return resolveModelPresentationGroup(selectedModel).key;
  }, [selectedModel]);
  const selectedGroupLabel = React.useMemo(() => {
    if (!selectedModel) {
      return t("vendor");
    }
    return resolveModelPresentationGroup(selectedModel).label;
  }, [selectedModel]);
  const modelGroups = React.useMemo(() => resolveModelGroups(modelOptions), [modelOptions]);
  const activeDesktopGroupKey = activeGroupKey || selectedGroupKey || modelGroups[0]?.key || "";
  const activeDesktopGroup = React.useMemo(
    () => modelGroups.find((group) => group.key === activeDesktopGroupKey) ?? modelGroups[0] ?? null,
    [activeDesktopGroupKey, modelGroups],
  );
  const hasDesktopModelSubmenu = Boolean(activeDesktopGroup?.items.length);
  const mobileGroup = React.useMemo(
    () => modelGroups.find((group) => group.key === mobileGroupKey) ?? null,
    [mobileGroupKey, modelGroups],
  );
  const pricingLabels = React.useMemo(
    () => ({
      freeModel: t("freeModel"),
      freeModelDescription: t("freeModelDescription"),
      tieredPricing: t("tieredPricing"),
      callPricing: t("callPricing"),
      durationPricing: t("durationPricing"),
      tokenPricing: t("tokenPricing"),
      input: t("input"),
      output: t("output"),
      cacheRead: t("cacheRead"),
      perCall: t("perCall"),
      perSecond: t("perSecond"),
      callUnit: t("callUnit"),
      secondUnit: t("secondUnit"),
      billingDisplay: {
        cacheWrite: t("cacheWrite"),
        cacheWrite5m: t("cacheWrite5m"),
        cacheWrite1h: t("cacheWrite1h"),
        cacheWrite5m1h: t("cacheWrite5m1h"),
        claudeCacheWriteMixedNote: (multiplier: string) => t("claudeCacheWriteMixedNote", { multiplier }),
        claudeCacheWriteNote: (timeout: "5m" | "1h", multiplier: string) => t("claudeCacheWriteNote", { timeout, multiplier }),
        claudeFastModeNote: (multiplier: string) => t("claudeFastModeNote", { multiplier }),
        openaiServiceTierNote: (tier: string, multiplier: string) => t("openaiServiceTierNote", { tier, multiplier }),
        cacheWritePricingLabel: t("cacheWritePricingLabel"),
        cacheWritePricingNote: t("cacheWritePricingNote"),
      },
    }),
    [t],
  );

  const billingDisplay = React.useMemo<BillingDisplayOptions>(
    () => ({
      currency: billingDisplayCurrency,
      usdToCnyRate: billingDisplayUsdToCnyRate,
    }),
    [billingDisplayCurrency, billingDisplayUsdToCnyRate],
  );

  React.useEffect(() => {
    if (!open || !isMobile) {
      setMobileGroupKey(null);
    }
  }, [isMobile, open]);

  const updateDesktopSubmenuMetrics = React.useCallback(() => {
    if (!open || isMobile) {
      setDesktopSubmenuSide("right");
      setDesktopSubmenuTop(0);
      setDesktopSubmenuWidth(DESKTOP_MODEL_MENU_WIDTH);
      setDesktopGroupListMaxHeight(320);
      setDesktopSubmenuListMaxHeight(320);
      return;
    }
  }, [open]);

    const menuRoot = desktopMenuRootRef.current;
    const groupMenu = desktopGroupMenuRef.current;
    if (!menuRoot || !groupMenu) {
      return;
    }

    const menuRootRect = menuRoot.getBoundingClientRect();
    const groupMenuRect = groupMenu.getBoundingClientRect();
    const submenu = desktopSubmenuRef.current;
    const submenuRect = submenu?.getBoundingClientRect();
    const activeGroupButton = activeDesktopGroup
      ? desktopGroupItemRefs.current.get(activeDesktopGroup.key)
      : null;
    const activeGroupRect = activeGroupButton?.getBoundingClientRect();
    const triggerRect = document.getElementById("chat-model-menu-trigger")?.getBoundingClientRect();
    const viewportLeft = MODEL_MENU_COLLISION_PADDING;
    const viewportRight = window.innerWidth - MODEL_MENU_COLLISION_PADDING;
    const viewportTop = MODEL_MENU_COLLISION_PADDING;
    const viewportBottom = window.innerHeight - MODEL_MENU_COLLISION_PADDING;
    const viewportHeight = Math.max(0, viewportBottom - viewportTop);
    const rightAvailableWidth = Math.max(0, viewportRight - groupMenuRect.right - DESKTOP_MODEL_SUBMENU_GAP);
    const leftAvailableWidth = Math.max(0, groupMenuRect.left - viewportLeft - DESKTOP_MODEL_SUBMENU_GAP);
    const nextSubmenuSide =
      rightAvailableWidth >= DESKTOP_MODEL_MENU_WIDTH || rightAvailableWidth >= leftAvailableWidth
        ? "right"
        : "left";
    const nextSubmenuWidth = Math.max(
      DESKTOP_MODEL_MENU_MIN_SCROLL_HEIGHT,
      Math.min(
        DESKTOP_MODEL_MENU_WIDTH,
        nextSubmenuSide === "right" ? rightAvailableWidth : leftAvailableWidth,
      ),
    );
    const nextGroupListMaxHeight = resolveDesktopModelMenuListMaxHeight({
      viewportTop,
      viewportBottom,
      triggerTop: triggerRect?.top,
      triggerBottom: triggerRect?.bottom,
      sideOffset: DESKTOP_MODEL_MENU_SIDE_OFFSET,
      verticalChrome: DESKTOP_GROUP_MENU_VERTICAL_CHROME,
    });

    let nextSubmenuTop = 0;
    let nextSubmenuListMaxHeight = nextGroupListMaxHeight;
    if (hasDesktopModelSubmenu && activeGroupRect) {
      const submenuHeight = submenuRect?.height ?? nextGroupListMaxHeight + DESKTOP_SUBMENU_VERTICAL_CHROME;
      const submenuOuterHeight = Math.min(submenuHeight, viewportHeight);
      const maxViewportTop = Math.max(viewportTop, viewportBottom - submenuOuterHeight);
      // Anchor in viewport coordinates, then convert to an offset within
      // menuRoot (which may sit above viewportTop for a frame before re-shift).
      const preferredSubmenuViewportTop = Math.min(
        Math.max(activeGroupRect.top, viewportTop),
        maxViewportTop,
      );
      nextSubmenuTop = preferredSubmenuViewportTop - menuRootRect.top;
      const actualSubmenuViewportTop = menuRootRect.top + nextSubmenuTop;
      nextSubmenuListMaxHeight = resolveDesktopMenuListMaxHeight(
        Math.min(viewportHeight, viewportBottom - actualSubmenuViewportTop),
        DESKTOP_SUBMENU_VERTICAL_CHROME,
      );
    }

    setDesktopSubmenuSide(nextSubmenuSide);
    setDesktopSubmenuTop(nextSubmenuTop);
    setDesktopSubmenuWidth(nextSubmenuWidth);
    setDesktopGroupListMaxHeight(nextGroupListMaxHeight);
    setDesktopSubmenuListMaxHeight(nextSubmenuListMaxHeight);
  }, [activeDesktopGroup, hasDesktopModelSubmenu, isMobile, open]);

  React.useLayoutEffect(() => {
    updateDesktopSubmenuMetrics();

    if (!open || isMobile) {
      return;
    }

    window.addEventListener("resize", updateDesktopSubmenuMetrics);
    window.addEventListener("scroll", updateDesktopSubmenuMetrics, true);
    if (typeof ResizeObserver === "undefined") {
      return () => {
        window.removeEventListener("resize", updateDesktopSubmenuMetrics);
        window.removeEventListener("scroll", updateDesktopSubmenuMetrics, true);
      };
    }

    const observer = new ResizeObserver(updateDesktopSubmenuMetrics);
    if (desktopMenuRootRef.current) {
      observer.observe(desktopMenuRootRef.current);
    }
    if (desktopGroupMenuRef.current) {
      observer.observe(desktopGroupMenuRef.current);
    }
    if (desktopSubmenuRef.current) {
      observer.observe(desktopSubmenuRef.current);
    }
    const activeGroupButton = activeDesktopGroup
      ? desktopGroupItemRefs.current.get(activeDesktopGroup.key)
      : null;
    if (activeGroupButton) {
      observer.observe(activeGroupButton);
    }

    return () => {
      window.removeEventListener("resize", updateDesktopSubmenuMetrics);
      window.removeEventListener("scroll", updateDesktopSubmenuMetrics, true);
      observer.disconnect();
    };
  }, [activeDesktopGroup, hasDesktopModelSubmenu, isMobile, open, updateDesktopSubmenuMetrics]);

  const handleOpenChange = React.useCallback(
    (nextOpen: boolean) => {
      resetDesktopModelPanelLayout();
      if (nextOpen) {
        setActiveGroupKey(selectedGroupKey || modelGroups[0]?.key || "");
        if (onModelCatalogRefresh) {
          void Promise.resolve(onModelCatalogRefresh()).catch(() => undefined);
        }
      } else {
        setSearchQuery("");
      }
      setOpen(nextOpen);
    },
    [modelGroups, onModelCatalogRefresh, selectedGroupKey],
  );

  const updateDesktopModelPanelLayout = React.useCallback((layoutKey: number) => {
    if (!open || isMobile || isSearchMode || typeof window === "undefined") {
      return;
    }
    if (desktopModelPanelKeyRef.current !== layoutKey) {
      return;
    }
    const menu = desktopPopoverContentRef.current;
    if (!menu || !activeDesktopVendorGroup) {
      return;
    }

    const menuRect = menu.getBoundingClientRect();
    if (menuRect.width <= 0 || menuRect.height <= 0) {
      return;
    }
    const panelWidth = Math.min(
      desktopModelMenuWidthValue,
      Math.max(0, window.innerWidth - MODEL_MENU_COLLISION_GUTTER * 2),
    );
    const panelChromeHeight = MODEL_MENU_MODEL_PANEL_CHROME_HEIGHT;
    const actualContentHeight = activeDesktopVendorGroup.items.length > 0
      ? activeDesktopVendorGroup.items.length * MODEL_MENU_MODEL_ROW_HEIGHT
        + Math.max(0, activeDesktopVendorGroup.items.length - 1) * MODEL_MENU_ROW_GAP
        + MODEL_MENU_LIST_PADDING_BOTTOM
      : 0;
    const contentHeight = Math.min(actualContentHeight, MODEL_MENU_MAX_HEIGHT);
    const maxListHeight = Math.max(
      MODEL_MENU_MODEL_ROW_HEIGHT,
      window.innerHeight - MODEL_MENU_COLLISION_GUTTER * 2 - panelChromeHeight,
    );
    const initialListHeight = Math.min(
      contentHeight,
      maxListHeight,
    );
    const initialPanelHeight = panelChromeHeight + initialListHeight;
    const preferredY = menuRect.top + initialPanelHeight <= window.innerHeight - MODEL_MENU_COLLISION_GUTTER
      ? menuRect.top
      : menuRect.bottom - initialPanelHeight;
    const y = Math.min(
      Math.max(preferredY, MODEL_MENU_COLLISION_GUTTER),
      Math.max(MODEL_MENU_COLLISION_GUTTER, window.innerHeight - initialPanelHeight - MODEL_MENU_COLLISION_GUTTER),
    );
    const listMaxHeight = Math.min(
      contentHeight,
      Math.max(
        MODEL_MENU_MODEL_ROW_HEIGHT,
        Math.min(
          maxListHeight,
          window.innerHeight - y - MODEL_MENU_COLLISION_GUTTER - panelChromeHeight,
        ),
      ),
    );
    const rightX = menuRect.right + MODEL_MENU_PANEL_GAP;
    const leftX = menuRect.left - MODEL_MENU_PANEL_GAP - panelWidth;
    const rightFits = rightX + panelWidth <= window.innerWidth - MODEL_MENU_COLLISION_GUTTER;
    const leftFits = leftX >= MODEL_MENU_COLLISION_GUTTER;
    const preferredX = rightFits || !leftFits ? rightX : leftX;
    const x = Math.min(
      Math.max(preferredX, MODEL_MENU_COLLISION_GUTTER),
      Math.max(MODEL_MENU_COLLISION_GUTTER, window.innerWidth - panelWidth - MODEL_MENU_COLLISION_GUTTER),
    );

    setDesktopModelPanelLayout({ key: layoutKey, x, y, width: panelWidth, listMaxHeight });
  }, [activeDesktopVendorGroup, desktopModelMenuWidthValue, isMobile, isSearchMode, open]);

  React.useLayoutEffect(() => {
    if (!open || isMobile || isSearchMode) {
      setDesktopModelPanelLayout(null);
      return;
    }

    const layoutKey = desktopModelPanelKey;
    let frameID = window.requestAnimationFrame(() => {
      frameID = window.requestAnimationFrame(() => {
        updateDesktopModelPanelLayout(layoutKey);
      });
    });
    const update = () => updateDesktopModelPanelLayout(layoutKey);
    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    return () => {
      window.cancelAnimationFrame(frameID);
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
    };
  }, [
    activeDesktopVendorGroup,
    desktopModelMenuWidthValue,
    desktopModelPanelKey,
    isSearchMode,
    isMobile,
    open,
    updateDesktopModelPanelLayout,
  ]);

  const closeMenu = React.useCallback(() => {
    handleOpenChange(false);
  }, [handleOpenChange]);

  const selectDesktopGroup = React.useCallback((groupKey: string) => {
    if (groupKey === activeDesktopGroupKey) {
      return;
    }
    setActiveGroupKey(groupKey);
  }, [activeDesktopGroupKey]);

  return (
    <>
      <div className="min-w-0 max-w-[min(320px,100%)] shrink">
        <Popover open={open} onOpenChange={handleOpenChange}>
          <PopoverTrigger asChild>
            <InputGroupButton
              id="chat-model-menu-trigger"
              type="button"
              variant="ghost"
              size="sm"
              className="w-full min-w-0 max-w-[min(320px,100%)] rounded-lg px-1.5 hover:bg-accent focus-visible:bg-accent data-[state=open]:bg-accent sm:px-2"
              disabled={disabled || loading}
              aria-label={t("selectModel")}
            >
              {loading ? (
                <ChatModelTriggerSkeleton />
              ) : selectedModel ? (
                <ChatModelIdentity model={selectedModel} density="compact" />
              ) : selectedPlatformModelName.trim() ? (
                <span className="truncate text-[12px] font-medium text-foreground">
                  {selectedPlatformModelName}
                </span>
              ) : (
                <span className="truncate text-[12px] font-medium text-muted-foreground">
                  {t("selectModel")}
                </span>
              )}
            </InputGroupButton>
          </PopoverTrigger>
          <PopoverContent
            align="end"
            side="bottom"
            sideOffset={DESKTOP_MODEL_MENU_SIDE_OFFSET}
            collisionPadding={24}
            onOpenAutoFocus={(event) => {
              if (!isMobile && selectedModelButtonRef.current) {
                event.preventDefault();
                selectedModelButtonRef.current.focus();
              }
            }}
            className={cn(
              "relative overflow-visible rounded-xl",
              isMobile
                ? "w-[min(20rem,calc(100vw-3rem))] p-1.5"
                : "w-[min(14rem,calc(100vw-3rem))] border-0 bg-transparent p-0 shadow-none",
            )}
          >
            {isMobile ? (
              <>
                <div className="flex h-7 items-center justify-between gap-2 px-2">
                  {mobileGroup ? (
                    <button
                      type="button"
                      className="-ml-1.5 flex h-7 min-w-0 items-center gap-0.5 rounded-md px-0.5 text-[11px] font-medium text-muted-foreground outline-none transition-colors hover:bg-accent hover:text-foreground focus-visible:bg-accent focus-visible:text-foreground"
                      onClick={() => setMobileGroupKey(null)}
                    >
                      <ChevronLeft className="size-3.5" strokeWidth={1.8} />
                      <span>{t("group")}</span>
                    </button>
                  ) : (
                    <span className="text-[11px] font-medium text-foreground">{t("group")}</span>
                  )}
                  <span className="min-w-0 truncate text-right text-[10px] font-medium text-muted-foreground">
                    {mobileGroup ? mobileGroup.label : selectedGroupLabel}
                  </span>
                </div>
                {modelGroups.length === 0 ? (
                  <div className="px-2 py-3 text-[11px] leading-4 text-muted-foreground">
                    {t("empty")}
                  </div>
                ) : (
                  <ModelMenuScrollContainer>
                    {mobileGroup ? (
                      <div className="flex flex-col gap-0.5">
                        {mobileGroup.items.map((item) => (
                          <ChatModelMenuItem
                            key={item.platformModelName}
                            model={item}
                            selected={item.platformModelName === selectedPlatformModelName}
                            onSelect={() => {
                              onModelChange(item.platformModelName);
                              closeMenu();
                            }}
                            billingDisplay={billingDisplay}
                            pricingLabels={pricingLabels}
                            viewPricingLabel={t("viewPricing")}
                            pricingTooltipSide="right"
                          />
                        ))}
                      </div>
                    ) : (
                      <div className="flex flex-col gap-0.5">
                        {modelGroups.map((group) => {
                          const selectedGroup = group.key === selectedGroupKey;
                          const groupIconURL = resolveModelIconURL(group.icon);
                          return (
                            <button
                              type="button"
                              key={group.key}
                              className={cn(
                                "flex h-7 w-full items-center justify-between gap-2 rounded-md px-2 py-0 text-left text-[11px] font-medium outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground",
                                selectedGroup ? "bg-accent text-accent-foreground" : "text-muted-foreground",
                              )}
                              onClick={() => {
                                setMobileGroupKey(group.key);
                              }}
                            >
                              <ModelIcon iconUrl={groupIconURL} label={group.label} />
                              <span className="min-w-0 flex-1 truncate font-medium">{group.label}</span>
                              <span className="shrink-0 text-[10px] tabular-nums text-muted-foreground/80">
                                {group.items.length}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                    )}
                  </ModelMenuScrollContainer>
                )}
              </>
            ) : (
              <div ref={desktopMenuRootRef} className="relative min-w-0">
                {hasDesktopModelSubmenu ? (
                  <div
                    ref={desktopSubmenuRef}
                    style={{
                      top: desktopSubmenuTop,
                      width: desktopSubmenuWidth,
                    } as React.CSSProperties}
                    className={cn(
                      "absolute flex max-h-[calc(100dvh-3rem)] flex-col overflow-hidden rounded-xl border-[0.5px] border-border bg-popover p-1.5 shadow-xs",
                      desktopSubmenuSide === "right" ? "left-[calc(100%+0.5rem)]" : "right-[calc(100%+0.5rem)]",
                    )}
                  >
                    <ModelMenuScrollContainer maxHeight={desktopSubmenuListMaxHeight}>
                      <div className="flex flex-col gap-0.5">
                        {activeDesktopGroup?.items.map((item) => (
                          <ChatModelMenuItem
                            key={item.platformModelName}
                            model={item}
                            selected={item.platformModelName === selectedPlatformModelName}
                            buttonRef={item.platformModelName === selectedPlatformModelName ? selectedModelButtonRef : undefined}
                            onSelect={() => {
                              onModelChange(item.platformModelName);
                              closeMenu();
                            }}
                            billingDisplay={billingDisplay}
                            pricingLabels={pricingLabels}
                            viewPricingLabel={t("viewPricing")}
                            pricingTooltipSide="right"
                          />
                        ))}
                      </div>
                    </ModelMenuScrollContainer>
                  </div>
                ) : null}

                <div
                  ref={desktopGroupMenuRef}
                  className="flex min-w-0 max-h-[calc(100dvh-3rem)] flex-col overflow-hidden rounded-xl border-[0.5px] border-border bg-popover p-1.5 shadow-xs"
                >
                  <div className="flex h-7 shrink-0 items-center justify-between gap-3 px-2">
                    <span className="text-[11px] font-medium text-foreground">{t("group")}</span>
                    <span className="truncate text-[10px] font-medium text-muted-foreground">
                      {selectedGroupLabel}
                    </span>
                  </div>
                  {modelGroups.length === 0 ? (
                    <div className="px-2 py-3 text-[11px] leading-4 text-muted-foreground">
                      {t("empty")}
                    </div>
                  ) : (
                    <div className="min-h-0 min-w-0">
                      <ModelMenuScrollContainer maxHeight={desktopGroupListMaxHeight} onScroll={updateDesktopSubmenuMetrics}>
                        <div className="flex flex-col gap-0.5">
                          {modelGroups.map((group) => {
                            const selectedGroup = group.key === selectedGroupKey;
                            const activeGroup = group.key === activeDesktopGroup?.key;
                            const groupIconURL = resolveModelIconURL(group.icon);
                            return (
                              <button
                                type="button"
                                key={group.key}
                                ref={(node) => {
                                  if (node) {
                                    desktopGroupItemRefs.current.set(group.key, node);
                                    return;
                                  }
                                  desktopGroupItemRefs.current.delete(group.key);
                                }}
                                className={cn(
                                  "flex h-7 w-full items-center gap-2 rounded-md px-2 py-0 text-left text-[11px] font-medium outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent focus-visible:text-accent-foreground",
                                  activeGroup ? "bg-accent text-accent-foreground" : "text-muted-foreground",
                                  selectedGroup && !activeGroup ? "text-foreground" : null,
                                )}
                                onMouseEnter={() => selectDesktopGroup(group.key)}
                                onFocus={() => selectDesktopGroup(group.key)}
                                onClick={() => selectDesktopGroup(group.key)}
                              >
                                <ModelIcon iconUrl={groupIconURL} label={group.label} />
                                <span className="min-w-0 flex-1 truncate font-medium">{group.label}</span>
                                <span className="shrink-0 text-[10px] tabular-nums text-muted-foreground/80">
                                  {group.items.length}
                                </span>
                                <ChevronRight className="size-3.5 shrink-0 text-muted-foreground/65" strokeWidth={1.8} />
                              </button>
                            );
                          })}
                        </div>
                      </ModelMenuScrollContainer>
                    </div>
                  )}
                </div>
              </div>
            </DrawerContent>
          </Drawer>
        ) : (
          <Popover open={open} onOpenChange={handleOpenChange}>
            <PopoverTrigger asChild>
              {triggerButton}
            </PopoverTrigger>
            <PopoverContent
              align="end"
              sideOffset={8}
              className="relative overflow-visible rounded-xl p-1.5"
              ref={desktopPopoverContentRef}
              style={{ width: vendorMenuWidth }}
              onInteractOutside={(event) => {
                const target = event.target;
                if (target instanceof Node && desktopModelPanelRef.current?.contains(target)) {
                  event.preventDefault();
                }
              }}
            >
              {modelPickerBody}
            </PopoverContent>
          </Popover>
        )}
      </div>
      {open && !isMobile && !isSearchMode && activeDesktopVendorGroup && desktopModelPanelLayout && desktopModelPanelLayout.key === desktopModelPanelKey && typeof document !== "undefined"
        ? createPortal(
          <div
            ref={desktopModelPanelRef}
            className="fixed z-[60] rounded-xl border-[0.5px] border-border bg-popover p-1.5 text-popover-foreground shadow-xs"
            style={{
              left: desktopModelPanelLayout.x,
              top: desktopModelPanelLayout.y,
              width: desktopModelPanelLayout.width,
            }}
          >
            <ModelMenuScrollContainer maxHeight={desktopModelMenuMaxHeight}>
              <ModelPickerModelList
                items={activeDesktopVendorGroup.items}
                density="compact"
                selectedPlatformModelName={selectedPlatformModelName}
                billingDisplay={billingDisplay}
                pricingLabels={pricingLabels}
                viewPricingLabel={t("viewPricing")}
                onSelectModel={handleModelSelect}
              />
            </ModelMenuScrollContainer>
          </div>,
          document.body,
        )
        : null}
    </>
  );
}
