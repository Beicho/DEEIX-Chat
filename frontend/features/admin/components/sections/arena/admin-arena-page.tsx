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

  function formatWinRate(value: number): string {
    return `${(value * 100).toFixed(1)}%`;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-balance text-2xl font-semibold tracking-tight">{t("title")}</h1>
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
          <div className="divide-y md:hidden">
            {entries.map((entry, index) => (
              <div key={entry.model} className="space-y-3 p-4">
                <div className="flex min-w-0 items-center gap-2">
                  <span className="flex min-h-8 min-w-8 items-center justify-center rounded-md bg-muted text-sm font-medium text-muted-foreground">
                    {index < 3 ? <Trophy className="size-3.5 text-primary" aria-hidden="true" /> : index + 1}
                  </span>
                  <span className="min-w-0 flex-1 truncate text-sm font-medium">{entry.model}</span>
                  <span className="text-sm font-semibold tabular-nums">{formatWinRate(entry.winRate)}</span>
                </div>
                <dl className="grid grid-cols-2 gap-2 text-sm">
                  <div className="rounded-md bg-muted/35 px-3 py-2">
                    <dt className="text-xs text-muted-foreground">{t("wins")}</dt>
                    <dd className="font-medium tabular-nums">{entry.winCount}</dd>
                  </div>
                  <div className="rounded-md bg-muted/35 px-3 py-2">
                    <dt className="text-xs text-muted-foreground">{t("battles")}</dt>
                    <dd className="font-medium tabular-nums">{entry.totalBattles}</dd>
                  </div>
                </dl>
              </div>
            ))}
          </div>

          <div className="hidden overflow-x-auto md:block">
            <div className="min-w-[520px]">
              <div className="grid grid-cols-[2.5rem_minmax(0,1fr)_5rem_5rem_5rem] gap-2 border-b bg-muted/30 px-4 py-2.5 text-xs font-medium text-muted-foreground">
                <span>#</span>
                <span>{t("model")}</span>
                <span className="text-right">{t("wins")}</span>
                <span className="text-right">{t("battles")}</span>
                <span className="text-right">{t("winRate")}</span>
              </div>
              {entries.map((entry, index) => (
                <div
                  key={entry.model}
                  className="grid grid-cols-[2.5rem_minmax(0,1fr)_5rem_5rem_5rem] items-center gap-2 border-b px-4 py-3 text-sm last:border-0"
                >
                  <span className="flex items-center gap-1 font-medium text-muted-foreground">
                    {index < 3 ? <Trophy className="size-3.5 text-primary" aria-hidden="true" /> : null}
                    {index + 1}
                  </span>
                  <span className="truncate font-medium">{entry.model}</span>
                  <span className="text-right tabular-nums">{entry.winCount}</span>
                  <span className="text-right tabular-nums text-muted-foreground">{entry.totalBattles}</span>
                  <span className="text-right font-medium tabular-nums">{formatWinRate(entry.winRate)}</span>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
}
