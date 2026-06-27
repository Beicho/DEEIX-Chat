"use client"

import * as React from "react"
import { usePathname, useRouter, useSearchParams } from "next/navigation"
import { AnimatePresence, motion, type Transition } from "motion/react"
import { FileText, PencilLine, Plus, RefreshCw, Star, StarOff, Trash, Upload, type LucideIcon } from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { Ellipsis } from "@/components/animate-ui/icons/ellipsis"
import { FolderArchiveIcon } from "@/components/ui/folder-archive"
import { FolderOpenIcon } from "@/components/ui/folder-open"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Spinner } from "@/components/ui/spinner"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuItemIcon,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { CenteredEmptyState } from "@/components/ui/empty-state"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar"
import {
  ConversationShareDialog,
  sharePatchFromDTO,
} from "@/features/chat/components/sections/conversation-share-dialog"
import { useConversationExportAction } from "@/features/chat/hooks/use-conversation-export-action"
import { useChatSession } from "@/features/chat/context/chat-session-context"
import { DeleteFilesOption } from "@/features/recent/components/delete-files-option"
import { useChatPreferences } from "@/features/settings/hooks/use-chat-preferences"
import { useActiveSidebarConversation } from "@/features/layouts/hooks/use-active-sidebar-conversation"
import { SidebarConversationItem } from "@/features/layouts/components/navigation/sidebar-conversation-item"
import type {
  SidebarConversationDeleteTarget,
  SidebarConversationRenameTarget,
} from "@/features/layouts/types/navigation"
import { useSidebarRecents } from "@/features/recent/context/sidebar-recents-context"
import { mergeUniqueByPublicID, sortByUpdatedAtDesc, upsertByPublicID, removeByPublicID } from "@/features/recent/utils/conversation-list"
import { listConversations } from "@/shared/api/conversation"
import type { ConversationDTO } from "@/shared/api/conversation.types"
import { listFiles, uploadFile } from "@/shared/api/file"
import type { FileObjectDTO } from "@/shared/api/file.types"
import {
  addProjectKnowledgeDocuments,
  deleteProjectKnowledgeDocument,
  listProjectKnowledgeDocuments,
  reindexProjectKnowledgeDocument,
} from "@/shared/api/project-knowledge"
import type { ProjectKnowledgeDocumentDTO } from "@/shared/api/project-knowledge.types"
import { resolveAccessToken } from "@/shared/auth/resolve-access-token"
import { resolveLocalizedErrorMessage } from "@/i18n/resolve-error-message"
import { cn } from "@/lib/utils"

type ProjectDraft = {
  publicID?: string
  name: string
  systemPrompt: string
}

type ProjectActionTarget = {
  publicID?: string
  name: string
}

type KnowledgeTarget = {
  publicID: string
  name: string
}

type ProjectConversationState = {
  items: ConversationDTO[]
  loading: boolean
  loaded: boolean
  error: boolean
}

const PROJECT_CONVERSATION_PAGE_SIZE = 30
const PROJECT_TREE_ACCORDION_TRANSITION: Transition = {
  duration: 0.26,
  ease: [0.22, 1, 0.36, 1],
}
const PROJECT_DIALOG_LAYOUT_TRANSITION = {
  layout: {
    duration: 0.22,
    ease: [0.16, 1, 0.3, 1] as const,
  },
}
const PROJECT_TREE_ACCORDION_MASK_STYLE = {
  maskImage: "linear-gradient(black var(--mask-stop), transparent var(--mask-stop))",
  WebkitMaskImage: "linear-gradient(black var(--mask-stop), transparent var(--mask-stop))",
  overflow: "hidden",
} satisfies React.CSSProperties
const PROJECT_CREATE_ACTION_CLASS =
  "static size-8 shrink-0 opacity-100 transition-[background-color,color,opacity,transform] duration-150 md:size-7"

type ProjectFolderIconHandle = {
  startAnimation: () => void
  stopAnimation: () => void
}

function ProjectGroupHeader({
  title,
  createLabel,
  onCreate,
}: {
  title: string
  createLabel: string
  onCreate: () => void
}) {
  return (
    <div className="group/project-create flex h-9 items-center md:h-8">
      <SidebarGroupLabel className="min-w-0 flex-1 shrink pr-2">{title}</SidebarGroupLabel>
      <SidebarGroupAction
        type="button"
        aria-label={createLabel}
        className={PROJECT_CREATE_ACTION_CLASS}
        onClick={onCreate}
      >
        <Plus />
      </SidebarGroupAction>
    </div>
  )
}

function ProjectTreeButton({
  active,
  contentID,
  expanded,
  name,
  onToggleExpanded,
}: {
  active: boolean
  contentID: string
  expanded: boolean
  name: string
  onToggleExpanded: () => void
}) {
  const iconRef = React.useRef<ProjectFolderIconHandle>(null)

  return (
    <button
      type="button"
      className={cn(
        "flex h-9 w-full min-w-0 items-center rounded-md text-sm outline-hidden ring-sidebar-ring transition-colors focus-visible:ring-2 md:h-8",
        active
          ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
          : "text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
        "pr-16",
      )}
      aria-controls={contentID}
      aria-expanded={expanded}
      aria-label={name}
      onClick={(event) => {
        event.preventDefault()
        event.stopPropagation()
        onToggleExpanded()
      }}
      onMouseEnter={() => {
        iconRef.current?.startAnimation()
      }}
      onMouseLeave={() => {
        iconRef.current?.stopAnimation()
      }}
    >
      <span className="flex h-9 w-9 shrink-0 items-center justify-center md:h-8 md:w-8">
        {expanded ? (
          <FolderOpenIcon
            ref={iconRef}
            size={18}
            strokeWidth={1.5}
            className="flex size-4.5 shrink-0 items-center justify-center text-current"
          />
        ) : (
          <FolderArchiveIcon
            ref={iconRef}
            size={18}
            strokeWidth={1.5}
            className="flex size-4.5 shrink-0 items-center justify-center text-current"
          />
        )}
      </span>
      <span className="min-w-0 flex-1 truncate text-left">{name}</span>
    </button>
  )
}

