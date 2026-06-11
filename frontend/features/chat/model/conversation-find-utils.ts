export type ConversationFindSource = {
  key: string;
  content: string;
};

export type ConversationFindMatch = {
  messageKey: string;
  index: number;
};

export function findConversationMatches(
  messages: readonly ConversationFindSource[],
  query: string,
): ConversationFindMatch[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return [];
  }

  const matches: ConversationFindMatch[] = [];
  for (const message of messages) {
    const content = message.content.toLowerCase();
    let index = content.indexOf(normalizedQuery);
    while (index >= 0) {
      matches.push({ messageKey: message.key, index });
      index = content.indexOf(normalizedQuery, index + normalizedQuery.length);
    }
  }
  return matches;
}

export function nextConversationMatchIndex(
  currentIndex: number,
  matchCount: number,
  direction: "next" | "previous",
): number {
  if (matchCount <= 0) {
    return -1;
  }
  if (currentIndex < 0 || currentIndex >= matchCount) {
    return direction === "previous" ? matchCount - 1 : 0;
  }
  return direction === "previous"
    ? (currentIndex - 1 + matchCount) % matchCount
    : (currentIndex + 1) % matchCount;
}
