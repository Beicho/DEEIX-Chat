"use client";

import * as React from "react";
import { Camera, ClipboardCopy, Download, FileText, ImageDown, MousePointerClick, Share2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuItemIcon,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

type ConversationShareExportActionsProps = {
  shareLabel: string;
  exportLabel: string;
  onShare?: () => void;
  onExport?: () => void | Promise<void>;
  screenshotLatestLabel?: string;
  screenshotSelectLabel?: string;
  onScreenshotLatest?: () => void;
  onScreenshotSelect?: () => void;
};

type ConversationShareExportMenuItemsProps = ConversationShareExportActionsProps & {
  onCloseMenu?: () => void;
};

function hasConversationShareExportAction({
  onShare,
  onExport,
  onExportMarkdown,
  onExportImage,
  onCopyMarkdown,
  onScreenshotFull,
  onScreenshotSelect,
}: Partial<ConversationShareExportActionsProps>) {
  return Boolean(
    onShare ||
      onExport ||
      onExportMarkdown ||
      onExportImage ||
      onCopyMarkdown ||
      onScreenshotFull ||
      onScreenshotSelect,
  );
}

export function ConversationShareExportMenuItems({
  shareLabel,
  exportLabel,
  onShare,
  onExport,
  screenshotLatestLabel,
  screenshotSelectLabel,
  onScreenshotLatest,
  onScreenshotSelect,
  onCloseMenu,
}: ConversationShareExportMenuItemsProps) {
  const hasScreenshot = Boolean(onScreenshotLatest || onScreenshotSelect);
  return (
    <>
      <DropdownMenuItem
        disabled={!onShare}
        onSelect={(event) => {
          event.preventDefault();
          if (!onShare) return;
          onCloseMenu?.();
          onShare();
        }}
      >
        <DropdownMenuItemIcon icon={Share2} className="text-current" />
        {shareLabel}
      </DropdownMenuItem>
      <DropdownMenuItem
        disabled={!onExport}
        onSelect={(event) => {
          event.preventDefault();
          if (!onExport) return;
          onCloseMenu?.();
          void onExport();
        }}
      >
        <DropdownMenuItemIcon icon={Download} className="text-current" />
        {exportLabel}
      </DropdownMenuItem>
      {exportMarkdownLabel ? (
        <DropdownMenuItem
          disabled={!onExportMarkdown}
          onSelect={(event) => {
            event.preventDefault();
            if (!onExportMarkdown) return;
            onCloseMenu?.();
            void onExportMarkdown();
          }}
        >
          <DropdownMenuItemIcon icon={FileText} />
          {exportMarkdownLabel}
        </DropdownMenuItem>
      ) : null}
      {exportImageLabel ? (
        <DropdownMenuItem
          disabled={!onExportImage}
          onSelect={(event) => {
            event.preventDefault();
            if (!onExportImage) return;
            onCloseMenu?.();
            void onExportImage();
          }}
        >
          <DropdownMenuItemIcon icon={ImageDown} />
          {exportImageLabel}
        </DropdownMenuItem>
      ) : null}
      {copyMarkdownLabel ? (
        <DropdownMenuItem
          disabled={!onCopyMarkdown}
          onSelect={(event) => {
            event.preventDefault();
            if (!onCopyMarkdown) return;
            onCloseMenu?.();
            void onCopyMarkdown();
          }}
        >
          <DropdownMenuItemIcon icon={ClipboardCopy} />
          {copyMarkdownLabel}
        </DropdownMenuItem>
      ) : null}
      {hasScreenshot ? (
        <>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            disabled={!onScreenshotSelect}
            onSelect={(event) => {
              event.preventDefault();
              if (!onScreenshotSelect) return;
              onCloseMenu?.();
              onScreenshotSelect();
            }}
          >
            <DropdownMenuItemIcon icon={MousePointerClick} className="text-current" />
            {screenshotSelectLabel}
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={!onScreenshotLatest}
            onSelect={(event) => {
              event.preventDefault();
              if (!onScreenshotLatest) {
                return;
              }
              onCloseMenu?.();
              onScreenshotLatest();
            }}
          >
            <DropdownMenuItemIcon icon={Camera} className="text-current" />
            {screenshotLatestLabel}
          </DropdownMenuItem>
        </>
      ) : null}
    </>
  );
}

type ConversationShareExportIconDropdownProps = {
  label: string;
  active?: boolean;
  className?: string;
} & ConversationShareExportActionsProps;

export function ConversationShareExportIconDropdown({
  label,
  shareLabel,
  exportLabel,
  exportMarkdownLabel,
  exportImageLabel,
  copyMarkdownLabel,
  screenshotFullLabel,
  screenshotSelectLabel,
  active = false,
  className,
  onShare,
  onExport,
  screenshotLatestLabel,
  screenshotSelectLabel,
  onScreenshotLatest,
  onScreenshotSelect,
}: ConversationShareExportIconDropdownProps) {
  const [open, setOpen] = React.useState(false);
  const hasAction = Boolean(onShare || onExport || onScreenshotLatest || onScreenshotSelect);

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            "relative size-8 shrink-0 rounded-lg text-muted-foreground shadow-none hover:bg-muted hover:text-foreground after:absolute after:-inset-1.5 after:content-[''] md:after:hidden",
            active && "text-foreground",
            className,
          )}
          disabled={!hasAction}
          aria-label={label}
          title={label}
        >
          <Share2 className="size-4 stroke-[1.8]" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={8} className="min-w-40">
        <ConversationShareExportMenuItems
          shareLabel={shareLabel}
          exportLabel={exportLabel}
          onShare={onShare}
          onExport={onExport}
          screenshotLatestLabel={screenshotLatestLabel}
          screenshotSelectLabel={screenshotSelectLabel}
          onScreenshotLatest={onScreenshotLatest}
          onScreenshotSelect={onScreenshotSelect}
          onCloseMenu={() => setOpen(false)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ConversationShareExportSubmenu({
  label,
  shareLabel,
  exportLabel,
  onShare,
  onExport,
  screenshotLatestLabel,
  screenshotSelectLabel,
  onScreenshotLatest,
  onScreenshotSelect,
  onCloseMenu,
}: { label: string } & ConversationShareExportMenuItemsProps) {
  const hasAction = Boolean(onShare || onExport || onScreenshotLatest || onScreenshotSelect);
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger disabled={!hasAction}>
        <DropdownMenuItemIcon icon={Share2} className="text-current" />
        {label}
      </DropdownMenuSubTrigger>
      <DropdownMenuSubContent className="min-w-40 p-1.5">
        <ConversationShareExportMenuItems
          shareLabel={shareLabel}
          exportLabel={exportLabel}
          onShare={onShare}
          onExport={onExport}
          screenshotLatestLabel={screenshotLatestLabel}
          screenshotSelectLabel={screenshotSelectLabel}
          onScreenshotLatest={onScreenshotLatest}
          onScreenshotSelect={onScreenshotSelect}
          onCloseMenu={onCloseMenu}
        />
      </DropdownMenuSubContent>
    </DropdownMenuSub>
  );
}