type ProjectInlineActionProps = React.ComponentPropsWithoutRef<"button"> & {
  label: string
  visible: boolean
  onHoverChange?: (hovered: boolean) => void
}

const ProjectInlineAction = React.forwardRef<HTMLButtonElement, ProjectInlineActionProps>(function ProjectInlineAction({
  label,
  visible,
  onHoverChange,
  tabIndex,
  onClick,
  onMouseEnter,
  onMouseLeave,
  className,
  children,
  ...props
}, ref) {
  return (
    <button
      {...props}
      ref={ref}
      type="button"
      aria-label={label}
      title={label}
      tabIndex={tabIndex ?? (visible ? undefined : -1)}
      className={cn(
        "pointer-events-auto absolute top-0 z-10 flex h-8 w-8 items-center justify-center rounded-md text-sidebar-foreground opacity-100 transition-[background-color,color,opacity] duration-150 after:absolute after:-inset-1.5 after:content-[''] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground md:pointer-events-none md:opacity-0 md:after:hidden md:group-hover/project-row:pointer-events-auto md:group-hover/project-row:opacity-100 md:group-focus-within/project-row:pointer-events-auto md:group-focus-within/project-row:opacity-100",
        visible && "pointer-events-auto opacity-100",
        className,
      )}
      onMouseEnter={(event) => {
        onMouseEnter?.(event)
        onHoverChange?.(true)
      }}
      onMouseLeave={(event) => {
        onMouseLeave?.(event)
        onHoverChange?.(false)
      }}
      onClick={(event) => {
        onClick?.(event)
        event.preventDefault()
        event.stopPropagation()
      }}
    >
      {children}
    </button>
  )
})

function ProjectActionIcon({ icon: Icon }: { icon: LucideIcon }) {
  return <Icon size={16} strokeWidth={1.6} className="size-4" />
}

