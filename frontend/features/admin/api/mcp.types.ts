import type {
  CreateServerRequest,
  ReorderServerOrderItem,
  ServerDataResponse,
  ServerListResponse,
  ServerResponse,
  ServerToolOrderListResponse,
  ServerToolOrderResponse,
  ToolListResponse,
  UpdateToolRequest,
} from "@deeix/api-contract";

export type AdminMCPServerDTO = ServerResponse;
export type AdminMCPServerPayload = Omit<
  CreateServerRequest,
  | "oauthAccessToken"
  | "oauthAuthURL"
  | "oauthClientID"
  | "oauthClientSecret"
  | "oauthRefreshToken"
  | "oauthScopes"
  | "oauthTokenURL"
  | "timeoutSeconds"
> & Partial<Pick<
  CreateServerRequest,
  | "oauthAccessToken"
  | "oauthAuthURL"
  | "oauthClientID"
  | "oauthClientSecret"
  | "oauthRefreshToken"
  | "oauthScopes"
  | "oauthTokenURL"
  | "timeoutSeconds"
>>;
export type AdminMCPServerListResponse = ServerListResponse;
export type AdminMCPServerDataResponse = ServerDataResponse;
export type AdminMCPToolListResponse = ToolListResponse;
export type AdminMCPToolPayload = Omit<UpdateToolRequest, "defaultEnabled" | "requiresConfirmation"> &
  Partial<Pick<UpdateToolRequest, "defaultEnabled" | "requiresConfirmation">>;
export type AdminMCPOrderItemPayload = ReorderServerOrderItem;
export type AdminMCPOrderGroupDTO = ServerToolOrderResponse;
export type AdminMCPOrderListResponse = ServerToolOrderListResponse;
