"use client";

import type { ConversationExportDTO } from "@/shared/api/conversation.types";

function safeFileNamePart(value: string) {
  const normalized = value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, "-")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
  return normalized || "conversation";
}

function formatExportTimestamp(value: string) {
  const date = new Date(value);
  const source = Number.isNaN(date.getTime()) ? new Date() : date;
  const pad = (part: number) => String(part).padStart(2, "0");
  return [
    source.getFullYear(),
    pad(source.getMonth() + 1),
    pad(source.getDate()),
    "-",
    pad(source.getHours()),
    pad(source.getMinutes()),
    pad(source.getSeconds()),
  ].join("");
}

function resolveConversationExportFileName(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "conversation";
  return `conversation-${safeFileNamePart(title)}-${formatExportTimestamp(data.exportedAt)}.json`;
}

export function resolveConversationMarkdownExportFileName(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "conversation";
  return `conversation-${safeFileNamePart(title)}-${formatExportTimestamp(data.exportedAt)}.md`;
}

export function resolveConversationImageExportFileName(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "conversation";
  return `conversation-${safeFileNamePart(title)}-${formatExportTimestamp(data.exportedAt)}.png`;
}

function orderedExportMessages(data: ConversationExportDTO) {
  const defaultIDs = new Set(data.defaultMessagePublicIDs?.filter(Boolean) ?? []);
  const source = defaultIDs.size > 0
    ? data.messages.filter((message) => defaultIDs.has(message.publicID))
    : data.messages;
  return [...source].sort((left, right) => {
    const leftTime = new Date(left.createdAt).getTime();
    const rightTime = new Date(right.createdAt).getTime();
    if (Number.isFinite(leftTime) && Number.isFinite(rightTime) && leftTime !== rightTime) {
      return leftTime - rightTime;
    }
    return left.id - right.id;
  });
}

function parseAttachmentNames(raw: string): string[] {
  if (!raw.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map((item) => {
        if (!item || typeof item !== "object") {
          return "";
        }
        const record = item as Record<string, unknown>;
        return String(record.file_name ?? record.fileName ?? "").trim();
      })
      .filter(Boolean);
  } catch {
    return [];
  }
}

function roleLabel(role: string) {
  switch (role.trim().toLowerCase()) {
    case "assistant":
      return "Assistant";
    case "system":
      return "System";
    case "user":
      return "User";
    default:
      return "Message";
  }
}