export function NavProjects() {
  const t = useTranslations("recent.projects")
  const tRecent = useTranslations("recent")
  const { isMobile, setOpenMobile } = useSidebar()
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const activeRecentProjectID = searchParams.get("project") ?? ""
  const activeChatProjectID = searchParams.get("project_id") ?? ""
  const activeProjectID = pathname === "/chat" ? activeChatProjectID : activeRecentProjectID
  const activeConversationID = useActiveSidebarConversation()
  const { deleteFilesByDefault: deleteConversationFilesByDefault } = useChatPreferences()
  const { requestNewConversation } = useChatSession()
  const {
    items,
    projects,
    lastChange,
    createProject,
    updateProject,
    deleteProject,
    renameByPublicID,
    setStarByPublicID,
    setProjectByPublicID,
    archiveByPublicID,
    deleteByPublicID,
    touchByPublicID,
  } = useSidebarRecents()
  const [draft, setDraft] = React.useState<ProjectDraft | null>(null)
  const [knowledgeTarget, setKnowledgeTarget] = React.useState<KnowledgeTarget | null>(null)
  const [deleteTarget, setDeleteTarget] = React.useState<ProjectActionTarget | null>(null)
  const [deleteProjectConversations, setDeleteProjectConversations] = React.useState(false)
  const [deleteProjectFiles, setDeleteProjectFiles] = React.useState(false)
  const [conversationRenameTarget, setConversationRenameTarget] = React.useState<SidebarConversationRenameTarget>(null)
  const [conversationDeleteTarget, setConversationDeleteTarget] = React.useState<SidebarConversationDeleteTarget>(null)
  const [deleteConversationFiles, setDeleteConversationFiles] = React.useState(false)
  const [shareTarget, setShareTarget] = React.useState<{ publicID: string; title: string } | null>(null)
  const [renameValue, setRenameValue] = React.useState("")
  const [expandedProjectIDs, setExpandedProjectIDs] = React.useState<Set<string>>(() => new Set())
  const [projectConversationState, setProjectConversationState] = React.useState<Record<string, ProjectConversationState>>({})
  const [openProjectMenuID, setOpenProjectMenuID] = React.useState<string | null>(null)
  const [hoveredProjectMenuID, setHoveredProjectMenuID] = React.useState<string | null>(null)
  const [hoveredProjectRowID, setHoveredProjectRowID] = React.useState<string | null>(null)
  const [focusedProjectRowID, setFocusedProjectRowID] = React.useState<string | null>(null)
  const projectConversationStateRef = React.useRef(projectConversationState)
  const activeConversationProjectID = React.useMemo(
    () => items.find((item) => item.publicID === activeConversationID)?.projectID ?? "",
    [activeConversationID, items],
  )
  const deleteProjectConversationsID = React.useId()
  const deleteProjectFilesID = React.useId()
  const deleteConversationFilesID = React.useId()
  const onExportConversation = useConversationExportAction({
    successMessage: tRecent("exported"),
    failureMessage: tRecent("exportFailed"),
  })
  const onExportMarkdownConversation = useConversationExportAction({
    successMessage: tRecent("exportMarkdownSuccess"),
    failureMessage: tRecent("exportMarkdownFailed"),
    format: "markdown",
  })
  const onCopyMarkdownConversation = useConversationExportAction({
    successMessage: tRecent("copyMarkdownSuccess"),
    failureMessage: tRecent("copyMarkdownFailed"),
    format: "markdown",
    action: "copy",
  })
  const onExportImageConversation = useConversationExportAction({
    successMessage: tRecent("exportImageSuccess"),
    failureMessage: tRecent("exportImageFailed"),
    format: "image",
    imageLabels: {
      titleFallback: tRecent("untitled"),
      exportedAt: tRecent("imageExport.exportedAt"),
      conversationID: tRecent("imageExport.conversationID"),
      roleAssistant: tRecent("imageExport.roleAssistant"),
      roleSystem: tRecent("imageExport.roleSystem"),
      roleUser: tRecent("imageExport.roleUser"),
      roleMessage: tRecent("imageExport.roleMessage"),
      model: tRecent("imageExport.model"),
      attachments: tRecent("imageExport.attachments"),
      noTextContent: tRecent("imageExport.noTextContent"),
      truncated: tRecent("imageExport.truncated"),
      watermark: tRecent("imageExport.watermark"),
    },
  })

  React.useEffect(() => {
    projectConversationStateRef.current = projectConversationState
  }, [projectConversationState])

  const closeDraft = React.useCallback(() => {
    setDraft(null)
  }, [])

  const onRenameConversation = React.useCallback((publicID: string, currentTitle: string) => {
    setConversationRenameTarget({ publicID, currentTitle })
    setRenameValue(currentTitle)
  }, [])

  const onRenameConversationCancel = React.useCallback(() => {
    setConversationRenameTarget(null)
    setRenameValue("")
  }, [])

  const onRenameConversationCommit = React.useCallback(
    async (publicID: string, currentTitle: string) => {
      const nextTitle = renameValue.trim()
      if (!nextTitle || nextTitle === currentTitle) {
        onRenameConversationCancel()
        return
      }
      await renameByPublicID(publicID, nextTitle)
      onRenameConversationCancel()
    },
    [onRenameConversationCancel, renameByPublicID, renameValue],
  )

  const onArchiveConversation = React.useCallback(
    async (publicID: string) => {
      await archiveByPublicID(publicID, true)
      if (activeConversationID === publicID) {
        router.push("/chat")
      }
    },
    [activeConversationID, archiveByPublicID, router],
  )

  const onDeleteConversation = React.useCallback((publicID: string, title: string) => {
    setDeleteConversationFiles(deleteConversationFilesByDefault)
    setConversationDeleteTarget({ publicID, title })
  }, [deleteConversationFilesByDefault])

  const confirmDeleteConversation = React.useCallback(async () => {
    if (!conversationDeleteTarget) {
      return
    }
    const ok = await deleteByPublicID(conversationDeleteTarget.publicID, { deleteFiles: deleteConversationFiles })
    if (ok && activeConversationID === conversationDeleteTarget.publicID) {
      router.push("/chat")
    }
    setConversationDeleteTarget(null)
    setDeleteConversationFiles(false)
  }, [activeConversationID, conversationDeleteTarget, deleteByPublicID, deleteConversationFiles, router])

  const loadProjectConversations = React.useCallback(async (projectID: string, force = false) => {
    const current = projectConversationStateRef.current[projectID]
    if (!force && (current?.loading || current?.loaded)) {
      return
    }

    setProjectConversationState((prev) => ({
      ...prev,
      [projectID]: {
        items: prev[projectID]?.items ?? [],
        loading: true,
        loaded: prev[projectID]?.loaded ?? false,
        error: false,
      },
    }))

    const token = await resolveAccessToken()
    if (!token) {
      setProjectConversationState((prev) => ({
        ...prev,
        [projectID]: {
          items: [],
          loading: false,
          loaded: true,
          error: false,
        },
      }))
      return
    }

    try {
      const data = await listConversations(token, {
        page: 1,
        pageSize: PROJECT_CONVERSATION_PAGE_SIZE,
        status: "active",
        starred: "all",
        project: projectID,
      })
      setProjectConversationState((prev) => ({
        ...prev,
        [projectID]: {
          items: mergeUniqueByPublicID(prev[projectID]?.items ?? [], data.results ?? [], sortByUpdatedAtDesc),
          loading: false,
          loaded: true,
          error: false,
        },
      }))
    } catch {
      setProjectConversationState((prev) => ({
        ...prev,
        [projectID]: {
          items: prev[projectID]?.items ?? [],
          loading: false,
          loaded: false,
          error: true,
        },
      }))
    }
  }, [])

  const ensureProjectExpanded = React.useCallback(
    (projectID: string) => {
      const shouldLoad = !projectConversationStateRef.current[projectID]?.loaded
      setExpandedProjectIDs((prev) => {
        if (prev.has(projectID)) {
          return prev
        }
        const next = new Set(prev)
        next.add(projectID)
        return next
      })
      if (shouldLoad) {
        void loadProjectConversations(projectID)
      }
    },
    [loadProjectConversations],
  )

  const toggleProjectExpanded = React.useCallback(
    (projectID: string) => {
      const shouldLoad = !projectConversationStateRef.current[projectID]?.loaded
      let expandedNext = false
      setExpandedProjectIDs((prev) => {
        const next = new Set(prev)
        if (next.has(projectID)) {
          next.delete(projectID)
        } else {
          next.add(projectID)
          expandedNext = true
        }
        return next
      })
      if (expandedNext && shouldLoad) {
        void loadProjectConversations(projectID)
      }
    },
    [loadProjectConversations],
  )

  const startProjectConversation = React.useCallback(
    (projectID: string) => {
      ensureProjectExpanded(projectID)
      requestNewConversation({ projectID })
      router.push(`/chat?project_id=${encodeURIComponent(projectID)}`)
      if (isMobile) {
        setOpenMobile(false)
      }
    },
    [ensureProjectExpanded, isMobile, requestNewConversation, router, setOpenMobile],
  )

  React.useEffect(() => {
    if (activeProjectID) {
      ensureProjectExpanded(activeProjectID)
    }
  }, [activeProjectID, ensureProjectExpanded])

  React.useEffect(() => {
    if (!lastChange) {
      return
    }

    setProjectConversationState((prev) => {
      const targetProjectID =
        lastChange.type === "remove"
          ? ""
          : (lastChange.item?.projectID ?? lastChange.patch?.projectID ?? items.find((item) => item.publicID === lastChange.publicID)?.projectID ?? "")
      const projectIDs = Array.from(new Set([...Object.keys(prev), targetProjectID].filter(Boolean)))
      if (projectIDs.length === 0) {
        return prev
      }

      let changed = false
      const next = { ...prev }

      for (const projectID of projectIDs) {
        const state = prev[projectID]
        if (!state?.loaded && projectID !== targetProjectID) {
          continue
        }

        if (lastChange.type === "remove") {
          const current = state ?? { items: [], loading: false, loaded: true, error: false }
          const itemsNext = removeByPublicID(current.items, lastChange.publicID)
          if (itemsNext.length !== current.items.length) {
            next[projectID] = { ...current, items: itemsNext }
            changed = true
          }
          continue
        }

        const current = state ?? { items: [], loading: false, loaded: true, error: false }
        const existing = current.items.find((item) => item.publicID === lastChange.publicID)
        const base =
          lastChange.item ??
          (existing ? { ...existing, ...(lastChange.patch ?? {}) } : items.find((item) => item.publicID === lastChange.publicID))

        if (!base) {
          continue
        }

        const updated = lastChange.patch ? { ...base, ...lastChange.patch } : base
        const belongsToProject = updated.projectID === projectID && updated.status !== "archived"
        const currentlyPresent = Boolean(existing)
        if (belongsToProject) {
          next[projectID] = {
            ...current,
            items: upsertByPublicID(current.items, updated, sortByUpdatedAtDesc),
            loaded: true,
            error: false,
          }
          changed = true
        } else if (currentlyPresent) {
          next[projectID] = { ...current, items: removeByPublicID(current.items, updated.publicID) }
          changed = true
        }
      }

      return changed ? next : prev
    })
    if (lastChange.type !== "remove") {
      const projectID = lastChange.item?.projectID ?? lastChange.patch?.projectID ?? ""
      if (projectID) {
        setExpandedProjectIDs((prev) => {
          if (prev.has(projectID)) {
            return prev
          }
          const next = new Set(prev)
          next.add(projectID)
          return next
        })
      }
    }
  }, [items, lastChange])

  const commitDraft = React.useCallback(async () => {
    const name = draft?.name.trim() ?? ""
    if (!draft || !name) {
      closeDraft()
      return
    }
    if (draft.publicID) {
      await updateProject(draft.publicID, { name, systemPrompt: draft.systemPrompt.trim() })
    } else {
      await createProject({ name, systemPrompt: draft.systemPrompt.trim() })
    }
    closeDraft()
  }, [closeDraft, createProject, draft, updateProject])

  const confirmDelete = React.useCallback(async () => {
    if (!deleteTarget?.publicID) {
      return
    }
    const deletingProjectID = deleteTarget.publicID
    const deletingActiveConversation =
      deleteProjectConversations &&
      (
        projectConversationState[deletingProjectID]?.items.some((item) => item.publicID === activeConversationID) ||
        items.some((item) => item.projectID === deletingProjectID && item.publicID === activeConversationID)
      )
    const deleted = await deleteProject(deletingProjectID, {
      deleteConversations: deleteProjectConversations,
      deleteFiles: deleteProjectConversations && deleteProjectFiles,
    })
    if (deleted && pathname === "/recent" && activeProjectID === deleteTarget.publicID) {
      router.replace("/recent")
    }
    if (deleted && deletingActiveConversation) {
      router.push("/chat")
    }
    if (deleted) {
      setExpandedProjectIDs((prev) => {
        const next = new Set(prev)
        next.delete(deletingProjectID)
        return next
      })
      setProjectConversationState((prev) => {
        const { [deletingProjectID]: _deleted, ...next } = prev
        return next
      })
    }
    setDeleteTarget(null)
    setDeleteProjectConversations(false)
    setDeleteProjectFiles(false)
  }, [
    activeConversationID,
    activeProjectID,
    deleteProject,
    deleteProjectConversations,
    deleteProjectFiles,
    deleteTarget,
    items,
    pathname,
    projectConversationState,
    router,
  ])

  React.useEffect(() => {
    if (!deleteTarget) {
      setDeleteProjectConversations(false)
      setDeleteProjectFiles(false)
    }
  }, [deleteTarget])

  React.useEffect(() => {
    if (!deleteProjectConversations) {
      setDeleteProjectFiles(false)
    }
  }, [deleteProjectConversations])

  React.useEffect(() => {
    if (!conversationDeleteTarget) {
      setDeleteConversationFiles(false)
    }
  }, [conversationDeleteTarget])

  if (projects.length === 0) {
    return (
      <>
        <div className="relative z-10 group-data-[collapsible=icon]:pointer-events-none group-data-[collapsible=icon]:opacity-0">
          <SidebarGroup>
            <ProjectGroupHeader
              title={t("title")}
              createLabel={t("create")}
              onCreate={() => setDraft({ name: "", systemPrompt: "" })}
            />
            <div className="px-2 py-1 text-xs text-sidebar-foreground/55">{t("empty")}</div>
          </SidebarGroup>
        </div>
        <ProjectDialog draft={draft} setDraft={setDraft} onOpenChange={(open) => !open && closeDraft()} onSubmit={commitDraft} />
      </>
    )
  }

  return (
    <>
      <div className="relative z-10 group-data-[collapsible=icon]:pointer-events-none group-data-[collapsible=icon]:opacity-0">
        <SidebarGroup>
          <ProjectGroupHeader
            title={t("title")}
            createLabel={t("create")}
            onCreate={() => setDraft({ name: "", systemPrompt: "" })}
          />
          <SidebarMenu>
            {projects.map((project) => {
              const expanded = expandedProjectIDs.has(project.publicID)
              const conversationState = projectConversationState[project.publicID]
              const hasActiveChild = Boolean(conversationState?.items.some((item) => item.publicID === activeConversationID))
              const active =
                ((pathname === "/recent" || pathname === "/chat") && activeProjectID === project.publicID) ||
                activeConversationProjectID === project.publicID ||
                hasActiveChild
              const rowHovered = hoveredProjectRowID === project.publicID
              const rowFocused = focusedProjectRowID === project.publicID
              const menuHovered = hoveredProjectMenuID === project.publicID
              const menuOpen = openProjectMenuID === project.publicID
              const showProjectActions = rowHovered || rowFocused || menuHovered || menuOpen
              const projectConversationContentID = `sidebar-project-${project.publicID}-conversations`
              return (
                <SidebarMenuItem key={project.publicID}>
                  <div
                    className="group/project-row relative"
                    onMouseEnter={() => setHoveredProjectRowID(project.publicID)}
                    onMouseLeave={() => setHoveredProjectRowID(null)}
                    onFocus={() => setFocusedProjectRowID(project.publicID)}
                    onBlur={(event) => {
                      const nextTarget = event.relatedTarget
                      if (!(nextTarget instanceof Node) || !event.currentTarget.contains(nextTarget)) {
                        setFocusedProjectRowID(null)
                      }
                    }}
                  >
                    <ProjectTreeButton
                      active={active}
                      contentID={projectConversationContentID}
                      expanded={expanded}
                      name={project.name}
                      onToggleExpanded={() => toggleProjectExpanded(project.publicID)}
                    />
                    <ProjectInlineAction
                      label={t("newChatInProject")}
                      visible={showProjectActions}
                      className="right-8"
                      onClick={() => startProjectConversation(project.publicID)}
                    >
                      <ProjectActionIcon icon={Plus} />
                    </ProjectInlineAction>
                    <DropdownMenu
                      modal={false}
                      open={menuOpen}
                      onOpenChange={(open) => setOpenProjectMenuID(open ? project.publicID : null)}
                    >
                      <DropdownMenuTrigger asChild>
                        <ProjectInlineAction
                          label={t("menu")}
                          visible={showProjectActions}
                          className="right-0"
                          onHoverChange={(hovered) => setHoveredProjectMenuID(hovered ? project.publicID : null)}
                        >
                          <Ellipsis size={16} strokeWidth={1.4} animate={menuHovered ? "default" : undefined} />
                        </ProjectInlineAction>
                      </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-max min-w-36 max-w-[calc(100vw-2rem)]">
                        <DropdownMenuItem
                          onSelect={(event) => {
                            event.preventDefault()
                            setKnowledgeTarget({ publicID: project.publicID, name: project.name })
                          }}
                        >
                          <DropdownMenuItemIcon icon={FileText} />
                          {t("knowledge.menu")}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onSelect={(event) => {
                            event.preventDefault()
                            setDraft({ publicID: project.publicID, name: project.name, systemPrompt: project.systemPrompt ?? "" })
                          }}
                        >
                          <DropdownMenuItemIcon icon={PencilLine} />
                          {t("edit")}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onSelect={(event) => {
                            event.preventDefault()
                            setDeleteTarget({ publicID: project.publicID, name: project.name })
                          }}
                        >
                          <DropdownMenuItemIcon icon={Trash} className="text-current" />
                          {t("delete")}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                  <AnimatePresence initial={false}>
                    {expanded ? (
                      <motion.div
                        key={`${project.publicID}-conversations`}
                        id={projectConversationContentID}
                        initial={{ height: 0, opacity: 0, "--mask-stop": "0%", y: 6 }}
                        animate={{ height: "auto", opacity: 1, "--mask-stop": "100%", y: 0 }}
                        exit={{ height: 0, opacity: 0, "--mask-stop": "0%", y: 6 }}
                        transition={PROJECT_TREE_ACCORDION_TRANSITION}
                        style={PROJECT_TREE_ACCORDION_MASK_STYLE}
                      >
                        <SidebarMenuSub className="mx-0 w-full translate-x-0 border-l-0 px-0 py-0.5">
                          {conversationState?.loading ? (
                            <SidebarMenuSubItem>
                              <div className="flex h-7 w-full items-center gap-2 rounded-md pl-8 pr-2 text-xs text-muted-foreground">
                                <Spinner className="size-3.5" />
                                <span>{tRecent("loadingMore")}</span>
                              </div>
                            </SidebarMenuSubItem>
                          ) : null}
                          {conversationState?.error ? (
                            <SidebarMenuSubItem>
                              <button
                                type="button"
                                className="flex h-7 w-full min-w-0 items-center gap-2 rounded-md pl-8 pr-2 text-left text-xs text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                                onClick={() => void loadProjectConversations(project.publicID, true)}
                              >
                                <span className="truncate">{tRecent("loadMoreFailed")}</span>
                                <span className="ml-auto shrink-0 underline underline-offset-4">{tRecent("retry")}</span>
                              </button>
                            </SidebarMenuSubItem>
                          ) : null}
                          {conversationState?.loaded && conversationState.items.length === 0 ? (
                            <SidebarMenuSubItem>
                              <div className="w-full rounded-md py-1 pl-8 pr-2 text-xs text-sidebar-foreground/55">{tRecent("empty")}</div>
                            </SidebarMenuSubItem>
                          ) : null}
                          {conversationState?.items.map((conversation) => {
                            const title = conversation.title || tRecent("untitled")
                            return (
                              <SidebarConversationItem
                                key={conversation.publicID}
                                active={activeConversationID === conversation.publicID}
                                item={{
                                  publicID: conversation.publicID,
                                  title,
                                  url: `/chat?conversation_id=${conversation.publicID}`,
                                  starred: conversation.isStarred,
                                  shareActive: conversation.shareStatus === "active" && Boolean(conversation.shareID?.trim()),
                                }}
                                starAction={{
                                  label: conversation.isStarred ? tRecent("row.unstar") : tRecent("row.star"),
                                  icon: conversation.isStarred ? StarOff : Star,
                                  onSelect: (targetPublicID) => {
                                    void setStarByPublicID(targetPublicID, !conversation.isStarred)
                                  },
                                }}
                                projectMenu={{
                                  label: tRecent("row.moveToProject"),
                                  unassignedLabel: tRecent("projects.unassigned"),
                                  currentProjectID: conversation.projectID,
                                  projects,
                                  onSelect: (targetPublicID, projectID) => {
                                    void setProjectByPublicID(targetPublicID, projectID)
                                  },
                                }}
                                isTransferring={false}
                                isRenaming={conversationRenameTarget?.publicID === conversation.publicID}
                                renameValue={conversationRenameTarget?.publicID === conversation.publicID ? renameValue : title}
                                rowClassName="w-full"
                                linkClassName="pl-8"
                                onRenameValueChange={setRenameValue}
                                onRenameCommit={onRenameConversationCommit}
                                onRenameCancel={onRenameConversationCancel}
                                onRename={onRenameConversation}
                                onArchive={onArchiveConversation}
                                onShare={(publicID, shareTitle) => setShareTarget({ publicID, title: shareTitle })}
                                onExport={onExportConversation}
                                onExportMarkdown={onExportMarkdownConversation}
                                onExportImage={onExportImageConversation}
                                onCopyMarkdown={onCopyMarkdownConversation}
                                onDelete={onDeleteConversation}
                                onNavigate={isMobile ? () => setOpenMobile(false) : undefined}
                                menuTriggerID={`project-conversation-menu-trigger-${conversation.publicID}`}
                              />
                            )
                          })}
                        </SidebarMenuSub>
                      </motion.div>
                    ) : null}
                  </AnimatePresence>
                </SidebarMenuItem>
              )
            })}
          </SidebarMenu>
        </SidebarGroup>
      </div>

      <ProjectDialog draft={draft} setDraft={setDraft} onOpenChange={(open) => !open && closeDraft()} onSubmit={commitDraft} />

      <ProjectKnowledgePanel
        target={knowledgeTarget}
        onOpenChange={(open) => {
          if (!open) {
            setKnowledgeTarget(null)
          }
        }}
      />

      <AlertDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null)
            setDeleteProjectConversations(false)
            setDeleteProjectFiles(false)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("deleteDescription", { name: deleteTarget?.name ?? t("untitled") })}
            </AlertDialogDescription>
            <div className="mt-1 flex items-start gap-2 py-2 text-left">
              <Checkbox
                id={deleteProjectConversationsID}
                checked={deleteProjectConversations}
                className="mt-0.5"
                onCheckedChange={(checked) => setDeleteProjectConversations(checked === true)}
              />
              <label htmlFor={deleteProjectConversationsID} className="cursor-pointer space-y-1">
                <span className="block text-xs font-medium text-foreground">{t("deleteConversationsLabel")}</span>
                <span className="block text-xs leading-5 text-muted-foreground">{t("deleteConversationsDescription")}</span>
              </label>
            </div>
            {deleteProjectConversations ? (
              <DeleteFilesOption
                id={deleteProjectFilesID}
                checked={deleteProjectFiles}
                onCheckedChange={setDeleteProjectFiles}
              />
            ) : null}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => void confirmDelete()}>
              {t("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={Boolean(conversationDeleteTarget)}
        onOpenChange={(open) => {
          if (!open) {
            setConversationDeleteTarget(null)
            setDeleteConversationFiles(false)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tRecent("dialogs.deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {tRecent("dialogs.deleteDescription", {
                label: tRecent("deleteConversationLabel", { title: conversationDeleteTarget?.title || tRecent("untitled") }),
              })}
            </AlertDialogDescription>
            <DeleteFilesOption
              id={deleteConversationFilesID}
              checked={deleteConversationFiles}
              onCheckedChange={setDeleteConversationFiles}
            />
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tRecent("dialogs.cancel")}</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={() => void confirmDeleteConversation()}>
              {tRecent("dialogs.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {shareTarget ? (
        <ConversationShareDialog
          open={Boolean(shareTarget)}
          onOpenChange={(open) => !open && setShareTarget(null)}
          conversationPublicID={shareTarget.publicID}
          conversationTitle={shareTarget.title}
          onExportImage={() => onExportImageConversation(shareTarget.publicID)}
          onShareChange={(share) => {
            touchByPublicID(shareTarget.publicID, sharePatchFromDTO(share))
          }}
        />
      ) : null}
    </>
  )
}

