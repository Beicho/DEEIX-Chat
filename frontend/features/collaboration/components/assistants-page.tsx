"use client";

import * as React from "react";
import { Bot, Clock, Plus, Store } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CenteredEmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  createAssistant,
  createScheduledPrompt,
  installAssistant,
  listAssistantMarketplace,
  listAssistants,
  listScheduledPrompts,
} from "@/shared/api/collaboration";
import { listConversations } from "@/shared/api/conversation";
import type { ConversationDTO } from "@/shared/api/conversation.types";
import { listPublicModels } from "@/shared/api/model";
import type { PublicModelDTO } from "@/shared/api/model.types";
import type { AssistantDTO, ScheduledPromptDTO } from "@/shared/api/collaboration.types";
import { useAuthSession } from "@/shared/auth/auth-session-context";
import { cn } from "@/lib/utils";

const SELECTED_ASSISTANT_KEY = "deeix-chat:selected-assistant";

function selectedAssistantID() {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(SELECTED_ASSISTANT_KEY) ?? "";
}

function writeSelectedAssistantID(value: string) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(SELECTED_ASSISTANT_KEY, value);
  window.dispatchEvent(new CustomEvent("deeix-chat:selected-assistant", { detail: value }));
}

function AssistantCard({
  assistant,
  selected,
  installable,
  pending,
  onSelect,
  onInstall,
}: {
  assistant: AssistantDTO;
  selected?: boolean;
  installable?: boolean;
  pending?: boolean;
  onSelect?: () => void;
  onInstall?: () => void;
}) {
  const t = useTranslations("collaboration.assistants");
  const initials = assistant.name.trim().slice(0, 2).toUpperCase() || "AI";
  return (
    <Card className={cn("border-border/70 bg-card", selected && "ring-2 ring-ring/50")}>
      <CardHeader className="flex flex-row items-start gap-3 space-y-0 p-4">
        <Avatar className="size-10 shrink-0">
          <AvatarImage src={assistant.avatarURL} alt="" />
          <AvatarFallback>{initials}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <CardTitle className="truncate text-sm">{assistant.name}</CardTitle>
          <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{assistant.description || assistant.openingMessage}</p>
        </div>
        {assistant.visibility === "public" ? <Badge variant="secondary">{t("public")}</Badge> : null}
      </CardHeader>
      <CardContent className="flex flex-wrap gap-2 px-4 pb-4 pt-0">
        {onSelect ? (
          <Button type="button" size="sm" variant={selected ? "secondary" : "outline"} className="min-h-9" onClick={onSelect}>
            {selected ? t("selected") : t("useAssistant")}
          </Button>
        ) : null}
        {installable ? (
          <Button type="button" size="sm" className="min-h-9" disabled={pending || assistant.installed} onClick={onInstall}>
            {assistant.installed ? t("installed") : t("install")}
          </Button>
        ) : null}
      </CardContent>
    </Card>
  );
}

function LoadingCards() {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border/70 p-4">
          <Skeleton className="h-5 w-1/2" />
          <Skeleton className="mt-3 h-12 w-full" />
        </div>
      ))}
    </div>
  );
}

