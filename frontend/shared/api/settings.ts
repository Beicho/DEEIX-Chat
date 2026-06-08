import { apiRequest } from "@/shared/api/http-client";
import { authedRequest } from "@/shared/api/authed-client";
import type { ModelOptionPolicy, NativeToolDefinition } from "@/shared/lib/model-option-policy";

type ModelOptionPolicyResponse = {
  mode: string;
  allowedPathsJSON: string;
  deniedPathsJSON: string;
  nativeTools?: NativeToolDefinition[];
};

export type MCPPolicy = {
  maxSelectedToolsPerMessage: number;
};

export type BrandingSettings = {
  appName: string;
  logoURL: string;
  logoDarkURL: string;
};

export async function getBrandingSettings(): Promise<BrandingSettings> {
  return apiRequest<BrandingSettings>("/api/v1/settings/branding");
}

export async function getModelOptionPolicy(accessToken: string): Promise<ModelOptionPolicy> {
  const data = await authedRequest<ModelOptionPolicyResponse>(
    "/api/v1/settings/model-option-policy",
    { accessToken },
    true,
  );
  return {
    mode: data.mode,
    allowedPathsJSON: data.allowedPathsJSON,
    deniedPathsJSON: data.deniedPathsJSON,
    nativeTools: data.nativeTools ?? [],
  };
}

export async function getMCPPolicy(accessToken: string): Promise<MCPPolicy> {
  const data = await authedRequest<MCPPolicy>(
    "/api/v1/settings/mcp-policy",
    { accessToken },
    true,
  );
  return {
    maxSelectedToolsPerMessage: data.maxSelectedToolsPerMessage,
  };
}
