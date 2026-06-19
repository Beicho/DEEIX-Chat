"use client";

import * as React from "react";
import { useTranslations } from "next-intl";
import { AlertCircle, Check, Loader2, RotateCcw, Search, Send, Swords, Trophy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { submitArenaVote } from "@/features/arena/api/arena-api";
import { createConversation, streamMessage } from "@/shared/api/conversation";
import { listPublicModels } from "@/shared/api/model";
import type { PublicModelDTO } from "@/shared/api/model.types";
import { readAccessToken } from "@/shared/auth/session";

type ArenaColumn = {
  model: string;
  vendor: string;
  content: string;
  status: "idle" | "streaming" | "complete" | "error";
  error?: string;
};

function randomGroupID(): string {
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => b.toString(16).padStart(2, "0")).join("");
}

function supportsChat(model: PublicModelDTO): boolean {
  try {
    const kinds = JSON.parse(model.kindsJSON || "[]") as string[];
    return kinds.length === 0 || kinds.includes("chat");
  } catch {
    return true;
  }
}

function modelSubtitle(model: PublicModelDTO): string {
  const parts = [model.vendor, model.pricing?.isFree ? "Free" : model.pricing?.mode].filter(Boolean);
  return parts.join(" · ");
}

export function ArenaPage() {
  const t = useTranslations("arena");
  const [availableModels, setAvailableModels] = React.useState<PublicModelDTO[]>([]);
  const [selectedModels, setSelectedModels] = React.useState<string[]>([]);
  const [modelQuery, setModelQuery] = React.useState("");
  const [prompt, setPrompt] = React.useState("");
  const [columns, setColumns] = React.useState<ArenaColumn[]>([]);
  const [running, setRunning] = React.useState(false);
  const [blindMode, setBlindMode] = React.useState(true);
  const [revealed, setRevealed] = React.useState(false);
  const [messageGroupID, setMessageGroupID] = React.useState("");
  const [arenaConversationID, setArenaConversationID] = React.useState("");
  const [votedModel, setVotedModel] = React.useState<string | null>(null);
  const [voteError, setVoteError] = React.useState<string | null>(null);
  const [loadError, setLoadError] = React.useState<string | null>(null);
  const [modelsLoading, setModelsLoading] = React.useState(true);

  React.useEffect(() => {
    const token = readAccessToken();
    if (!token) {
      setLoadError(t("notAuthenticated"));
      setModelsLoading(false);
      return;
    }
    listPublicModels(token)
      .then((models) => {
        const chatModels = models.filter(supportsChat).filter((model) => model.platformModelName);
        setAvailableModels(chatModels);
        setSelectedModels(chatModels.slice(0, 2).map((model) => model.platformModelName));
      })
      .catch(() => setLoadError(t("loadModelsFailed")))
      .finally(() => setModelsLoading(false));
  }, [t]);

  const selectedSet = React.useMemo(() => new Set(selectedModels), [selectedModels]);
  const filteredModels = React.useMemo(() => {
    const query = modelQuery.trim().toLowerCase();
    if (!query) return availableModels;
    return availableModels.filter((model) => {
      const haystack = `${model.platformModelName} ${model.vendor} ${model.description}`.toLowerCase();
      return haystack.includes(query);
    });
  }, [availableModels, modelQuery]);
  const selectedModelDetails = React.useMemo(
    () => selectedModels.map((name) => availableModels.find((model) => model.platformModelName === name)).filter(Boolean) as PublicModelDTO[],
    [availableModels, selectedModels],
  );

  function toggleModel(name: string) {
    setSelectedModels((prev) => {
      if (prev.includes(name)) return prev.filter((model) => model !== name);
      if (prev.length >= 4) return prev;
      return [...prev, name];
    });
  }

  async function runArena() {
    const token = readAccessToken();
    const trimmedPrompt = prompt.trim();
    if (!token || selectedModelDetails.length < 2 || !trimmedPrompt || running) return;

    setRunning(true);
    setRevealed(!blindMode);
    setVotedModel(null);
    setVoteError(null);
    const groupID = randomGroupID();
    setMessageGroupID(groupID);
    setArenaConversationID("");

    const initialColumns = selectedModelDetails.map<ArenaColumn>((model) => ({
      model: model.platformModelName,
      vendor: model.vendor,
      content: "",
      status: "streaming",
    }));
    setColumns(initialColumns);

    await Promise.all(
      selectedModelDetails.map(async (model, index) => {
        try {
          const conversation = await createConversation(token, {
            title: `Arena: ${trimmedPrompt.slice(0, 40)}`,
            model: model.platformModelName,
          });
          if (index === 0) setArenaConversationID(conversation.publicID);
          await streamMessage(
            token,
            conversation.publicID,
            {
              contentType: "text",
              content: trimmedPrompt,
              model: model.platformModelName,
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
            },
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
                error: err instanceof Error ? err.message : t("responseFailed"),
              };
            }
            return next;
          });
        }
      }),
    );
    setRunning(false);
  }

  async function vote(model: string) {
    const token = readAccessToken();
    if (!token || !messageGroupID || !arenaConversationID || votedModel) return;
    try {
      setVoteError(null);
      const result = await submitArenaVote(arenaConversationID, {
        messageGroupID,
        winnerModel: model,
        blindMode,
      });
      setVotedModel(result.winnerModel || model);
      if (blindMode) setRevealed(true);
    } catch {
      setVoteError(t("voteFailed"));
    }
  }

  function resetArena() {
    setColumns([]);
    setMessageGroupID("");
    setArenaConversationID("");
    setVotedModel(null);
    setVoteError(null);
    setRevealed(!blindMode);
  }

  const canRun = selectedModels.length >= 2 && prompt.trim().length > 0 && !running;
  const allComplete = columns.length > 0 && columns.every((col) => col.status === "complete" || col.status === "error");
  const completedCount = columns.filter((col) => col.status === "complete" || col.status === "error").length;

  return (
    <div className="h-full min-h-0 overflow-y-auto overscroll-y-contain px-3 py-4 sm:px-4 sm:py-6">
      <div className="mx-auto max-w-7xl pb-[calc(1rem+env(safe-area-inset-bottom))]">
        <div className="mb-5 flex items-start gap-3">
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
            <AlertCircle className="size-4 shrink-0" aria-hidden="true" />
            {loadError}
          </Card>
        ) : null}

        <div className="grid gap-4 lg:grid-cols-[22rem_minmax(0,1fr)]">
          <section className="space-y-4">
            <Card className="p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <p className="text-sm font-medium">{t("selectedModels", { count: selectedModels.length })}</p>
                  <p className="text-xs text-muted-foreground">{t("selectModels", { min: 2, max: 4 })}</p>
                </div>
                {modelsLoading ? <Loader2 className="size-4 animate-spin text-muted-foreground" aria-hidden="true" /> : null}
              </div>
              <div className="relative mb-3">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <Input
                  value={modelQuery}
                  onChange={(event) => setModelQuery(event.target.value)}
                  placeholder={t("searchModels")}
                  className="min-h-11 pl-9 sm:min-h-9"
                  disabled={running}
                />
              </div>
              <div className="max-h-80 space-y-2 overflow-y-auto pr-1">
                {filteredModels.map((model) => {
                  const selected = selectedSet.has(model.platformModelName);
                  const disabled = !selected && selectedModels.length >= 4;
                  return (
                    <button
                      key={model.platformModelName}
                      type="button"
                      onClick={() => toggleModel(model.platformModelName)}
                      disabled={running || disabled}
                      className={cn(
                        "flex min-h-11 w-full items-center gap-3 rounded-md border px-3 py-2 text-left text-sm transition-colors focus-visible:ring-[3px] focus-visible:ring-ring/50",
                        selected ? "border-primary bg-primary/10 text-primary" : "border-border text-foreground hover:bg-muted",
                        disabled && "cursor-not-allowed opacity-40",
                      )}
                    >
                      <span className={cn("flex size-5 shrink-0 items-center justify-center rounded-full border", selected ? "border-primary bg-primary text-primary-foreground" : "border-border")}>
                        {selected ? <Check className="size-3.5" aria-hidden="true" /> : null}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-medium">{model.platformModelName}</span>
                        <span className="block truncate text-xs text-muted-foreground">{modelSubtitle(model)}</span>
                      </span>
                    </button>
                  );
                })}
              </div>
            </Card>

            <Card className="p-4">
              <label className="flex min-h-11 items-center justify-between gap-3 text-sm">
                <span>
                  <span className="block font-medium">{t("blindMode")}</span>
                  <span className="block text-xs text-muted-foreground">{t("blindModeHelp")}</span>
                </span>
                <input
                  type="checkbox"
                  checked={blindMode}
                  onChange={(event) => {
                    setBlindMode(event.target.checked);
                    if (columns.length === 0) setRevealed(!event.target.checked);
                  }}
                  className="size-4 rounded border-border"
                  disabled={running}
                />
              </label>
            </Card>
          </section>

          <section className="space-y-4">
            <Card className="p-4">
              <Textarea
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                placeholder={t("promptPlaceholder")}
                rows={4}
                disabled={running}
                className="mb-3 min-h-32 resize-none"
              />
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-xs text-muted-foreground">
                  {t("billingNote", { count: selectedModels.length })}
                </p>
                <div className="flex flex-col gap-2 sm:flex-row">
                  {columns.length > 0 ? (
                    <Button type="button" variant="outline" onClick={resetArena} disabled={running} className="min-h-11 gap-2 sm:min-h-10">
                      <RotateCcw className="size-4" aria-hidden="true" />
                      {t("reset")}
                    </Button>
                  ) : null}
                  <Button onClick={() => void runArena()} disabled={!canRun} className="min-h-11 gap-2 sm:min-h-10">
                    {running ? <Loader2 className="size-4 animate-spin" aria-hidden="true" /> : <Send className="size-4" aria-hidden="true" />}
                    {running ? t("running") : columns.length > 0 ? t("runAgain") : t("compare")}
                  </Button>
                </div>
              </div>
            </Card>

            {columns.length > 0 ? (
              <div className="space-y-3">
                <div className="flex flex-col gap-2 rounded-md border bg-muted/20 px-3 py-2 text-sm sm:flex-row sm:items-center sm:justify-between">
                  <span className="text-muted-foreground">
                    {running ? t("progress", { done: completedCount, total: columns.length }) : t("readyToVote")}
                  </span>
                  {blindMode && allComplete && !revealed ? (
                    <Button type="button" variant="ghost" size="sm" className="min-h-9" onClick={() => setRevealed(true)}>
                      {t("revealModels")}
                    </Button>
                  ) : null}
                </div>

                <div
                  className={cn(
                    "grid gap-4",
                    columns.length === 2 && "xl:grid-cols-2",
                    columns.length === 3 && "xl:grid-cols-3",
                    columns.length >= 4 && "lg:grid-cols-2 xl:grid-cols-4",
                  )}
                >
                  {columns.map((col, index) => (
                    <Card key={`${col.model}-${index}`} className="flex min-h-[20rem] flex-col p-4">
                      <div className="mb-3 flex items-start justify-between gap-2 border-b pb-3">
                        <div className="min-w-0">
                          <span className="block truncate text-sm font-medium">
                            {revealed ? col.model : t("modelPlaceholder", { index: index + 1 })}
                          </span>
                          <span className="block truncate text-xs text-muted-foreground">
                            {revealed ? col.vendor || t("unknownVendor") : t("hiddenUntilVote")}
                          </span>
                        </div>
                        {col.status === "streaming" ? (
                          <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" aria-hidden="true" />
                        ) : col.status === "error" ? (
                          <AlertCircle className="size-4 shrink-0 text-destructive" aria-hidden="true" />
                        ) : null}
                      </div>
                      <div className="min-h-0 flex-1 whitespace-pre-wrap break-words text-sm leading-6 text-foreground">
                        {col.status === "error" ? (
                          <span className="text-destructive">{col.error}</span>
                        ) : (
                          col.content || <span className="text-muted-foreground">{t("waiting")}</span>
                        )}
                      </div>
                      {allComplete && col.status === "complete" ? (
                        <Button
                          variant={votedModel === col.model ? "default" : "outline"}
                          onClick={() => void vote(col.model)}
                          disabled={votedModel !== null}
                          className="mt-4 min-h-11 gap-2 sm:min-h-10"
                        >
                          <Trophy className="size-4" aria-hidden="true" />
                          {votedModel === col.model ? t("voted") : t("voteThis")}
                        </Button>
                      ) : null}
                    </Card>
                  ))}
                </div>
              </div>
            ) : (
              <Card className="p-6 text-sm text-muted-foreground">{t("emptyState")}</Card>
            )}

            {voteError ? (
              <Card className="flex items-center gap-2 p-4 text-sm text-destructive">
                <AlertCircle className="size-4 shrink-0" aria-hidden="true" />
                {voteError}
              </Card>
            ) : null}

            {votedModel && revealed ? (
              <Card className="flex items-center gap-2 p-4 text-sm">
                <Trophy className="size-4 text-primary" aria-hidden="true" />
                {t("voteResult", { model: votedModel })}
              </Card>
            ) : null}
          </section>
        </div>
      </div>
    </div>
  );
}
