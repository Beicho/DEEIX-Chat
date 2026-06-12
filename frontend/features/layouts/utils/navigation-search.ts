import type { ConversationDTO, ConversationSearchResultDTO } from "@/shared/api/conversation.types"
import type { ConversationSearchResult } from "@/features/layouts/types/navigation"
import {
  conversationMatchesSearch,
  conversationSearchText,
  normalizeConversationSearchText,
} from "@/shared/lib/conversation-search"

export function normalizeSearchText(value: string) {
  return normalizeConversationSearchText(value)
}

export function toConversationSearchResult(item: ConversationDTO, untitled = "New chat"): ConversationSearchResult {
  return {
    resultID: item.publicID,
    publicID: item.publicID,
    title: item.title?.trim() || untitled,
    searchText: conversationSearchText(item),
    href: `/chat?conversation_id=${item.publicID}`,
    updatedAt: item.updatedAt,
  }
}

export function toServerConversationSearchResult(
  item: ConversationSearchResultDTO,
  untitled = "New chat",
): ConversationSearchResult {
  const conversation = item.conversation
  const title = conversation.title?.trim() || untitled
  const snippet = item.snippet?.trim() || ""
  const messagePublicID = item.messagePublicID?.trim() || ""

  return {
    resultID: messagePublicID ? `${conversation.publicID}:${messagePublicID}` : `${conversation.publicID}:title`,
    publicID: conversation.publicID,
    title,
    searchText: [conversationSearchText(conversation), snippet, messagePublicID].filter(Boolean).join(" "),
    href: `/chat?conversation_id=${conversation.publicID}`,
    snippet,
    messagePublicID,
    matchedTitle: item.matchedTitle,
    updatedAt: item.matchedAt || conversation.updatedAt,
  }
}

export function filterConversationSearchResults(
  items: readonly ConversationDTO[],
  query: string,
  maxResults?: number,
  untitled?: string,
) {
  const normalizedQuery = normalizeSearchText(query)
  const results = items
    .filter((item) => conversationMatchesSearch(item, normalizedQuery))
    .map((item) => toConversationSearchResult(item, untitled))

  return typeof maxResults === "number" ? results.slice(0, maxResults) : results
}

export function formatUpdatedAtLabel(value: string, locale = "en-US", todayLabel = "Today", yesterdayLabel = "Yesterday") {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ""
  }

  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const diffDays = Math.round((today.getTime() - target.getTime()) / 86400000)

  if (diffDays === 0) {
    return todayLabel
  }

  if (diffDays === 1) {
    return yesterdayLabel
  }

  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
  }).format(date)
}
