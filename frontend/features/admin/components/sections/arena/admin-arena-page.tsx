"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { Trophy } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { getArenaLeaderboard } from "@/features/arena/api/arena-api";
import type { ArenaLeaderboardEntry } from "@/features/arena/types/arena";

export function AdminArenaPage() {
  const t = useTranslations("admin.arena");
  const [entries, setEntries] = React.useState<ArenaLeaderboardEntry[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    getArenaLeaderboard()
      .then(setEntries)
      .catch(() => setError(t("loadFailed")))
      .finally(() => setLoading(false));
  }, [t]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Spinner />
        </div>
      ) : error ? (
        <Card className="p-4 text-sm text-destructive">{error}</Card>
      ) : entries.length === 0 ? (
        <Card className="p-8 text-center text-sm text-muted-foreground">{t("empty")}</Card>
      ) : (
        <Card className="overflow-hidden">
          <div className="grid grid-cols-[2.5rem_1fr_5rem_5rem_5rem] gap-2 border-b bg-muted/30 px-4 py-2.5 text-xs font-medium text-muted-foreground">
            <span>#</span>
            <span>{t("model")}</span>
            <span className="text-right">{t("wins")}</span>
            <span className="text-right">{t("battles")}</span>
            <span className="text-right">{t("winRate")}</span>
          </div>
          {entries.map((entry, index) => (
            <div
              key={entry.model}
              className="grid grid-cols-[2.5rem_1fr_5rem_5rem_5rem] items-center gap-2 border-b px-4 py-3 text-sm last:border-0"
            >
              <span className="flex items-center gap-1 font-medium text-muted-foreground">
                {index < 3 ? <Trophy className="size-3.5 text-primary" /> : null}
                {index + 1}
              </span>
              <span className="truncate font-medium">{entry.model}</span>
              <span className="text-right tabular-nums">{entry.winCount}</span>
              <span className="text-right tabular-nums text-muted-foreground">{entry.totalBattles}</span>
              <span className="text-right font-medium tabular-nums">
                {(entry.winRate * 100).toFixed(1)}%
              </span>
            </div>
          ))}
        </Card>
      )}
    </div>
  );
}
