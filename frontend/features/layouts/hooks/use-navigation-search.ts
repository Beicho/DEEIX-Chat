"use client"

import * as React from "react"
import { useRouter } from "next/navigation"

import { searchConversations } from "@/shared/api/conversation"
import { useAuthSession } from "@/shared/auth/auth-session-context"
import {
  filterConversationSearchResults,
  toServerConversationSearchResult,
} from "@/features/layouts/utils/navigation-search"
import { hasPlatformModifierKey } from "@/shared/lib/platform-shortcuts"
import type { ConversationDTO } from "@/shared/api/conversation.types"
import type { ConversationSearchResult } from "@/features/layouts/types/navigation"

type UseNavigationSearchOptions = {
  items: readonly ConversationDTO[]
  maxResults?: number
  untitled?: string
}

const SERVER_SEARCH_MIN_QUERY_LENGTH = 2
const SERVER_SEARCH_DEBOUNCE_MS = 180

export function useNavigationSearch({ items, maxResults, untitled }: UseNavigationSearchOptions) {
  const { accessToken } = useAuthSession()
  const router = useRouter()
  const [open, setOpen] = React.useState(false)
  const [query, setQuery] = React.useState("")
  const [serverResults, setServerResults] = React.useState<ConversationSearchResult[]>([])
  const [serverLoading, setServerLoading] = React.useState(false)
  const [serverFailed, setServerFailed] = React.useState(false)

  React.useEffect(() => {
    if (!open) {
      setQuery("")
    }
  }, [open])

  const normalizedQuery = query.trim()
  const shouldUseServerSearch = open && normalizedQuery.length >= SERVER_SEARCH_MIN_QUERY_LENGTH

  const localResults = React.useMemo(
    () => filterConversationSearchResults(items, query, maxResults, untitled),
    [items, maxResults, query, untitled],
  )

  React.useEffect(() => {
    if (!shouldUseServerSearch) {
      setServerResults([])
      setServerLoading(false)
      setServerFailed(false)
      return
    }

    let cancelled = false
    setServerLoading(true)
    setServerFailed(false)
    const timer = window.setTimeout(() => {
      void searchConversations(accessToken, normalizedQuery, {
        page: 1,
        pageSize: maxResults ?? 8,
      })
        .then((data) => {
          if (cancelled) {
            return
          }
          setServerResults((data.results ?? []).map((item) => toServerConversationSearchResult(item, untitled)))
        })
        .catch(() => {
          if (cancelled) {
            return
          }
          setServerResults([])
          setServerFailed(true)
        })
        .finally(() => {
          if (!cancelled) {
            setServerLoading(false)
          }
        })
    }, SERVER_SEARCH_DEBOUNCE_MS)

    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [accessToken, maxResults, normalizedQuery, shouldUseServerSearch, untitled])

  const results = shouldUseServerSearch && !serverFailed ? serverResults : localResults

  const openSearch = React.useCallback(() => {
    React.startTransition(() => {
      setOpen(true)
    })
  }, [])

  const selectResult = React.useCallback((href: string) => {
    setOpen(false)
    router.push(href)
  }, [router])

  return {
    open,
    setOpen,
    query,
    setQuery,
    results,
    loading: serverLoading,
    openSearch,
    selectResult,
  }
}

export function useNavigationShortcuts({
  onCreateConversation,
  onOpenSearch,
}: {
  onCreateConversation: () => void
  onOpenSearch: () => void
}) {
  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing || event.key === "Process") {
        return
      }

      if (!hasPlatformModifierKey(event)) {
        return
      }

      const normalizedKey = event.key.toLowerCase()
      if (event.shiftKey && normalizedKey === "o") {
        event.preventDefault()
        onCreateConversation()
        return
      }

      if (!event.shiftKey && normalizedKey === "k") {
        event.preventDefault()
        onOpenSearch()
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [onCreateConversation, onOpenSearch])
}
