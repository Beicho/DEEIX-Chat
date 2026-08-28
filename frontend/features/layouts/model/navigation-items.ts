import { createElement } from "react"
import { Bell, Bookmark, Gift, Swords } from "lucide-react"
import { Layers } from "@/components/animate-ui/icons/layers"
import { MessageCircleMore } from "@/components/animate-ui/icons/message-circle-more"
import { PlusIcon } from "@/components/ui/plus"
import { Search } from "@/components/animate-ui/icons/search"
import { Blend } from "@/components/animate-ui/icons/blend"
import type { NavigationItem } from "@/features/layouts/types/navigation"

function BellIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" | false }) {
  return createElement(Bell, { size, strokeWidth, className })
}

function BookmarkIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" | false }) {
  return createElement(Bookmark, { size, strokeWidth, className })
}

function GiftIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" | false }) {
  return createElement(Gift, { size, strokeWidth, className })
}

function SwordsIcon({ size = 18, strokeWidth = 1.6, className }: { size?: number; strokeWidth?: number; className?: string; animate?: "default" | false }) {
  return createElement(Swords, { size, strokeWidth, className })
}

export const NAVIGATION_ITEMS = [
  {
    id: "newChat",
    kind: "command",
    icon: PlusIcon,
    variant: "primary",
    group: "primary",
    shortcut: ["command", "shift", "O"],
  },
  {
    id: "search",
    kind: "command",
    icon: Search,
    group: "primary",
    shortcut: ["command", "K"],
  },
  {
    id: "recent",
    kind: "link",
    href: "/recent",
    icon: MessageCircleMore,
    group: "secondary",
  },
  {
    id: "checkin",
    kind: "link",
    href: "/checkin",
    icon: GiftIcon,
    group: "secondary",
  },
  {
    id: "arena",
    kind: "link",
    href: "/arena",
    icon: SwordsIcon,
    group: "secondary",
  },
  {
    id: "announcements",
    kind: "link",
    href: "/announcements",
    icon: BellIcon,
    group: "secondary",
  },
  {
    id: "bookmarks",
    kind: "link",
    href: "/bookmarks",
    icon: BookmarkIcon,
    group: "secondary",
  },
  {
    id: "files",
    kind: "link",
    href: "/files",
    icon: Layers,
    group: "secondary",
  },
  {
    id: "skillsPrompt",
    kind: "link",
    href: "/skills-prompt",
    icon: Blend,
    group: "secondary",
  },
] as const satisfies readonly NavigationItem[]
