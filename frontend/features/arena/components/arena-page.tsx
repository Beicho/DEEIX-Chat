"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { Swords, Send, Trophy, Loader2, AlertCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { readAccessToken } from "@/shared/auth/session";
import { createConversation, streamMessage } from "@/shared/api/conversation";
import { listPublicModels } from "@/shared/api/model";
import { submitArenaVote } from "@/features/arena/api/arena-api";

type ArenaColumn = {
  model: string;
  content: string;
  status: "idle" | "streaming" | "complete" | "error";
  error?: string;
};

function randomGroupID(): string {
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function ArenaPage() {
  const t = useTranslations("arena");
  const [availableModels, setAvailableModels] = React.useState<string[]>([]);
  const [selectedModels, setSelectedModels] = React.useState<string[]>([]);
  const [prompt, setPrompt] = React.useState("");
  const [columns, setColumns] = React.useState<ArenaColumn[]>([]);
  const [running, setRunning] = React.useState(false);
  const [blindMode, setBlindMode] = React.useState(false);
  const [revealed, setRevealed] = React.useState(false);
  const [messageGroupID, setMessageGroupID] = React.useState<string>("");
  const [arenaConversationID, setArenaConversationID] = React.useState<string>("");
  const [votedModel, setVotedModel] = React.useState<string | null>(null);
  const [loadError, setLoadError] = React.useState<string | null>(null);

  React.useEffect(() => {
    const token = readAccessToken();
    if (!token) {
      setLoadError(t("notAuthenticated"));
      return;
    }
    listPublicModels(token)
      .then((models) => {
        const names = models
          .filter((m) => {
            try {
              const kinds = JSON.parse(m.kindsJSON || "[]") as string[];
              return kinds.length === 0 || kinds.includes("chat");
            } catch {
              return true;
            }
          })
          .map((m) => m.platformModelName)
          .filter(Boolean);
        setAvailableModels(names);
      })
      .catch(() => setLoadError(t("loadModelsFailed")));
  }, [t]);

  function toggleModel(name: string) {
    setSelectedModels((prev) => {
      if (prev.includes(name)) return prev.filter((m) => m !== name);
      if (prev.length >= 4) return prev; // max 4
      return [...prev, name];
    });
  }

  async function runArena() {
    const token = readAccessToken();
    if (!token || selectedModels.length < 2 || !prompt.trim() || running) return;

    setRunning(true);
    setRevealed(!blindMode);
    setVotedModel(null);
    const groupID = randomGroupID();
    setMessageGroupID(groupID);

    const initialColumns: ArenaColumn[] = selectedModels.map((model) => ({
      model,
      content: "",
      status: "streaming",
    }));
    setColumns(initialColumns);

    // 为每个模型创建独立会话并并行流式
    await Promise.all(
      selectedModels.map(async (model, index) => {
        try {
          const conversation = await createConversation(token, {
            title: `Arena: ${prompt.slice(0, 40)}`,
            model,
          });
          if (index === 0) setArenaConversationID(conversation.publicID);
          await streamMessage(
            token,
            conversation.publicID,
            {
              contentType: "text",
              content: prompt,
              model,
              branchReason: "arena",
              messageGroupID: groupID,
            },
            {
              onDelta: (delta) => {
                setColumns((prev) => {
                  const next = [...prev];
                  if (next[index]) {
                    next[index] = { ...next[index], content: next[index].content + delta };
                  }
                  return next;
                });
              },
            }
          );
          setColumns((prev) => {
            const next = [...prev];
            if (next[index]) next[index] = { ...next[index], status: "complete" };
            return next;
          });
        } catch (err) {
          setColumns((prev) => {
            const next = [...prev];
            if (next[index]) {
              next[index] = {
                ...next[index],
                status: "error",
                error: err instanceof Error ? err.message : "Failed",
              };
            }
            return next;
          });
        }
      })
    );
    setRunning(false);
  }

  async function vote(model: string) {
    const token = readAccessToken();
    if (!token || !messageGroupID || !arenaConversationID || votedModel) return;
    try {
      await submitArenaVote(arenaConversationID, {
        messageGroupID,
        winnerModel: model,
        blindMode,
      });
      setVotedModel(model);
      if (blindMode) setRevealed(true);
    } catch {
      // 重复投票或失败，静默
      setVotedModel(model);
      if (blindMode) setRevealed(true);
    }
  }

  const canRun = selectedModels.length >= 2 && prompt.trim().length > 0 && !running;
  const allComplete =
    columns.length > 0 && columns.every((c) => c.status === "complete" || c.status === "error");

  return (
    <div className="h-full min-h-0 overflow-y-auto overscroll-y-contain px-3 py-4 sm:px-4 sm:py-6">
      <div className="mx-auto max-w-6xl">
        <div className="mb-5 flex items-start gap-3 sm:mb-6 sm:items-center">
          <div className="rounded-lg bg-primary/10 p-2">
            <Swords className="size-5 text-primary" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h1 className="text-balance text-2xl font-semibold tracking-tight">{t("title")}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t("description")}</p>
          </div>
        </div>

        {loadError ? (
          <Card className="mb-4 flex items-center gap-2 p-4 text-sm text-destructive">
            <AlertCircle className="size-4" aria-hidden="true" />
            {loadError}
          </Card>
        ) : null}

        {/* 模型选择 */}
        <Card className="mb-4 p-4">
          <div className="mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm font-medium">{t("selectModels", { min: 2, max: 4 })}</p>
            <label className="flex min-h-11 items-center gap-2 text-sm text-muted-foreground">
              <input
                type="checkbox"
                checked={blindMode}
                onChange={(e) => setBlindMode(e.target.checked)}
                className="size-4 rounded border-border"
                disabled={running}
              />
              {t("blindMode")}
            </label>
          </div>
          <div className="flex flex-wrap gap-2">
            {availableModels.map((name) => {
              const selected = selectedModels.includes(name);
              const disabled = !selected && selectedModels.length >= 4;
              return (
                <button
                  key={name}
                  type="button"
                  onClick={() => toggleModel(name)}
                  disabled={running || disabled}
                  className={cn(
                    "min-h-11 max-w-full rounded-md border px-3 py-2 text-left text-sm transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50",
                    selected
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border text-foreground hover:bg-muted",
                    disabled && "cursor-not-allowed opacity-40"
                  )}
                >
                  {name}
                </button>
              );
            })}
          </div>
        </Card>

        {/* 输入区 */}
        <Card className="mb-4 p-4">
          <Textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder={t("promptPlaceholder")}
            rows={3}
            disabled={running}
            className="mb-3 min-h-28 resize-none"
          />
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-xs text-muted-foreground">
              {t("billingNote", { count: selectedModels.length })}
            </p>
            <Button onClick={runArena} disabled={!canRun} className="min-h-11 gap-2 sm:min-h-10">
              {running ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
              {running ? t("running") : t("compare")}
            </Button>
          </div>
        </Card>

        {/* 分栏对比 */}
        {columns.length > 0 ? (
          <div
            className={cn(
              "grid gap-4",
              columns.length === 2 && "md:grid-cols-2",
              columns.length === 3 && "md:grid-cols-3",
              columns.length >= 4 && "md:grid-cols-2 lg:grid-cols-4"
            )}
          >
            {columns.map((col, index) => (
              <Card key={index} className="flex flex-col p-4">
                <div className="mb-2 flex items-center justify-between gap-2 border-b pb-2">
                  <span className="truncate text-sm font-medium">
                    {revealed ? col.model : t("modelPlaceholder", { index: index + 1 })}
                  </span>
                  {col.status === "streaming" ? (
                    <Loader2 className="size-3.5 shrink-0 animate-spin text-muted-foreground" />
                  ) : col.status === "error" ? (
                    <AlertCircle className="size-3.5 shrink-0 text-destructive" />
                  ) : null}
                </div>
                <div className="min-h-[160px] flex-1 whitespace-pre-wrap break-words text-sm leading-6 text-foreground">
                  {col.status === "error" ? (
                    <span className="text-destructive">{col.error}</span>
                  ) : (
                    col.content || <span className="text-muted-foreground">{t("waiting")}</span>
                  )}
                </div>
                {allComplete && col.status === "complete" ? (
                  <Button
                    variant={votedModel === col.model ? "default" : "outline"}
                    size="sm"
                    onClick={() => vote(col.model)}
                    disabled={votedModel !== null}
                    className="mt-3 min-h-11 gap-2 sm:min-h-9"
                  >
                    <Trophy className="size-3.5" />
                    {votedModel === col.model ? t("voted") : t("voteThis")}
                  </Button>
                ) : null}
              </Card>
            ))}
          </div>
        ) : null}

        {votedModel && revealed ? (
          <Card className="mt-4 flex items-center gap-2 p-4 text-sm">
            <Trophy className="size-4 text-primary" aria-hidden="true" />
            {t("voteResult", { model: votedModel })}
          </Card>
        ) : null}
      </div>
    </div>
  );
}
