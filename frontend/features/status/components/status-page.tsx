"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, AlertCircle, XCircle, Clock } from "lucide-react";
import { Card } from "@/components/ui/card";
import { getSystemStatus } from "../api/status-api";
import type { SystemStatus, ModelStatus } from "../types/status";

function StatusIndicator({ status }: { status: "operational" | "degraded" | "down" }) {
  switch (status) {
    case "operational":
      return <CheckCircle2 className="h-5 w-5 text-green-600" />;
    case "degraded":
      return <AlertCircle className="h-5 w-5 text-yellow-600" />;
    case "down":
      return <XCircle className="h-5 w-5 text-red-600" />;
  }
}

function AvailabilityBar({ availability }: { availability: number }) {
  const percentage = Math.round(availability * 100);
  let colorClass = "bg-green-500";
  if (percentage < 95) colorClass = "bg-yellow-500";
  if (percentage < 80) colorClass = "bg-red-500";

  return (
    <div className="flex items-center gap-3">
      <div className="h-2 w-full max-w-xs overflow-hidden rounded-full bg-muted">
        <div className={`h-full ${colorClass}`} style={{ width: `${percentage}%` }} />
      </div>
      <span className="min-w-[3rem] text-sm font-medium">{percentage}%</span>
    </div>
  );
}

function ModelStatusRow({ model }: { model: ModelStatus }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b py-4 last:border-0">
      <div className="flex items-center gap-3">
        <StatusIndicator status={model.status} />
        <div>
          <p className="font-medium">{model.modelName}</p>
          <p className="text-xs text-muted-foreground">
            Last checked: {new Date(model.lastChecked).toLocaleTimeString()}
          </p>
        </div>
      </div>
      <AvailabilityBar availability={model.availability} />
    </div>
  );
}

export function StatusPage() {
  const [status, setStatus] = useState<SystemStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadStatus();
    // Auto-refresh every 60s
    const interval = setInterval(loadStatus, 60000);
    return () => clearInterval(interval);
  }, []);

  async function loadStatus() {
    try {
      setError(null);
      const data = await getSystemStatus();
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load status");
    } finally {
      setLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="container mx-auto max-w-4xl px-4 py-8">
        <div className="flex items-center justify-center py-12">
          <Clock className="mr-2 h-5 w-5 animate-spin text-muted-foreground" />
          <span className="text-muted-foreground">Loading system status...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto max-w-4xl px-4 py-8">
        <Card className="p-6 text-center">
          <XCircle className="mx-auto mb-3 h-12 w-12 text-destructive" />
          <p className="text-destructive">{error}</p>
        </Card>
      </div>
    );
  }

  if (!status) return null;

  return (
    <div className="container mx-auto max-w-4xl px-4 py-8">
      {/* Header */}
      <div className="mb-8 text-center">
        <h1 className="mb-2 text-4xl font-bold">System Status</h1>
        <p className="text-muted-foreground">Real-time availability of AI models</p>
      </div>

      {/* Overall Status Card */}
      <Card className="mb-6 p-6">
        <div className="flex items-center gap-4">
          <StatusIndicator status={status.overallStatus} />
          <div className="flex-1">
            <h2 className="text-xl font-semibold">
              {status.overallStatus === "operational"
                ? "All Systems Operational"
                : status.overallStatus === "degraded"
                  ? "Degraded Performance"
                  : "Service Disruption"}
            </h2>
            <p className="text-sm text-muted-foreground">
              Last updated: {new Date(status.lastUpdated).toLocaleString()}
            </p>
          </div>
        </div>
      </Card>

      {/* Models Status */}
      <Card className="p-6">
        <h3 className="mb-4 text-lg font-semibold">Model Availability (Last 24h)</h3>
        <div>
          {status.models.map((model) => (
            <ModelStatusRow key={model.modelName} model={model} />
          ))}
        </div>
      </Card>

      {/* Footer Info */}
      <div className="mt-6 text-center text-sm text-muted-foreground">
        <p>Availability is calculated based on successful requests in the last 24 hours.</p>
        <p className="mt-1">This page updates automatically every minute.</p>
      </div>
    </div>
  );
}
