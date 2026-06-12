import { authedRequest } from "@/shared/api/authed-client";
import { pathParam } from "@/shared/api/http-client";
import type {
  AssistantDTO,
  AssistantPageDTO,
  AssistantRequest,
  ScheduledPromptDTO,
  ScheduledPromptPageDTO,
  ScheduledPromptRequest,
  TeamMemberDTO,
  TeamMemberRequest,
  TeamSpaceDTO,
  TeamSpaceRequest,
} from "@/shared/api/collaboration.types";

function pageQuery(page = 1, pageSize = 50) {
  return `?page=${encodeURIComponent(String(page))}&page_size=${encodeURIComponent(String(pageSize))}`;
}

export async function listAssistants(accessToken: string, options?: { includePublic?: boolean; page?: number; pageSize?: number }): Promise<AssistantPageDTO> {
  const query = `${pageQuery(options?.page, options?.pageSize)}${options?.includePublic ? "&includePublic=true" : ""}`;
  return authedRequest<AssistantPageDTO>(`/api/v1/assistants${query}`, { accessToken }, true);
}

export async function createAssistant(accessToken: string, payload: AssistantRequest): Promise<AssistantDTO> {
  return authedRequest<AssistantDTO>("/api/v1/assistants", { method: "POST", accessToken, body: payload }, true);
}

export async function updateAssistant(accessToken: string, assistantID: string, payload: AssistantRequest): Promise<AssistantDTO> {
  return authedRequest<AssistantDTO>(`/api/v1/assistants/${pathParam(assistantID)}`, { method: "PUT", accessToken, body: payload }, true);
}

export async function deleteAssistant(accessToken: string, assistantID: string): Promise<void> {
  await authedRequest<{ deleted: boolean }>(`/api/v1/assistants/${pathParam(assistantID)}`, { method: "DELETE", accessToken }, true);
}

export async function listAssistantMarketplace(accessToken: string, options?: { page?: number; pageSize?: number }): Promise<AssistantPageDTO> {
  return authedRequest<AssistantPageDTO>(`/api/v1/assistant-marketplace${pageQuery(options?.page, options?.pageSize)}`, { accessToken }, true);
}

export async function installAssistant(accessToken: string, assistantID: string): Promise<void> {
  await authedRequest<{ installed: boolean }>(`/api/v1/assistant-marketplace/${pathParam(assistantID)}/install`, { method: "POST", accessToken }, true);
}

export async function uninstallAssistant(accessToken: string, assistantID: string): Promise<void> {
  await authedRequest<{ installed: boolean }>(`/api/v1/assistant-marketplace/${pathParam(assistantID)}/install`, { method: "DELETE", accessToken }, true);
}

export async function listScheduledPrompts(accessToken: string, options?: { page?: number; pageSize?: number }): Promise<ScheduledPromptPageDTO> {
  return authedRequest<ScheduledPromptPageDTO>(`/api/v1/scheduled-prompts${pageQuery(options?.page, options?.pageSize)}`, { accessToken }, true);
}

export async function createScheduledPrompt(accessToken: string, payload: ScheduledPromptRequest): Promise<ScheduledPromptDTO> {
  return authedRequest<ScheduledPromptDTO>("/api/v1/scheduled-prompts", { method: "POST", accessToken, body: payload }, true);
}

export async function updateScheduledPrompt(accessToken: string, promptID: string, payload: ScheduledPromptRequest): Promise<ScheduledPromptDTO> {
  return authedRequest<ScheduledPromptDTO>(`/api/v1/scheduled-prompts/${pathParam(promptID)}`, { method: "PUT", accessToken, body: payload }, true);
}

export async function deleteScheduledPrompt(accessToken: string, promptID: string): Promise<void> {
  await authedRequest<{ deleted: boolean }>(`/api/v1/scheduled-prompts/${pathParam(promptID)}`, { method: "DELETE", accessToken }, true);
}

export async function listTeamSpaces(accessToken: string): Promise<TeamSpaceDTO[]> {
  return authedRequest<TeamSpaceDTO[]>("/api/v1/team-spaces", { accessToken }, true);
}

export async function createTeamSpace(accessToken: string, payload: TeamSpaceRequest): Promise<TeamSpaceDTO> {
  return authedRequest<TeamSpaceDTO>("/api/v1/team-spaces", { method: "POST", accessToken, body: payload }, true);
}

export async function addTeamMember(accessToken: string, teamID: string, payload: TeamMemberRequest): Promise<TeamMemberDTO> {
  return authedRequest<TeamMemberDTO>(`/api/v1/team-spaces/${pathParam(teamID)}/members`, { method: "POST", accessToken, body: payload }, true);
}

export async function removeTeamMember(accessToken: string, teamID: string, userID: number): Promise<void> {
  await authedRequest<{ deleted: boolean }>(`/api/v1/team-spaces/${pathParam(teamID)}/members/${pathParam(userID)}`, { method: "DELETE", accessToken }, true);
}
