import type { PagePayload } from "@/shared/api/common.types";

export type AssistantDTO = {
  publicID: string;
  ownerUserID: number;
  name: string;
  avatarURL: string;
  description: string;
  systemPrompt: string;
  defaultModel: string;
  openingMessage: string;
  visibility: "private" | "public" | string;
  status: string;
  publishedAt: string | null;
  installed: boolean;
  createdAt: string;
  updatedAt: string;
};

export type AssistantRequest = {
  name: string;
  avatarURL?: string;
  description?: string;
  systemPrompt: string;
  defaultModel?: string;
  openingMessage?: string;
  visibility?: "private" | "public";
};

export type ScheduledPromptDTO = {
  publicID: string;
  targetConversationID: string;
  targetConversationTitle: string;
  title: string;
  content: string;
  dueAt: string;
  nextRunAt: string;
  scheduleType: "once" | "daily" | "weekly" | "cron" | string;
  scheduleTime: string;
  scheduleWeekday: number;
  cronExpression: string;
  model: string;
  enabled: boolean;
  status: string;
  lastTriggeredAt: string | null;
  retryCount: number;
  lastError: string;
  createdAt: string;
  updatedAt: string;
};

export type ScheduledPromptRequest = {
  assistantID?: string;
  targetConversationID?: string;
  title: string;
  content: string;
  dueAt?: string;
  scheduleType?: "once" | "daily" | "weekly" | "cron";
  scheduleTime?: string;
  scheduleWeekday?: number;
  cronExpression?: string;
  model?: string;
  enabled: boolean;
};

export type TeamMemberDTO = {
  userID: number;
  role: "owner" | "admin" | "member" | string;
  username: string;
  displayName: string;
  avatarURL: string;
  email: string;
  createdAt: string;
};

export type TeamSpaceDTO = {
  publicID: string;
  ownerUserID: number;
  name: string;
  description: string;
  status: string;
  members: TeamMemberDTO[];
  createdAt: string;
  updatedAt: string;
};

export type TeamSpaceRequest = {
  name: string;
  description?: string;
};

export type TeamMemberRequest = {
  login: string;
  role?: "admin" | "member";
};

export type AssistantPageDTO = PagePayload<AssistantDTO>;
export type ScheduledPromptPageDTO = PagePayload<ScheduledPromptDTO>;
