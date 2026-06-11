type ShareMetadataMessage = {
  role: string;
  content: string;
};

const DESCRIPTION_LIMIT = 160;

function stripMarkdown(value: string): string {
  return value
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[[^\]]*]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)]\([^)]*\)/g, "$1")
    .replace(/[#>*_~|[\]()`-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function resolveShareCanonicalPath(shareID: string): string {
  const normalized = shareID.trim();
  return normalized ? `/share/${encodeURIComponent(normalized)}` : "/share";
}

export function buildShareMetadataDescription({
  fallback,
  messages,
}: {
  fallback: string;
  messages: ShareMetadataMessage[];
}): string {
  const source =
    messages.find((message) => message.role === "user" && stripMarkdown(message.content)) ??
    messages.find((message) => message.role === "assistant" && stripMarkdown(message.content));
  const description = source ? stripMarkdown(source.content) : fallback.trim();
  if (description.length <= DESCRIPTION_LIMIT) {
    return description;
  }
  return description.slice(0, DESCRIPTION_LIMIT - 1).trimEnd() + "…";
}
