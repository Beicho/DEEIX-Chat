"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Calendar, Gift, TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getCheckInStatus, claimDailyCheckIn } from "../api/checkin-api";
import type { CheckInStatus, CheckInClaim } from "../types/checkin";

export function CheckInPage() {
  const t = useTranslations("checkin");
  const [status, setStatus] = useState<CheckInStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [claiming, setClaiming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadStatus();
  }, []);

  async function loadStatus() {
    try {
      setLoading(true);
      setError(null);
      const data = await getCheckInStatus();
      setStatus(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load check-in status");
    } finally {
      setLoading(false);
    }
  }

  async function handleClaim() {
    if (!status || status.todayClaimed || claiming) return;
    try {
      setClaiming(true);
      setError(null);
      const result: CheckInClaim = await claimDailyCheckIn();
      // Update status with new data
      setStatus({
        ...status,
        todayClaimed: true,
        consecutiveDays: result.consecutiveDays,
        account: result.account,
        latestTransaction: result.transaction,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to claim reward");
    } finally {
      setClaiming(false);
    }
  }

  if (loading) {
    return (
      <div className="h-full min-h-0 overflow-y-auto overscroll-y-contain px-4 py-6">
        <div className="mx-auto flex max-w-4xl items-center justify-center py-12">
          <div className="text-muted-foreground">{t("loading")}</div>
        </div>
      </div>
    );
  }

  if (error && !status) {
    return (
      <div className="h-full min-h-0 overflow-y-auto overscroll-y-contain px-4 py-6">
        <Card className="p-6">
          <div className="text-center text-destructive">{error}</div>
          <Button onClick={loadStatus} className="mt-4 min-h-11 w-full">
            {t("retry")}
          </Button>
        </Card>
      </div>
    );
  }

  return (
    <div className="h-full min-h-0 overflow-y-auto overscroll-y-contain px-3 py-4 sm:px-4 sm:py-6">
      <div className="mx-auto max-w-4xl pb-[calc(1rem+env(safe-area-inset-bottom))]">
      <div className="mb-6">
        <h1 className="text-balance text-2xl font-semibold tracking-tight sm:text-3xl">{t("title")}</h1>
        <p className="mt-2 text-sm text-muted-foreground sm:text-base">{t("description")}</p>
      </div>

      {/* Stats Cards */}
      <div className="mb-6 grid gap-3 md:grid-cols-3">
        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <TrendingUp className="h-5 w-5 text-primary" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("consecutiveDays")}</p>
              <p className="text-2xl font-semibold">{status?.consecutiveDays || 0}</p>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <Gift className="h-5 w-5 text-primary" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("todayReward")}</p>
              <p className="text-2xl font-semibold">${status?.rewardUSD.toFixed(2) || "0.00"}</p>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <Calendar className="h-5 w-5 text-primary" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("status")}</p>
              <p className="text-lg font-semibold">
                {status?.todayClaimed ? t("claimed") : t("available")}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Main Check-in Card */}
      <Card className="p-5 sm:p-6">
        <div className="text-center">
          {status?.todayClaimed ? (
            <>
              <div className="mb-4 inline-flex rounded-full bg-green-500/10 p-4">
                <Gift className="h-12 w-12 text-green-600" aria-hidden="true" />
              </div>
              <h2 className="mb-2 text-2xl font-semibold">{t("alreadyClaimed")}</h2>
              <p className="mb-6 text-muted-foreground">{t("comeBackTomorrow")}</p>
              {status.nextCheckInDate && (
                <p className="text-sm text-muted-foreground">
                  {t("nextCheckIn")}: {new Date(status.nextCheckInDate).toLocaleDateString()}
                </p>
              )}
            </>
          ) : (
            <>
              <div className="mb-4 inline-flex rounded-full bg-primary/10 p-4">
                <Gift className="h-12 w-12 text-primary" aria-hidden="true" />
              </div>
              <h2 className="mb-2 text-2xl font-semibold">{t("readyToClaim")}</h2>
              <p className="mb-6 text-muted-foreground">
                {t("claimReward", { amount: (status?.rewardUSD ?? 0).toFixed(2) })}
              </p>
              <Button onClick={handleClaim} disabled={claiming} className="min-h-11 w-full sm:w-auto sm:min-w-[200px]">
                {claiming ? t("claiming") : t("claimNow")}
              </Button>
              {error && <p className="mt-3 text-sm text-destructive">{error}</p>}
            </>
          )}
        </div>

        {/* Balance Info */}
        {status?.account && (
          <div className="mt-6 border-t pt-4">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t("currentBalance")}</span>
              <span className="font-semibold">${status.account.balanceUSD.toFixed(2)}</span>
            </div>
          </div>
        )}
      </Card>

      {/* Info Section */}
      <Card className="mt-6 p-5 sm:p-6">
        <h3 className="mb-3 font-semibold">{t("howItWorks")}</h3>
        <ul className="space-y-2 text-sm text-muted-foreground">
          <li>• {t("rule1")}</li>
          <li>• {t("rule2")}</li>
          <li>• {t("rule3")}</li>
        </ul>
      </Card>
      </div>
    </div>
  );
}
