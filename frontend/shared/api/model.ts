import { authedRequest } from "@/shared/api/authed-client";
import { apiRequest } from "@/shared/api/http-client";
import type { PublicModelDTO } from "@/shared/api/model.types";

export async function listPublicModelCatalog(): Promise<PublicModelDTO[]> {
  return apiRequest<PublicModelDTO[]>("/api/v1/public/models");
}

export async function listPublicModels(accessToken: string): Promise<PublicModelDTO[]> {
  return authedRequest<PublicModelDTO[]>(
    "/api/v1/models",
    {
      accessToken,
    },
    true,
  );
}