export function AssistantsPage() {
  const t = useTranslations("collaboration.assistants");
  const { accessToken, userStatus } = useAuthSession();
  const [assistants, setAssistants] = React.useState<AssistantDTO[]>([]);
  const [marketplace, setMarketplace] = React.useState<AssistantDTO[]>([]);
  const [scheduled, setScheduled] = React.useState<ScheduledPromptDTO[]>([]);
  const [conversations, setConversations] = React.useState<ConversationDTO[]>([]);
  const [models, setModels] = React.useState<PublicModelDTO[]>([]);
  const [selectedID, setSelectedID] = React.useState("");
  const [loading, setLoading] = React.useState(true);
  const [pending, setPending] = React.useState(false);
  const [installingID, setInstallingID] = React.useState("");
  const [assistantForm, setAssistantForm] = React.useState({
    name: "",
    avatarURL: "",
    description: "",
    systemPrompt: "",
    defaultModel: "",
    openingMessage: "",
    visibility: "private" as "private" | "public",
  });
  const [scheduleForm, setScheduleForm] = React.useState({
    title: "",
    content: "",
    dueAt: "",
    targetConversationID: "",
    model: "",
    scheduleType: "daily" as "once" | "daily" | "weekly" | "cron",
    scheduleTime: "08:00",
    scheduleWeekday: 1,
    cronExpression: "",
    enabled: true,
  });

  const load = React.useCallback(async () => {
    if (!accessToken || userStatus !== "ready") {
      return;
    }
    setLoading(true);
    try {
      const [assistantData, marketplaceData, scheduledData, conversationData, modelData] = await Promise.all([
        listAssistants(accessToken, { includePublic: true }),
        listAssistantMarketplace(accessToken),
        listScheduledPrompts(accessToken),
        listConversations(accessToken, { pageSize: 50, status: "active" }),
        listPublicModels(accessToken),
      ]);
      setAssistants(assistantData.results ?? []);
      setMarketplace(marketplaceData.results ?? []);
      setScheduled(scheduledData.results ?? []);
      setConversations(conversationData.results ?? []);
      setModels(modelData ?? []);
    } catch {
      toast.error(t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [accessToken, t, userStatus]);

  React.useEffect(() => {
    setSelectedID(selectedAssistantID());
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  const saveAssistant = React.useCallback(async () => {
    if (!accessToken || !assistantForm.name.trim() || !assistantForm.systemPrompt.trim()) {
      return;
    }
    setPending(true);
    try {
      await createAssistant(accessToken, assistantForm);
      setAssistantForm({ name: "", avatarURL: "", description: "", systemPrompt: "", defaultModel: "", openingMessage: "", visibility: "private" });
      toast.success(t("toastCreated"));
      await load();
    } catch {
      toast.error(t("toastCreateFailed"));
    } finally {
      setPending(false);
    }
  }, [accessToken, assistantForm, load, t]);

  const install = React.useCallback(async (assistant: AssistantDTO) => {
    if (!accessToken) {
      return;
    }
    setInstallingID(assistant.publicID);
    try {
      await installAssistant(accessToken, assistant.publicID);
      toast.success(t("toastInstalled"));
      await load();
    } catch {
      toast.error(t("toastInstallFailed"));
    } finally {
      setInstallingID("");
    }
  }, [accessToken, load, t]);

  const selectAssistant = React.useCallback((assistant: AssistantDTO) => {
    writeSelectedAssistantID(assistant.publicID);
    setSelectedID(assistant.publicID);
    toast.success(t("toastSelected"));
  }, [t]);

  const saveSchedule = React.useCallback(async () => {
    if (!accessToken || !scheduleForm.title.trim() || !scheduleForm.content.trim()) {
      return;
    }
    if (scheduleForm.scheduleType === "once" && !scheduleForm.dueAt) {
      return;
    }
    if (scheduleForm.scheduleType !== "once" && !scheduleForm.scheduleTime && scheduleForm.scheduleType !== "cron") {
      return;
    }
    if (scheduleForm.scheduleType === "cron" && !scheduleForm.cronExpression.trim()) {
      return;
    }
    setPending(true);
    try {
      await createScheduledPrompt(accessToken, {
        title: scheduleForm.title,
        content: scheduleForm.content,
        dueAt: scheduleForm.dueAt ? new Date(scheduleForm.dueAt).toISOString() : undefined,
        targetConversationID: scheduleForm.targetConversationID || undefined,
        model: scheduleForm.model || undefined,
        scheduleType: scheduleForm.scheduleType,
        scheduleTime: scheduleForm.scheduleTime,
        scheduleWeekday: scheduleForm.scheduleWeekday,
        cronExpression: scheduleForm.cronExpression,
        enabled: scheduleForm.enabled,
        assistantID: selectedID || undefined,
      });
      setScheduleForm({ title: "", content: "", dueAt: "", targetConversationID: "", model: "", scheduleType: "daily", scheduleTime: "08:00", scheduleWeekday: 1, cronExpression: "", enabled: true });
      toast.success(t("toastScheduleCreated"));
      await load();
    } catch {
      toast.error(t("toastScheduleFailed"));
    } finally {
      setPending(false);
    }
  }, [accessToken, load, scheduleForm, selectedID, t]);

  return (
    <main className="h-full min-h-0 overflow-auto bg-background text-foreground">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-4 md:px-6 md:py-6">
        <header className="flex flex-col gap-2 border-b border-border/60 pb-4">
          <div className="flex items-center gap-2">
            <Bot className="size-4 text-muted-foreground" strokeWidth={1.8} />
            <h1 className="text-lg font-semibold tracking-tight">{t("title")}</h1>
          </div>
          <p className="text-sm text-muted-foreground">{t("description")}</p>
        </header>

        <section className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Bot className="size-4 text-muted-foreground" strokeWidth={1.8} />
              <h2 className="text-sm font-medium">{t("title")}</h2>
            </div>
            {loading ? <LoadingCards /> : assistants.length === 0 ? (
              <CenteredEmptyState title={t("emptyTitle")} description={t("emptyDescription")} />
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {assistants.map((assistant) => (
                  <AssistantCard
                    key={assistant.publicID}
                    assistant={assistant}
                    selected={assistant.publicID === selectedID}
                    onSelect={() => selectAssistant(assistant)}
                  />
                ))}
              </div>
            )}

            <div className="flex items-center gap-2 pt-2">
              <Store className="size-4 text-muted-foreground" strokeWidth={1.8} />
              <h2 className="text-sm font-medium">{t("marketplaceTitle")}</h2>
            </div>
            {loading ? <LoadingCards /> : marketplace.length === 0 ? (
              <CenteredEmptyState title={t("galleryEmpty")} />
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {marketplace.map((assistant) => (
                  <AssistantCard
                    key={assistant.publicID}
                    assistant={assistant}
                    installable
                    pending={installingID === assistant.publicID}
                    onInstall={() => install(assistant)}
                  />
                ))}
              </div>
            )}

            <div className="flex items-center gap-2 pt-2">
              <Clock className="size-4 text-muted-foreground" strokeWidth={1.8} />
              <h2 className="text-sm font-medium">{t("scheduledTitle")}</h2>
            </div>
            <div className="grid gap-2">
              {scheduled.length === 0 ? (
                <CenteredEmptyState title={t("scheduleEmpty")} />
              ) : scheduled.map((item) => (
                <div key={item.publicID} className="rounded-xl border border-border/70 bg-card p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate text-sm font-medium">{item.title}</div>
                      <div className="mt-1 line-clamp-2 text-xs text-muted-foreground">{item.content}</div>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                        <span>{t("scheduleNextRun", { time: new Date(item.nextRunAt || item.dueAt).toLocaleString() })}</span>
                        {item.targetConversationTitle ? <span>{t("scheduleTargetValue", { title: item.targetConversationTitle })}</span> : null}
                        {item.model ? <span>{t("scheduleModelValue", { model: item.model })}</span> : null}
                      </div>
                      {item.lastError ? <div className="mt-2 text-xs text-destructive">{t("scheduleLastError", { message: item.lastError })}</div> : null}
                    </div>
                    {item.enabled ? <Badge variant="secondary">{t("enabled")}</Badge> : null}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <aside className="space-y-4">
            <Card className="border-border/70 bg-card">
              <CardHeader className="p-4">
                <CardTitle className="flex items-center gap-2 text-sm">
                  <Plus className="size-4" strokeWidth={1.8} />
                  {t("createTitle")}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 p-4 pt-0">
                <Label className="text-xs">{t("name")}</Label>
                <Input value={assistantForm.name} onChange={(event) => setAssistantForm((prev) => ({ ...prev, name: event.target.value }))} />
                <Label className="text-xs">{t("avatarURL")}</Label>
                <Input value={assistantForm.avatarURL} onChange={(event) => setAssistantForm((prev) => ({ ...prev, avatarURL: event.target.value }))} />
                <Label className="text-xs">{t("descriptionField")}</Label>
                <Input value={assistantForm.description} onChange={(event) => setAssistantForm((prev) => ({ ...prev, description: event.target.value }))} />
                <Label className="text-xs">{t("systemPrompt")}</Label>
                <Textarea value={assistantForm.systemPrompt} className="min-h-24" onChange={(event) => setAssistantForm((prev) => ({ ...prev, systemPrompt: event.target.value }))} />
                <Label className="text-xs">{t("defaultModel")}</Label>
                <Input value={assistantForm.defaultModel} onChange={(event) => setAssistantForm((prev) => ({ ...prev, defaultModel: event.target.value }))} />
                <Label className="text-xs">{t("openingMessage")}</Label>
                <Textarea value={assistantForm.openingMessage} className="min-h-20" onChange={(event) => setAssistantForm((prev) => ({ ...prev, openingMessage: event.target.value }))} />
                <Label className="text-xs">{t("visibility")}</Label>
                <Select value={assistantForm.visibility} onValueChange={(value: "private" | "public") => setAssistantForm((prev) => ({ ...prev, visibility: value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="private">{t("private")}</SelectItem>
                    <SelectItem value="public">{t("public")}</SelectItem>
                  </SelectContent>
                </Select>
                <Button type="button" className="w-full" disabled={pending} onClick={() => void saveAssistant()}>
                  {t("saveAssistant")}
                </Button>
              </CardContent>
            </Card>

            <Card className="border-border/70 bg-card">
              <CardHeader className="p-4">
                <CardTitle className="text-sm">{t("scheduledTitle")}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 p-4 pt-0">
                <Label className="text-xs">{t("scheduleTitle")}</Label>
                <Input value={scheduleForm.title} onChange={(event) => setScheduleForm((prev) => ({ ...prev, title: event.target.value }))} />
                <Label className="text-xs">{t("scheduleContent")}</Label>
                <Textarea value={scheduleForm.content} className="min-h-20" onChange={(event) => setScheduleForm((prev) => ({ ...prev, content: event.target.value }))} />
                <Label className="text-xs">{t("scheduleTarget")}</Label>
                <Select value={scheduleForm.targetConversationID || "__new__"} onValueChange={(value) => setScheduleForm((prev) => ({ ...prev, targetConversationID: value === "__new__" ? "" : value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__new__">{t("scheduleTargetNew")}</SelectItem>
                    {conversations.map((item) => (
                      <SelectItem key={item.publicID} value={item.publicID}>{item.title || t("scheduleUntitledConversation")}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Label className="text-xs">{t("scheduleModel")}</Label>
                <Select value={scheduleForm.model || "__conversation__"} onValueChange={(value) => setScheduleForm((prev) => ({ ...prev, model: value === "__conversation__" ? "" : value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__conversation__">{t("scheduleModelConversation")}</SelectItem>
                    {models.map((item) => (
                      <SelectItem key={item.platformModelName} value={item.platformModelName}>{item.platformModelName}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Label className="text-xs">{t("scheduleType")}</Label>
                <Select value={scheduleForm.scheduleType} onValueChange={(value: "once" | "daily" | "weekly" | "cron") => setScheduleForm((prev) => ({ ...prev, scheduleType: value }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="daily">{t("scheduleDaily")}</SelectItem>
                    <SelectItem value="weekly">{t("scheduleWeekly")}</SelectItem>
                    <SelectItem value="once">{t("scheduleOnce")}</SelectItem>
                    <SelectItem value="cron">{t("scheduleCron")}</SelectItem>
                  </SelectContent>
                </Select>
                {scheduleForm.scheduleType === "once" ? (
                  <>
                    <Label className="text-xs">{t("scheduleDueAt")}</Label>
                    <Input type="datetime-local" value={scheduleForm.dueAt} onChange={(event) => setScheduleForm((prev) => ({ ...prev, dueAt: event.target.value }))} />
                  </>
                ) : null}
                {scheduleForm.scheduleType === "daily" || scheduleForm.scheduleType === "weekly" ? (
                  <>
                    {scheduleForm.scheduleType === "weekly" ? (
                      <>
                        <Label className="text-xs">{t("scheduleWeekday")}</Label>
                        <Select value={String(scheduleForm.scheduleWeekday)} onValueChange={(value) => setScheduleForm((prev) => ({ ...prev, scheduleWeekday: Number(value) }))}>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="1">{t("weekdayMonday")}</SelectItem>
                            <SelectItem value="2">{t("weekdayTuesday")}</SelectItem>
                            <SelectItem value="3">{t("weekdayWednesday")}</SelectItem>
                            <SelectItem value="4">{t("weekdayThursday")}</SelectItem>
                            <SelectItem value="5">{t("weekdayFriday")}</SelectItem>
                            <SelectItem value="6">{t("weekdaySaturday")}</SelectItem>
                            <SelectItem value="0">{t("weekdaySunday")}</SelectItem>
                          </SelectContent>
                        </Select>
                      </>
                    ) : null}
                    <Label className="text-xs">{t("scheduleTime")}</Label>
                    <Input type="time" value={scheduleForm.scheduleTime} onChange={(event) => setScheduleForm((prev) => ({ ...prev, scheduleTime: event.target.value }))} />
                  </>
                ) : null}
                {scheduleForm.scheduleType === "cron" ? (
                  <>
                    <Label className="text-xs">{t("scheduleCronExpression")}</Label>
                    <Input value={scheduleForm.cronExpression} onChange={(event) => setScheduleForm((prev) => ({ ...prev, cronExpression: event.target.value }))} />
                  </>
                ) : null}
                <div className="flex min-h-11 items-center justify-between rounded-lg border border-border/70 px-3">
                  <span className="text-sm">{t("enabled")}</span>
                  <Switch checked={scheduleForm.enabled} onCheckedChange={(enabled) => setScheduleForm((prev) => ({ ...prev, enabled }))} />
                </div>
                <Button type="button" className="w-full" disabled={pending} onClick={() => void saveSchedule()}>
                  {t("scheduleCreate")}
                </Button>
              </CardContent>
            </Card>
          </aside>
        </section>
      </div>
    </main>
  );
}