function ProjectKnowledgePanel({
  target,
  onOpenChange,
}: {
  target: KnowledgeTarget | null
  onOpenChange: (open: boolean) => void
}) {
  const isMobile = useSidebar().isMobile
  const t = useTranslations("recent.projects.knowledge")
  const open = Boolean(target)
  const content = target ? <ProjectKnowledgeContent target={target} /> : null

  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={onOpenChange}>
        <DrawerContent className="flex max-h-[min(58dvh,24rem)] flex-col overflow-hidden rounded-t-xl border-border bg-popover text-popover-foreground pb-[env(safe-area-inset-bottom)]">
          <DrawerHeader className="px-4 pb-2 pt-3 text-left">
            <DrawerTitle>{t("title")}</DrawerTitle>
            <DrawerDescription>{t("description", { name: target?.name ?? t("untitled") })}</DrawerDescription>
          </DrawerHeader>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-2">{content}</div>
          <DrawerFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              {t("close")}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("title")}</DialogTitle>
          <DialogDescription>{t("description", { name: target?.name ?? t("untitled") })}</DialogDescription>
        </DialogHeader>
        {content}
      </DialogContent>
    </Dialog>
  )
}

function ProjectKnowledgeContent({ target }: { target: KnowledgeTarget }) {
  const t = useTranslations("recent.projects.knowledge")
  const [documents, setDocuments] = React.useState<ProjectKnowledgeDocumentDTO[]>([])
  const [files, setFiles] = React.useState<FileObjectDTO[]>([])
  const [selectedFileID, setSelectedFileID] = React.useState("")
  const [loading, setLoading] = React.useState(false)
  const [pending, setPending] = React.useState(false)
  const uploadInputRef = React.useRef<HTMLInputElement>(null)

  const load = React.useCallback(async () => {
    const token = await resolveAccessToken()
    if (!token) {
      return
    }
    setLoading(true)
    try {
      const [documentData, fileData] = await Promise.all([
        listProjectKnowledgeDocuments(token, target.publicID),
        listFiles(token, { page: 1, pageSize: 80, sort: "last_used" }),
      ])
      setDocuments(documentData)
      setFiles(fileData.results ?? [])
    } catch (error) {
      toast.error(t("loadFailed"), { description: resolveLocalizedErrorMessage(error) })
    } finally {
      setLoading(false)
    }
  }, [target.publicID, t])

  React.useEffect(() => {
    void load()
  }, [load])

  const documentFileIDs = React.useMemo(() => new Set(documents.map((item) => item.fileID)), [documents])
  const selectableFiles = React.useMemo(
    () => files.filter((item) => !documentFileIDs.has(item.fileID)),
    [documentFileIDs, files],
  )

  const addFileID = React.useCallback(
    async (fileID: string) => {
      const normalized = fileID.trim()
      if (!normalized || pending) {
        return
      }
      const token = await resolveAccessToken()
      if (!token) {
        return
      }
      setPending(true)
      try {
        const next = await addProjectKnowledgeDocuments(token, target.publicID, { fileIDs: [normalized] })
        setDocuments(next)
        setSelectedFileID("")
        toast.success(t("addSuccess"))
      } catch (error) {
        toast.error(t("addFailed"), { description: resolveLocalizedErrorMessage(error) })
      } finally {
        setPending(false)
      }
    },
    [pending, t, target.publicID],
  )

  const onUploadChange = React.useCallback<React.ChangeEventHandler<HTMLInputElement>>(
    async (event) => {
      const file = event.target.files?.[0]
      event.target.value = ""
      if (!file || pending) {
        return
      }
      const token = await resolveAccessToken()
      if (!token) {
        return
      }
      setPending(true)
      try {
        const uploaded = await uploadFile(token, file)
        const next = await addProjectKnowledgeDocuments(token, target.publicID, { fileIDs: [uploaded.file.fileID] })
        setFiles((current) => [uploaded.file, ...current.filter((item) => item.fileID !== uploaded.file.fileID)])
        setDocuments(next)
        toast.success(t("uploadSuccess"))
      } catch (error) {
        toast.error(t("uploadFailed"), { description: resolveLocalizedErrorMessage(error) })
      } finally {
        setPending(false)
      }
    },
    [pending, t, target.publicID],
  )

  const removeDocument = React.useCallback(
    async (fileID: string) => {
      const token = await resolveAccessToken()
      if (!token || pending) {
        return
      }
      setPending(true)
      try {
        await deleteProjectKnowledgeDocument(token, target.publicID, fileID)
        setDocuments((current) => current.filter((item) => item.fileID !== fileID))
        toast.success(t("deleteSuccess"))
      } catch (error) {
        toast.error(t("deleteFailed"), { description: resolveLocalizedErrorMessage(error) })
      } finally {
        setPending(false)
      }
    },
    [pending, t, target.publicID],
  )

  const rebuildDocument = React.useCallback(
    async (fileID: string) => {
      const token = await resolveAccessToken()
      if (!token || pending) {
        return
      }
      setPending(true)
      try {
        const updated = await reindexProjectKnowledgeDocument(token, target.publicID, fileID)
        setDocuments((current) => current.map((item) => (item.fileID === fileID ? updated : item)))
        toast.success(t("reindexSuccess"))
      } catch (error) {
        toast.error(t("reindexFailed"), { description: resolveLocalizedErrorMessage(error) })
      } finally {
        setPending(false)
      }
    },
    [pending, t, target.publicID],
  )

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card p-3">
        <div className="flex flex-col gap-2 md:flex-row md:items-center">
          <Select
            value={selectedFileID}
            onValueChange={setSelectedFileID}
            disabled={pending || loading}
          >
            <SelectTrigger className="min-w-0 flex-1" aria-label={t("selectExisting")}>
              <SelectValue placeholder={t("selectPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {selectableFiles.map((file) => (
                <SelectItem key={file.fileID} value={file.fileID}>
                  {file.fileName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button type="button" onClick={() => void addFileID(selectedFileID)} disabled={!selectedFileID || pending}>
            {t("add")}
          </Button>
          <input ref={uploadInputRef} type="file" className="hidden" onChange={onUploadChange} />
          <Button type="button" variant="outline" onClick={() => uploadInputRef.current?.click()} disabled={pending}>
            <Upload className="size-4" />
            {t("upload")}
          </Button>
        </div>
      </div>

      <div className="min-h-44 rounded-lg border border-border">
        {loading ? (
          <div className="flex items-center gap-2 p-4 text-sm text-muted-foreground">
            <Spinner className="size-4" />
            <span>{t("loading")}</span>
          </div>
        ) : documents.length === 0 ? (
          <CenteredEmptyState
            className="min-h-44"
            title={t("emptyTitle")}
            description={t("emptyDescription")}
          />
        ) : (
          <div className="divide-y divide-border">
            {documents.map((item) => (
              <div key={item.fileID} className="flex min-w-0 flex-col gap-2 p-3 md:flex-row md:items-center">
                <div className="flex min-w-0 flex-1 items-start gap-2">
                  <FileText className="mt-0.5 size-4 shrink-0 text-muted-foreground" strokeWidth={1.6} />
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-foreground">{item.fileName || item.fileID}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{t("fileMeta", { size: formatKnowledgeFileSize(item.fileSize, t) })}</p>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <Badge variant="outline">{knowledgeStatusLabel(t, item.indexStatus)}</Badge>
                  <Button type="button" variant="ghost" size="icon" className="relative size-8 before:absolute before:-inset-1.5" onClick={() => void rebuildDocument(item.fileID)} disabled={pending} aria-label={t("reindex")}>
                    <RefreshCw className="size-4" />
                  </Button>
                  <Button type="button" variant="ghost" size="icon" className="relative size-8 before:absolute before:-inset-1.5" onClick={() => void removeDocument(item.fileID)} disabled={pending} aria-label={t("delete")}>
                    <Trash className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function normalizeKnowledgeStatus(status: string): "pending" | "indexing" | "ready" | "failed" | "stale" {
  switch (status) {
    case "indexing":
    case "ready":
    case "failed":
    case "stale":
      return status
    default:
      return "pending"
  }
}

function knowledgeStatusLabel(t: ReturnType<typeof useTranslations>, status: string): string {
  switch (normalizeKnowledgeStatus(status)) {
    case "indexing":
      return t("status.indexing")
    case "ready":
      return t("status.ready")
    case "failed":
      return t("status.failed")
    case "stale":
      return t("status.stale")
    default:
      return t("status.pending")
  }
}

function formatKnowledgeFileSize(bytes: number, t: ReturnType<typeof useTranslations>): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return t("fileSize.zero")
  }
  const units = ["byte", "kilobyte", "megabyte", "gigabyte"] as const
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024
    index += 1
  }
  const valueText = value >= 10 || index === 0 ? value.toFixed(0) : value.toFixed(1)
  switch (units[index]) {
    case "kilobyte":
      return t("fileSize.kilobyte", { value: valueText })
    case "megabyte":
      return t("fileSize.megabyte", { value: valueText })
    case "gigabyte":
      return t("fileSize.gigabyte", { value: valueText })
    default:
      return t("fileSize.byte", { value: valueText })
  }
}

function ProjectDialog({
  draft,
  setDraft,
  onOpenChange,
  onSubmit,
}: {
  draft: ProjectDraft | null
  setDraft: (draft: ProjectDraft | null) => void
  onOpenChange: (open: boolean) => void
  onSubmit: () => void | Promise<void>
}) {
  const t = useTranslations("recent.projects")
  const [submitting, setSubmitting] = React.useState(false)

  React.useEffect(() => {
    if (!draft) {
      setSubmitting(false)
    }
  }, [draft])

  const handleSubmit = React.useCallback<React.FormEventHandler<HTMLFormElement>>(
    async (event) => {
      event.preventDefault()
      if (!draft?.name.trim() || submitting) {
        return
      }
      setSubmitting(true)
      try {
        await onSubmit()
      } finally {
        setSubmitting(false)
      }
    },
    [draft?.name, onSubmit, submitting],
  )

  return (
    <Dialog open={Boolean(draft)} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{draft?.publicID ? t("editTitle") : t("createTitle")}</DialogTitle>
          <DialogDescription>{draft?.publicID ? t("editDescription") : t("createDescription")}</DialogDescription>
        </DialogHeader>

        <motion.form layout transition={PROJECT_DIALOG_LAYOUT_TRANSITION} onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t("nameLabel")}</p>
            <Input
              autoFocus
              value={draft?.name ?? ""}
              maxLength={80}
              placeholder={t("namePlaceholder")}
              onChange={(event) => draft && setDraft({ ...draft, name: event.target.value })}
              disabled={submitting}
              required
            />
          </div>
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground">{t("systemPromptLabel")}</p>
            <Textarea
              value={draft?.systemPrompt ?? ""}
              maxLength={12000}
              placeholder={t("systemPromptPlaceholder")}
              className="min-h-32 resize-y"
              onChange={(event) => draft && setDraft({ ...draft, systemPrompt: event.target.value })}
              disabled={submitting}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)} disabled={submitting}>
              {t("cancel")}
            </Button>
            <Button type="submit" disabled={!draft?.name.trim() || submitting}>
              {t("save")}
            </Button>
          </DialogFooter>
        </motion.form>
      </DialogContent>
    </Dialog>
  )
}
