import type { SystemStatus } from "../types/status";

// Mock data for now - will be replaced with real API call
export async function getSystemStatus(): Promise<SystemStatus> {
  // TODO: Replace with actual API call when backend is ready
  // return await fetch('/api/v1/status/models').then(r => r.json());

  return {
    overallStatus: "operational",
    lastUpdated: new Date().toISOString(),
    models: [
      { modelName: "GPT-4", availability: 0.99, status: "operational", lastChecked: new Date().toISOString() },
      { modelName: "GPT-4o", availability: 0.98, status: "operational", lastChecked: new Date().toISOString() },
      { modelName: "Claude 3.5 Sonnet", availability: 0.97, status: "operational", lastChecked: new Date().toISOString() },
      { modelName: "Claude 3 Opus", availability: 0.96, status: "operational", lastChecked: new Date().toISOString() },
      { modelName: "Gemini Pro", availability: 0.95, status: "operational", lastChecked: new Date().toISOString() },
      { modelName: "GPT-3.5 Turbo", availability: 0.94, status: "operational", lastChecked: new Date().toISOString() },
    ],
  };
}
