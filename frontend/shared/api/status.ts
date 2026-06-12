import { apiRequest } from "@/shared/api/http-client";
import type { ModelAvailabilityDTO } from "@/shared/api/status.types";

export async function getModelAvailability(): Promise<ModelAvailabilityDTO> {
  return apiRequest<ModelAvailabilityDTO>("/api/v1/status/model-availability");
}
