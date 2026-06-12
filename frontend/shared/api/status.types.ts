export type ModelAvailabilityStatus = "normal" | "degraded" | "down" | string;

export type ModelAvailabilityItemDTO = {
  modelName: string;
  callCount: number;
  successRate: number;
  status: ModelAvailabilityStatus;
};

export type ModelAvailabilityDTO = {
  windowHours: number;
  generatedAt: string;
  models: ModelAvailabilityItemDTO[];
};
