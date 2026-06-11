export type ConversationNativeShareInput = {
  title: string;
  text: string;
  url: string;
};

export type ConversationNativeShareData = {
  title: string;
  text?: string;
  url: string;
};

export function buildConversationNativeShareData(input: ConversationNativeShareInput): ConversationNativeShareData {
  const title = input.title.trim() || "DEEIX Chat";
  const text = input.text.trim();
  const url = input.url.trim();
  return text ? { title, text, url } : { title, url };
}

export function isNativeShareAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}
