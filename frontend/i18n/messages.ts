import enAdminAnnouncements from "@/i18n/messages/en-US/admin-announcements.json";
import enAdminBilling from "@/i18n/messages/en-US/admin-billing.json";
import enAdminConversation from "@/i18n/messages/en-US/admin-conversation.json";
import enAdminFiles from "@/i18n/messages/en-US/admin-files.json";
import enAdminLogin from "@/i18n/messages/en-US/admin-login.json";
import enAdminLogs from "@/i18n/messages/en-US/admin-logs.json";
import enAdminModels from "@/i18n/messages/en-US/admin-models.json";
import enAdminModeration from "@/i18n/messages/en-US/admin-moderation.json";
import enAdminPrompts from "@/i18n/messages/en-US/admin-prompts.json";
import enAdminTools from "@/i18n/messages/en-US/admin-tools.json";
import enAdminUpstreams from "@/i18n/messages/en-US/admin-upstreams.json";
import enAdminUsers from "@/i18n/messages/en-US/admin-users.json";
import enAdmin from "@/i18n/messages/en-US/admin.json";
import enAnnouncements from "@/i18n/messages/en-US/announcements.json";
import enArena from "@/i18n/messages/en-US/arena.json";
import enBookmarks from "@/i18n/messages/en-US/bookmarks.json";
import enChat from "@/i18n/messages/en-US/chat.json";
import enCheckin from "@/i18n/messages/en-US/checkin.json";
import enCollaboration from "@/i18n/messages/en-US/collaboration.json";
import enCommon from "@/i18n/messages/en-US/common.json";
import enErrors from "@/i18n/messages/en-US/errors.json";
import enFiles from "@/i18n/messages/en-US/files.json";
import enGuide from "@/i18n/messages/en-US/guide.json";
import enLanding from "@/i18n/messages/en-US/landing.json";
import enLogin from "@/i18n/messages/en-US/login.json";
import enNotifications from "@/i18n/messages/en-US/notifications.json";
import enPrompts from "@/i18n/messages/en-US/prompts.json";
import enRecent from "@/i18n/messages/en-US/recent.json";
import enSettings from "@/i18n/messages/en-US/settings.json";
import enShare from "@/i18n/messages/en-US/share.json";
import enStatus from "@/i18n/messages/en-US/status.json";
import type { AppLocale } from "@/i18n/config";

export type AppMessages = typeof DEFAULT_MESSAGES;

export const DEFAULT_MESSAGES = {
  common: enCommon,
  collaboration: enCollaboration,
  errors: enErrors,
  login: enLogin,
  prompts: enPrompts,
  guide: enGuide,
  landing: enLanding,
  chat: enChat,
  announcements: enAnnouncements,
  arena: enArena,
  bookmarks: enBookmarks,
  checkin: enCheckin,
  notifications: enNotifications,
  recent: enRecent,
  share: enShare,
  files: enFiles,
  settings: enSettings,
  status: enStatus,
  admin: enAdmin,
  adminAnnouncements: enAdminAnnouncements,
  adminBilling: enAdminBilling,
  adminConversation: enAdminConversation,
  adminFiles: enAdminFiles,
  adminLogin: enAdminLogin,
  adminLogs: enAdminLogs,
  adminModels: enAdminModels,
  adminModeration: enAdminModeration,
  adminPrompts: enAdminPrompts,
  adminTools: enAdminTools,
  adminUpstreams: enAdminUpstreams,
  adminUsers: enAdminUsers,
};

export async function loadLocaleMessages(locale: AppLocale): Promise<AppMessages> {
  if (locale === "en-US") {
    return DEFAULT_MESSAGES;
  }

  const [
    common,
    collaboration,
    errors,
    login,
    prompts,
    guide,
    landing,
    chat,
    announcements,
    arena,
    bookmarks,
    checkin,
    notifications,
    recent,
    share,
    files,
    settings,
    status,
    admin,
    adminAnnouncements,
    adminBilling,
    adminConversation,
    adminFiles,
    adminLogin,
    adminLogs,
    adminModels,
    adminModeration,
    adminPrompts,
    adminTools,
    adminUpstreams,
    adminUsers,
  ] = await Promise.all([
    import("@/i18n/messages/zh-CN/common.json"),
    import("@/i18n/messages/zh-CN/collaboration.json"),
    import("@/i18n/messages/zh-CN/errors.json"),
    import("@/i18n/messages/zh-CN/login.json"),
    import("@/i18n/messages/zh-CN/prompts.json"),
    import("@/i18n/messages/zh-CN/guide.json"),
    import("@/i18n/messages/zh-CN/landing.json"),
    import("@/i18n/messages/zh-CN/chat.json"),
    import("@/i18n/messages/zh-CN/announcements.json"),
    import("@/i18n/messages/zh-CN/arena.json"),
    import("@/i18n/messages/zh-CN/bookmarks.json"),
    import("@/i18n/messages/zh-CN/checkin.json"),
    import("@/i18n/messages/zh-CN/notifications.json"),
    import("@/i18n/messages/zh-CN/recent.json"),
    import("@/i18n/messages/zh-CN/share.json"),
    import("@/i18n/messages/zh-CN/files.json"),
    import("@/i18n/messages/zh-CN/settings.json"),
    import("@/i18n/messages/zh-CN/status.json"),
    import("@/i18n/messages/zh-CN/admin.json"),
    import("@/i18n/messages/zh-CN/admin-announcements.json"),
    import("@/i18n/messages/zh-CN/admin-billing.json"),
    import("@/i18n/messages/zh-CN/admin-conversation.json"),
    import("@/i18n/messages/zh-CN/admin-files.json"),
    import("@/i18n/messages/zh-CN/admin-login.json"),
    import("@/i18n/messages/zh-CN/admin-logs.json"),
    import("@/i18n/messages/zh-CN/admin-models.json"),
    import("@/i18n/messages/zh-CN/admin-moderation.json"),
    import("@/i18n/messages/zh-CN/admin-prompts.json"),
    import("@/i18n/messages/zh-CN/admin-tools.json"),
    import("@/i18n/messages/zh-CN/admin-upstreams.json"),
    import("@/i18n/messages/zh-CN/admin-users.json"),
  ]);

  return {
    common: common.default,
    collaboration: collaboration.default,
    errors: errors.default,
    login: login.default,
    prompts: prompts.default,
    guide: guide.default,
    landing: landing.default,
    chat: chat.default,
    announcements: announcements.default,
    arena: arena.default,
    bookmarks: bookmarks.default,
    checkin: checkin.default,
    notifications: notifications.default,
    recent: recent.default,
    share: share.default,
    files: files.default,
    settings: settings.default,
    status: status.default,
    admin: admin.default,
    adminAnnouncements: adminAnnouncements.default,
    adminBilling: adminBilling.default,
    adminConversation: adminConversation.default,
    adminFiles: adminFiles.default,
    adminLogin: adminLogin.default,
    adminLogs: adminLogs.default,
    adminModels: adminModels.default,
    adminModeration: adminModeration.default,
    adminPrompts: adminPrompts.default,
    adminTools: adminTools.default,
    adminUpstreams: adminUpstreams.default,
    adminUsers: adminUsers.default,
  };
}
