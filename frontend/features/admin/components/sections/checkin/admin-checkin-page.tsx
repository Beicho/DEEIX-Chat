"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Gift, TrendingUp, Users } from "lucide-react";

export function AdminCheckInPage() {
  const t = useTranslations("admin.checkin");
  const [baseReward, setBaseReward] = useState("0.01");
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    setSaving(true);
    // TODO: Call admin API to update checkin config when backend supports it
    // For now, show message that backend doesn't support config yet
    setTimeout(() => {
      alert(t("configNotSupported"));
      setSaving(false);
    }, 500);
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-2 text-muted-foreground">{t("description")}</p>
      </div>

      {/* Stats Overview */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <Users className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("activeUsers")}</p>
              <p className="text-2xl font-semibold">-</p>
              <p className="text-xs text-muted-foreground">{t("last7Days")}</p>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <Gift className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("totalRewards")}</p>
              <p className="text-2xl font-semibold">-</p>
              <p className="text-xs text-muted-foreground">{t("allTime")}</p>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2.5">
              <TrendingUp className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">{t("avgStreak")}</p>
              <p className="text-2xl font-semibold">-</p>
              <p className="text-xs text-muted-foreground">{t("days")}</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Reward Configuration */}
      <Card className="p-6">
        <h2 className="mb-4 text-xl font-semibold">{t("rewardConfig")}</h2>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="baseReward">{t("baseRewardLabel")}</Label>
            <div className="flex gap-2">
              <Input
                id="baseReward"
                type="number"
                step="0.01"
                min="0"
                value={baseReward}
                onChange={(e) => setBaseReward(e.target.value)}
                className="max-w-xs"
              />
              <span className="flex items-center text-sm text-muted-foreground">USD</span>
            </div>
            <p className="text-sm text-muted-foreground">{t("baseRewardHelp")}</p>
          </div>

          <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900/50 dark:bg-yellow-950/20">
            <p className="text-sm text-yellow-800 dark:text-yellow-200">
              <strong>{t("note")}:</strong> {t("configNotSupportedYet")}
            </p>
          </div>

          <Button onClick={handleSave} disabled={saving}>
            {saving ? t("saving") : t("saveChanges")}
          </Button>
        </div>
      </Card>

      {/* Info Card */}
      <Card className="p-6">
        <h3 className="mb-3 font-semibold">{t("systemInfo")}</h3>
        <ul className="space-y-2 text-sm text-muted-foreground">
          <li>• {t("info1")}</li>
          <li>• {t("info2")}</li>
          <li>• {t("info3")}</li>
        </ul>
      </Card>
    </div>
  );
}
