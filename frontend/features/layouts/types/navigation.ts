import type * as React from "react"
import type { LucideIcon } from "lucide-react"

export type ShortcutKey = "shift" | "command" | string

export type NavigationIconProps = {
  size?: number
  strokeWidth?: number
  className?: string
  animate?: "default" | false
  "aria-hidden"?: boolean
}

export type NavigationItem = {
  id: "newChat" | "search" | "recent" | "files" | "knowledgeBases" | "skillsPrompt" | "announcements" | "bookmarks" | "checkin" | "arena"
  title: string
  url: string
  icon: React.ComponentType<NavigationIconProps>
  group: "primary" | "secondary"
  variant?: "primary"
  shortcut?: readonly ShortcutKey[]
  external?: boolean
}

export type ConversationSearchResult = {
  resultID?: string
  publicID: string
  title: string
  searchText?: string
  href: string
  snippet?: string
  messagePublicID?: string
  matchedTitle?: boolean
  updatedAt: string
  projectName?: string
  status?: string
}

export type SidebarConversationItem = {
  publicID: string
  title: string
  url: string
  starred?: boolean
  shareActive?: boolean
  labelsJSON?: string
}

export type SidebarConversationRenameTarget = {
  publicID: string
  currentTitle: string
} | null

export type SidebarConversationDeleteTarget = {
  publicID: string
  title: string
} | null

export type SidebarConversationStarAction = {
  label: string
  icon: LucideIcon
  onSelect: (publicID: string) => void
}

export type SidebarConversationProjectMenu = {
  label: string
  unassignedLabel: string
  currentProjectID?: string
  projects: Array<{
    publicID: string
    name: string
  }>
  onSelect: (publicID: string, projectID?: string) => void
}
