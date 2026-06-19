"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { Gift, TrendingUp, Users } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { getAdminCheckInView, updateAdminCheckInConfig, type AdminCheckInView } from "@/features/admin/api/checkin";

function formatUSD(value: number): string {
  return `$${value.toFixed(2)}`;
}

export function AdminCheckInPage() {
  const t = useTranslations("admin.checkin");
  const [view, setView] = React.useState<AdminCheckInView | null>(null);
  const [baseReward, setBaseReward] = React.useState("0.01");
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [saveStatus, setSaveStatus] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getAdminCheckInView();
      setView(data);
      setBaseReward(data.config.rewardUSD.toFixed(2));
    } catch {
      setError(t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function handleSave() {
    const reward = Number(baseReward);
    if (!Number.isFinite(reward) || reward <= 0) {
      setSaveStatus(t("invalidReward"));
      return;
    }
    try {
      setSaving(true);
      setSaveStatus(null);
      const data = await updateAdminCheckInConfig({ rewardUSD: reward });
      setView(data);
      setBaseReward(data.config.rewardUSD.toFixed(2));
      setSaveStatus(t("saved"));
    } catch {
      setSaveStatus(t("saveFailed"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6 pb-[calc(1rem+env(safe-area-inset-bottom))]">
      <div>
        <h1 className="text-balance text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Spinner />
        </div>
      ) : error ? (
        <Card className="p-4">
          <p className="text-sm text-destructive">{error}</p>
          <Button type="button" variant="outline" className="mt-3 min-h-11 sm:min-h-9" onClick={() => void load()}>
            {t("retry")}
          </Button>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 md:grid-cols-3">
            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-primary/10 p-2.5">
                  <Users className="h-5 w-5 text-primary" aria-hidden="true" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("activeUsers")}</p>
                  <p className="text-2xl font-semibold tabular-nums">{view?.stats.activeUsersLast7Days ?? 0}</p>
                  <p className="text-xs text-muted-foreground">{t("last7Days")}</p>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-primary/10 p-2.5">
                  <Gift className="h-5 w-5 text-primary" aria-hidden="true" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("totalRewards")}</p>
                  <p className="text-2xl font-semibold tabular-nums">{formatUSD(view?.stats.totalRewardUSD ?? 0)}</p>
                  <p className="text-xs text-muted-foreground">{t("allTime")}</p>
                </div>
              </div>
            </Card>

            <Card className="p-4">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-primary/10 p-2.5">
                  <TrendingUp className="h-5 w-5 text-primary" aria-hidden="true" />
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">{t("avgStreak")}</p>
                  <p className="text-2xl font-semibold tabular-nums">
                    {(view?.stats.averageConsecutiveDays ?? 0).toFixed(1)}
                  </p>
                  <p className="text-xs text-muted-foreground">{t("days")}</p>
                </div>
              </div>
            </Card>
          </div>

          <Card className="p-5 sm:p-6">
            <h2 className="mb-4 text-lg font-semibold">{t("rewardConfig")}</h2>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="baseReward">{t("baseRewardLabel")}</Label>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    id="baseReward"
                    type="number"
                    step="0.01"
                    min="0.01"
                    value={baseReward}
                    onChange={(e) => setBaseReward(e.target.value)}
                    className="min-h-11 max-w-xs sm:min-h-9"
                  />
                  <span className="text-sm text-muted-foreground">USD</span>
                </div>
                <p className="text-sm text-muted-foreground">{t("baseRewardHelp")}</p>
              </div>

              <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
                <Button type="button" onClick={() => void handleSave()} disabled={saving} className="min-h-11 sm:min-h-9">
                  {saving ? t("saving") : t("saveChanges")}
                </Button>
                {saveStatus ? <p className="text-sm text-muted-foreground">{saveStatus}</p> : null}
              </div>
            </div>
          </Card>

          <Card className="p-5 sm:p-6">
            <h3 className="mb-3 font-semibold">{t("systemInfo")}</h3>
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li>• {t("info1")}</li>
              <li>• {t("info2")}</li>
              <li>• {t("info3")}</li>
            </ul>
          </Card>
        </>
      )}
    </div>
  );
}
