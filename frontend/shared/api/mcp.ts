import { authedRequest } from "@/shared/api/authed-client";
import { pathParam } from "@/shared/api/http-client";
import type {
  CreateMCPServerRequest,
  MCPConnectionTestResponse,
  MCPOAuthStartResponse,
  MCPServerDTO,
  MCPServerDataResponse,
  MCPServerListResponse,
  MCPToolDTO,
  MCPToolListResponse,
  MCPToolPreferenceDTO,
  UpsertMCPToolPreferenceRequest,
} from "@/shared/api/mcp.types";

export async function listAvailableMCPTools(accessToken: string): Promise<MCPToolDTO[]> {
  const data = await authedRequest<MCPToolListResponse>(
    "/api/v1/mcp/tools",
    {
      method: "GET",
      accessToken,
    },
    true,
  );
  return data.results ?? [];
}

export async function listUserMCPServers(accessToken: string): Promise<MCPServerDTO[]> {
  const data = await authedRequest<MCPServerListResponse>(
    "/api/v1/mcp/servers",
    {
      method: "GET",
      accessToken,
    },
    true,
  );
  return data.results ?? [];
}

export async function createUserMCPServer(accessToken: string, body: CreateMCPServerRequest): Promise<MCPServerDTO> {
  const data = await authedRequest<MCPServerDataResponse>(
    "/api/v1/mcp/servers",
    {
      method: "POST",
      accessToken,
      body,
    },
    true,
  );
  return data.server;
}

export async function updateUserMCPServer(accessToken: string, serverID: number, body: CreateMCPServerRequest): Promise<MCPServerDTO> {
  const data = await authedRequest<MCPServerDataResponse>(
    `/api/v1/mcp/servers/${pathParam(serverID)}`,
    {
      method: "PATCH",
      accessToken,
      body,
    },
    true,
  );
  return data.server;
}

export async function syncUserMCPServer(accessToken: string, serverID: number): Promise<MCPToolDTO[]> {
  const data = await authedRequest<MCPToolListResponse>(
    `/api/v1/mcp/servers/${pathParam(serverID)}/sync`,
    {
      method: "POST",
      accessToken,
    },
    true,
  );
  return data.results ?? [];
}

export async function testUserMCPServer(accessToken: string, serverID: number): Promise<MCPConnectionTestResponse> {
  return authedRequest<MCPConnectionTestResponse>(
    `/api/v1/mcp/servers/${pathParam(serverID)}/test`,
    {
      method: "POST",
      accessToken,
    },
    true,
  );
}

export async function getMCPToolPreference(
  accessToken: string,
  conversationPublicID: string,
): Promise<MCPToolPreferenceDTO> {
  const query = conversationPublicID.trim()
    ? `?conversationPublicID=${encodeURIComponent(conversationPublicID.trim())}`
    : "";
  return authedRequest<MCPToolPreferenceDTO>(
    `/api/v1/mcp/tool-selection${query}`,
    {
      method: "GET",
      accessToken,
    },
    true,
  );
}

export async function putMCPToolPreference(
  accessToken: string,
  body: UpsertMCPToolPreferenceRequest,
): Promise<MCPToolPreferenceDTO> {
  return authedRequest<MCPToolPreferenceDTO>(
    "/api/v1/mcp/tool-selection",
    {
      method: "PUT",
      accessToken,
      body,
    },
    true,
  );
}

export async function startMCPServerOAuth(
  accessToken: string,
  serverID: number,
  redirectURI: string,
): Promise<MCPOAuthStartResponse> {
  return authedRequest<MCPOAuthStartResponse>(
    `/api/v1/mcp/servers/${pathParam(serverID)}/oauth/start`,
    {
      method: "POST",
      accessToken,
      body: { redirectURI },
    },
    true,
  );
}
