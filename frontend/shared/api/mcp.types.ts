export type MCPToolDTO = {
  id: number;
  serverID: number;
  serverName: string;
  name: string;
  displayName: string;
  description: string;
  inputSchemaJSON: string;
  attachmentInputMode: "none" | "image";
  attachmentArgument: string;
  attachmentEncoding: "" | "base64" | "data_url";
  attachmentPromptArgument: string;
  status: string;
  sortOrder: number;
  defaultEnabled: boolean;
  requiresConfirmation: boolean;
  toolKind: string;
  createdAt: string;
  updatedAt: string;
};

export type MCPToolListResponse = {
  results: MCPToolDTO[];
};

export type MCPServerDTO = {
  id: number;
  ownerUserID: number;
  scope: "platform" | "user" | string;
  name: string;
  baseURL: string;
  headersJSON: string;
  status: string;
  timeoutSeconds: number;
  oauthClientID: string;
  oauthAuthURL: string;
  oauthTokenURL: string;
  oauthScopes: string;
  oauthStatus: string;
  toolCount: number;
  activeToolCount: number;
  lastSyncedAt: string | null;
  lastError: string;
  createdAt: string;
  updatedAt: string;
};

export type MCPServerListResponse = {
  results: MCPServerDTO[];
};

export type MCPServerDataResponse = {
  server: MCPServerDTO;
};

export type CreateMCPServerRequest = {
  name: string;
  baseURL: string;
  authToken?: string;
  headersJSON?: string;
  status?: string;
  timeoutSeconds?: number;
  oauthClientID?: string;
  oauthClientSecret?: string;
  oauthAuthURL?: string;
  oauthTokenURL?: string;
  oauthScopes?: string;
};

export type MCPToolPreferenceDTO = {
  conversationPublicID: string;
  selectedToolIDs: number[];
  confirmedToolIDs: number[];
  webSearchEnabled: boolean;
  codeSandboxEnabled: boolean;
  researchMaxLLMCalls: number;
  researchMaxToolCalls: number;
  updatedAt: string;
};

export type UpsertMCPToolPreferenceRequest = Omit<MCPToolPreferenceDTO, "updatedAt">;

export type MCPConnectionTestResponse = {
  ok: boolean;
  errorCode: string;
  message: string;
  toolCount: number;
};

export type MCPOAuthStartResponse = {
  authorizationURL: string;
  state: string;
};
