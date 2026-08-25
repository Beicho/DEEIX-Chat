import { createElement } from "react"
import { Bell, Bookmark, Gift, Swords } from "lucide-react"
import { Layers } from "@/components/animate-ui/icons/layers"
import { MessageCircleMore } from "@/components/animate-ui/icons/message-circle-more"
import { PlusIcon } from "@/components/ui/plus"
import { Search } from "@/components/animate-ui/icons/search"
import { Blend } from "@/components/animate-ui/icons/blend"
import type { NavigationItem } from "@/features/layouts/types/navigation"

function BellIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" }) {
  return createElement(Bell, { size, strokeWidth, className })
}

function BookmarkIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" }) {
  return createElement(Bookmark, { size, strokeWidth, className })
}

function GiftIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" }) {
  return createElement(Gift, { size, strokeWidth, className })
}

function SwordsIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" }) {
  return createElement(Swords, { size, strokeWidth, className })
}

export const NAVIGATION_ITEMS = [
  {
    id: "newChat",
    title: "New chat",
    url: "#",
    icon: PlusIcon,
    variant: "primary",
    group: "primary",
    shortcut: ["command", "shift", "O"],
  },
  {
    id: "search",
    title: "Search",
    url: "#",
    icon: Search,
    group: "primary",
    shortcut: ["command", "K"],
  },
  {
    id: "recent",
    title: "Recent",
    url: "/recent",
    icon: MessageCircleMore,
    group: "secondary",
  },
  {
    id: "checkin",
    title: "Check-in",
    url: "/checkin",
    icon: GiftIcon,
    group: "secondary",
  },
  {
    id: "arena",
    title: "Arena",
    url: "/arena",
    icon: SwordsIcon,
    group: "secondary",
  },
  {
    id: "announcements",
    title: "Announcements",
    url: "/announcements",
    icon: BellIcon,
    group: "secondary",
  },
  {
    id: "bookmarks",
    title: "Bookmarks",
    url: "/bookmarks",
    icon: BookmarkIcon,
    group: "secondary",
  },
  {
    id: "files",
    title: "Files",
    url: "/files",
    icon: Layers,
    group: "secondary",
  },
  {
    id: "skillsPrompt",
    title: "Skills & Prompts",
    url: "/skills-prompt",
    icon: Blend,
    group: "secondary",
  },
] as const satisfies readonly NavigationItem[]
