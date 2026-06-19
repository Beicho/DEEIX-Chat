export interface ModelStatus {
  modelName: string;
  availability: number; // 0-1 (0.95 = 95%)
  status: "operational" | "degraded" | "down";
  lastChecked: string;
}

export interface SystemStatus {
  overallStatus: "operational" | "degraded" | "down";
  models: ModelStatus[];
  lastUpdated: string;
}
