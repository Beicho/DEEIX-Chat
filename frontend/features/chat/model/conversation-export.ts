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

export function resolveConversationExportFileName(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "conversation";
  return `conversation-${safeFileNamePart(title)}-${formatExportTimestamp(data.exportedAt)}.json`;
}

export function resolveConversationMarkdownExportFileName(data: ConversationExportDTO) {
  const title = data.conversation?.title?.trim() || data.conversation?.publicID || "conversation";
  return `conversation-${safeFileNamePart(title)}-${formatExportTimestamp(data.exportedAt)}.md`;
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