function escapeMarkdownTitle(value: string) {
  return value.replace(/([\\`*_{}\[\]()#+\-.!|>])/g, "\\$1");
}

export function conversationExportToMarkdown(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "Conversation";
  const lines: string[] = [
    `# ${escapeMarkdownTitle(title)}`,
    "",
    `- Exported at: ${data.exportedAt}`,
    `- Conversation ID: ${data.conversation?.publicID || ""}`,
    "",
  ];

  for (const message of orderedExportMessages(data)) {
    const label = roleLabel(message.role);
    lines.push(`## ${label}`);
    if (message.platformModelName?.trim()) {
      lines.push(``);
      lines.push(`Model: ${message.platformModelName.trim()}`);
    }
    const attachmentNames = parseAttachmentNames(message.attachments);
    if (attachmentNames.length > 0) {
      lines.push("");
      lines.push("Attachments:");
      for (const name of attachmentNames) {
        lines.push(`- ${name}`);
      }
    }
    lines.push("");
    lines.push(message.content.trim() || "_No text content_");
    lines.push("");
  }

  return `${lines.join("\n").replace(/\n{3,}/g, "\n\n").trimEnd()}\n`;
}

export function downloadConversationExport(data: ConversationExportDTO) {
  const blob = new Blob([`${JSON.stringify(data, null, 2)}\n`], {
    type: "application/json;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = resolveConversationExportFileName(data);
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export function downloadConversationMarkdownExport(data: ConversationExportDTO) {
  const blob = new Blob([conversationExportToMarkdown(data)], {
    type: "text/markdown;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = resolveConversationMarkdownExportFileName(data);
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export async function copyConversationMarkdownExport(data: ConversationExportDTO) {
  await navigator.clipboard.writeText(conversationExportToMarkdown(data));
}

export type ConversationImageExportLabels = {
  titleFallback: string;
  exportedAt: string;
  conversationID: string;
  roleAssistant: string;
  roleSystem: string;
  roleUser: string;
  roleMessage: string;
  model: string;
  attachments: string;
  noTextContent: string;
  truncated: string;
  watermark: string;
};

type ConversationImageTextBlock = {
  align: "left" | "right";
  role: string;
  model: string;
  lines: string[];
  attachments: string[];
  height: number;
};

type ConversationImagePalette = {
  background: string;
  foreground: string;
  card: string;
  cardForeground: string;
  muted: string;
  mutedForeground: string;
  primary: string;
  primaryForeground: string;
  border: string;
};

const IMAGE_EXPORT_WIDTH = 860;
const IMAGE_EXPORT_PADDING = 40;
const IMAGE_EXPORT_BUBBLE_PADDING_X = 18;
const IMAGE_EXPORT_BUBBLE_PADDING_Y = 14;
const IMAGE_EXPORT_BUBBLE_MAX_WIDTH = IMAGE_EXPORT_WIDTH - IMAGE_EXPORT_PADDING * 2;
const IMAGE_EXPORT_LINE_HEIGHT = 24;
const IMAGE_EXPORT_META_LINE_HEIGHT = 18;
const IMAGE_EXPORT_MAX_CANVAS_SIDE = 32000;

function getResolvedCSSValue(styles: CSSStyleDeclaration, name: string, fallback: string) {
  return styles.getPropertyValue(name).trim() || fallback;
}

function getImageExportPalette(): ConversationImagePalette {
  const styles = getComputedStyle(document.documentElement);
  return {
    background: getResolvedCSSValue(styles, "--background", "Canvas"),
    foreground: getResolvedCSSValue(styles, "--foreground", "CanvasText"),
    card: getResolvedCSSValue(styles, "--card", "Canvas"),
    cardForeground: getResolvedCSSValue(styles, "--card-foreground", "CanvasText"),
    muted: getResolvedCSSValue(styles, "--muted", "Canvas"),
    mutedForeground: getResolvedCSSValue(styles, "--muted-foreground", "GrayText"),
    primary: getResolvedCSSValue(styles, "--primary", "Highlight"),
    primaryForeground: getResolvedCSSValue(styles, "--primary-foreground", "HighlightText"),
    border: getResolvedCSSValue(styles, "--border", "GrayText"),
  };
}

function getCanvasFontFamily() {
  const styles = getComputedStyle(document.documentElement);
  return getResolvedCSSValue(styles, "--font-chat", styles.fontFamily || "sans-serif");
}

function setCanvasFont(
  context: CanvasRenderingContext2D,
  size: number,
  family: string,
  weight: number,
) {
  context.font = `${weight} ${size}px ${family}`;
}

function splitLongCanvasToken(token: string): string[] {
  return Array.from(token);
}

function wrapCanvasText(
  context: CanvasRenderingContext2D,
  text: string,
  maxWidth: number,
): string[] {
  const normalized = text.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const result: string[] = [];
  for (const paragraph of normalized.split("\n")) {
    if (!paragraph.trim()) {
      result.push("");
      continue;
    }
    const words = paragraph.split(/(\s+)/).filter((part) => part.length > 0);
    let line = "";
    for (const word of words) {
      const candidate = line ? `${line}${word}` : word.trimStart();
      if (context.measureText(candidate).width <= maxWidth) {
        line = candidate;
        continue;
      }
      if (line.trim()) {
        result.push(line.trimEnd());
        line = "";
      }
      if (context.measureText(word).width <= maxWidth) {
        line = word.trimStart();
        continue;
      }
      for (const char of splitLongCanvasToken(word)) {
        const charCandidate = line ? `${line}${char}` : char;
        if (context.measureText(charCandidate).width > maxWidth && line) {
          result.push(line);
          line = char;
        } else {
          line = charCandidate;
        }
      }
    }
    if (line.trim()) {
      result.push(line.trimEnd());
    }
  }
  return result.length > 0 ? result : [""];
}

function drawRoundedRect(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
) {
  const resolvedRadius = Math.min(radius, width / 2, height / 2);
  context.beginPath();
  context.moveTo(x + resolvedRadius, y);
  context.lineTo(x + width - resolvedRadius, y);
  context.quadraticCurveTo(x + width, y, x + width, y + resolvedRadius);
  context.lineTo(x + width, y + height - resolvedRadius);
  context.quadraticCurveTo(x + width, y + height, x + width - resolvedRadius, y + height);
  context.lineTo(x + resolvedRadius, y + height);
  context.quadraticCurveTo(x, y + height, x, y + height - resolvedRadius);
  context.lineTo(x, y + resolvedRadius);
  context.quadraticCurveTo(x, y, x + resolvedRadius, y);
  context.closePath();
}

function resolveConversationImageRoleLabel(role: string, labels: ConversationImageExportLabels) {
  switch (role.trim().toLowerCase()) {
    case "assistant":
      return labels.roleAssistant;
    case "system":
      return labels.roleSystem;
    case "user":
      return labels.roleUser;
    default:
      return labels.roleMessage;
  }
}

function measureConversationImageBlocks(
  context: CanvasRenderingContext2D,
  data: ConversationExportDTO,
  labels: ConversationImageExportLabels,
  fontFamily: string,
) {
  setCanvasFont(context, 16, fontFamily, 400);
  return orderedExportMessages(data).map((message): ConversationImageTextBlock => {
    const lines = wrapCanvasText(
      context,
      message.content.trim() || labels.noTextContent,
      IMAGE_EXPORT_BUBBLE_MAX_WIDTH - IMAGE_EXPORT_BUBBLE_PADDING_X * 2,
    );
    const attachments = parseAttachmentNames(message.attachments);
    const attachmentHeight = attachments.length > 0
      ? IMAGE_EXPORT_META_LINE_HEIGHT * (attachments.length + 1) + 8
      : 0;
    const modelHeight = message.platformModelName?.trim() ? IMAGE_EXPORT_META_LINE_HEIGHT + 6 : 0;
    return {
      align: message.role === "user" ? "right" : "left",
      role: resolveConversationImageRoleLabel(message.role, labels),
      model: message.platformModelName?.trim() || "",
      lines,
      attachments,
      height:
        22 +
        IMAGE_EXPORT_BUBBLE_PADDING_Y * 2 +
        lines.length * IMAGE_EXPORT_LINE_HEIGHT +
        modelHeight +
        attachmentHeight +
        18,
    };
  });
}

function drawWrappedText(
  context: CanvasRenderingContext2D,
  lines: string[],
  x: number,
  y: number,
  lineHeight: number,
) {
  lines.forEach((line, index) => {
    if (line) {
      context.fillText(line, x, y + index * lineHeight);
    }
  });
}

function triggerConversationImageDownload(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

async function maybeShareConversationImage(blob: Blob, fileName: string, title: string) {
  if (typeof navigator === "undefined" || typeof navigator.share !== "function" || typeof File === "undefined") {
    return false;
  }
  if (typeof window !== "undefined" && !window.matchMedia("(pointer: coarse)").matches) {
    return false;
  }
  const file = new File([blob], fileName, { type: "image/png" });
  const shareData = { title, files: [file] } as ShareData;
  if (typeof navigator.canShare === "function" && !navigator.canShare(shareData)) {
    return false;
  }
  try {
    await navigator.share(shareData);
    return true;
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") {
      return true;
    }
    return false;
  }
}

async function canvasToPNGBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  const blob = await new Promise<Blob | null>((resolve) => {
    canvas.toBlob(resolve, "image/png");
  });
  if (!blob) {
    throw new Error("Unable to create PNG image.");
  }
  return blob;
}

export async function createConversationImageExportBlob(
  data: ConversationExportDTO,
  labels: ConversationImageExportLabels,
): Promise<Blob> {
  if (typeof document === "undefined") {
    throw new Error("Image export requires a browser.");
  }

  await document.fonts?.ready.catch(() => undefined);

  const palette = getImageExportPalette();
  const fontFamily = getCanvasFontFamily();
  const measureCanvas = document.createElement("canvas");
  const measureContext = measureCanvas.getContext("2d");
  if (!measureContext) {
    throw new Error("Canvas is not available.");
  }

  const title = data.conversation?.title?.trim() || data.conversation?.publicID || labels.titleFallback;
  setCanvasFont(measureContext, 28, fontFamily, 700);
  const titleLines = wrapCanvasText(measureContext, title, IMAGE_EXPORT_WIDTH - IMAGE_EXPORT_PADDING * 2);
  const blocks = measureConversationImageBlocks(measureContext, data, labels, fontFamily);
  const logicalPixelRatio = Math.min(Math.max(window.devicePixelRatio || 1, 1), 2);
  const maxLogicalHeight = Math.floor(IMAGE_EXPORT_MAX_CANVAS_SIDE / logicalPixelRatio);
  const headerHeight = 38 + titleLines.length * 34 + 56;
  let totalHeight = IMAGE_EXPORT_PADDING + headerHeight + IMAGE_EXPORT_PADDING;
  const visibleBlocks: ConversationImageTextBlock[] = [];
  let truncated = false;
  for (const block of blocks) {
    if (totalHeight + block.height + 64 > maxLogicalHeight) {
      truncated = true;
      break;
    }
    visibleBlocks.push(block);
    totalHeight += block.height;
  }
  if (truncated) {
    totalHeight += 54;
  }
  totalHeight += 42;

  const canvas = document.createElement("canvas");
  canvas.width = Math.ceil(IMAGE_EXPORT_WIDTH * logicalPixelRatio);
  canvas.height = Math.ceil(totalHeight * logicalPixelRatio);
  canvas.style.width = `${IMAGE_EXPORT_WIDTH}px`;
  canvas.style.height = `${totalHeight}px`;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Canvas is not available.");
  }
  context.scale(logicalPixelRatio, logicalPixelRatio);
  context.textBaseline = "top";
  context.fillStyle = palette.background;
  context.fillRect(0, 0, IMAGE_EXPORT_WIDTH, totalHeight);

  let y = IMAGE_EXPORT_PADDING;
  setCanvasFont(context, 28, fontFamily, 700);
  context.fillStyle = palette.foreground;
  drawWrappedText(context, titleLines, IMAGE_EXPORT_PADDING, y, 34);
  y += titleLines.length * 34 + 16;

  setCanvasFont(context, 12, fontFamily, 500);
  context.fillStyle = palette.mutedForeground;
  context.fillText(`${labels.exportedAt}: ${data.exportedAt}`, IMAGE_EXPORT_PADDING, y);
  y += IMAGE_EXPORT_META_LINE_HEIGHT;
  context.fillText(`${labels.conversationID}: ${data.conversation?.publicID || ""}`, IMAGE_EXPORT_PADDING, y);
  y += 38;

  for (const block of visibleBlocks) {
    const bubbleX = block.align === "right"
      ? IMAGE_EXPORT_WIDTH - IMAGE_EXPORT_PADDING - IMAGE_EXPORT_BUBBLE_MAX_WIDTH
      : IMAGE_EXPORT_PADDING;
    const bubbleY = y + 22;
    const bubbleHeight = block.height - 40;
    setCanvasFont(context, 12, fontFamily, 700);
    context.fillStyle = palette.mutedForeground;
    context.textAlign = block.align === "right" ? "right" : "left";
    context.fillText(
      block.role,
      block.align === "right" ? IMAGE_EXPORT_WIDTH - IMAGE_EXPORT_PADDING : IMAGE_EXPORT_PADDING,
      y,
    );
    context.textAlign = "left";

    drawRoundedRect(context, bubbleX, bubbleY, IMAGE_EXPORT_BUBBLE_MAX_WIDTH, bubbleHeight, 18);
    context.fillStyle = block.align === "right" ? palette.primary : palette.card;
    context.fill();
    context.strokeStyle = block.align === "right" ? palette.primary : palette.border;
    context.lineWidth = 1;
    context.stroke();

    let textY = bubbleY + IMAGE_EXPORT_BUBBLE_PADDING_Y;
    if (block.model) {
      setCanvasFont(context, 12, fontFamily, 600);
      context.fillStyle = block.align === "right" ? palette.primaryForeground : palette.mutedForeground;
      context.fillText(`${labels.model}: ${block.model}`, bubbleX + IMAGE_EXPORT_BUBBLE_PADDING_X, textY);
      textY += IMAGE_EXPORT_META_LINE_HEIGHT + 6;
    }
    setCanvasFont(context, 16, fontFamily, 400);
    context.fillStyle = block.align === "right" ? palette.primaryForeground : palette.cardForeground;
    drawWrappedText(
      context,
      block.lines,
      bubbleX + IMAGE_EXPORT_BUBBLE_PADDING_X,
      textY,
      IMAGE_EXPORT_LINE_HEIGHT,
    );
    textY += block.lines.length * IMAGE_EXPORT_LINE_HEIGHT;
    if (block.attachments.length > 0) {
      textY += 8;
      setCanvasFont(context, 12, fontFamily, 600);
      context.fillStyle = block.align === "right" ? palette.primaryForeground : palette.mutedForeground;
      context.fillText(labels.attachments, bubbleX + IMAGE_EXPORT_BUBBLE_PADDING_X, textY);
      textY += IMAGE_EXPORT_META_LINE_HEIGHT;
      setCanvasFont(context, 12, fontFamily, 400);
      for (const attachment of block.attachments) {
        context.fillText(`- ${attachment}`, bubbleX + IMAGE_EXPORT_BUBBLE_PADDING_X, textY);
        textY += IMAGE_EXPORT_META_LINE_HEIGHT;
      }
    }
    y += block.height;
  }

  if (truncated) {
    drawRoundedRect(context, IMAGE_EXPORT_PADDING, y, IMAGE_EXPORT_BUBBLE_MAX_WIDTH, 38, 14);
    context.fillStyle = palette.muted;
    context.fill();
    setCanvasFont(context, 13, fontFamily, 500);
    context.fillStyle = palette.mutedForeground;
    context.fillText(labels.truncated, IMAGE_EXPORT_PADDING + 16, y + 11);
    y += 54;
  }

  setCanvasFont(context, 12, fontFamily, 600);
  context.fillStyle = palette.mutedForeground;
  context.textAlign = "right";
  context.fillText(labels.watermark, IMAGE_EXPORT_WIDTH - IMAGE_EXPORT_PADDING, totalHeight - IMAGE_EXPORT_PADDING);
  context.textAlign = "left";

  return canvasToPNGBlob(canvas);
}

export async function downloadConversationImageExport(
  data: ConversationExportDTO,
  labels: ConversationImageExportLabels,
) {
  const blob = await createConversationImageExportBlob(data, labels);
  const fileName = resolveConversationImageExportFileName(data);
  const title = data.conversation?.title?.trim() || labels.titleFallback;
  const shared = await maybeShareConversationImage(blob, fileName, title);
  if (!shared) {
    triggerConversationImageDownload(blob, fileName);
  }
}
