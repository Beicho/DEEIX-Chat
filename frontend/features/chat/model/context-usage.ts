export const CONTEXT_USAGE_APPROX_CHARS_PER_TOKEN = 4;
export const CONTEXT_USAGE_DEFAULT_WINDOW_TOKENS = 128000;

export type ContextUsageTone = "default" | "warning" | "danger";

export function estimateConversationTokens(contents: readonly string[]): number {
  const totalCharacters = contents.reduce((sum, content) => sum + content.length, 0);
  return Math.ceil(totalCharacters / CONTEXT_USAGE_APPROX_CHARS_PER_TOKEN);
}

export function resolveContextUsageTone(ratio: number): ContextUsageTone {
  if (ratio >= 0.9) {
    return "danger";
  }
  if (ratio >= 0.75) {
    return "warning";
  }
  return "default";
}

export function resolveContextUsageRatio(estimatedTokens: number, windowTokens = CONTEXT_USAGE_DEFAULT_WINDOW_TOKENS): number {
  if (!Number.isFinite(windowTokens) || windowTokens <= 0) {
    return 0;
  }
  return Math.min(1, Math.max(0, estimatedTokens / windowTokens));
}
