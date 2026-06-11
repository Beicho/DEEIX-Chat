export type SpeechAction = "play" | "pause" | "resume" | "unsupported";

export function normalizeSpeechText(content: string): string {
  return content
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[#>*_~|-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function resolveSpeechAction({
  supported,
  speaking,
  paused,
  messageKey,
  activeMessageKey,
}: {
  supported: boolean;
  speaking: boolean;
  paused: boolean;
  messageKey: string;
  activeMessageKey: string | null;
}): SpeechAction {
  if (!supported) {
    return "unsupported";
  }
  if (speaking && activeMessageKey === messageKey) {
    return paused ? "resume" : "pause";
  }
  return "play";
}
