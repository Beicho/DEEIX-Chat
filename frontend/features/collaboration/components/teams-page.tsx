"use client";

import * as React from "react";
import { Users } from "lucide-react";
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
import { Textarea } from "@/components/ui/textarea";
import { addTeamMember, createTeamSpace, listTeamSpaces, removeTeamMember } from "@/shared/api/collaboration";
import type { TeamSpaceDTO } from "@/shared/api/collaboration.types";
import { useAuthSession } from "@/shared/auth/auth-session-context";

function TeamSkeleton() {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="rounded-xl border border-border/70 p-4">
          <Skeleton className="h-5 w-1/2" />
          <Skeleton className="mt-3 h-16 w-full" />
        </div>
      ))}
    </div>
  );
}

export function TeamsPage() {
  const t = useTranslations("collaboration.teams");
  const { accessToken, userStatus } = useAuthSession();
  const [teams, setTeams] = React.useState<TeamSpaceDTO[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [pending, setPending] = React.useState(false);
  const [form, setForm] = React.useState({ name: "", description: "" });
  const [memberForms, setMemberForms] = React.useState<Record<string, { login: string; role: "admin" | "member" }>>({});

  const load = React.useCallback(async () => {
    if (!accessToken || userStatus !== "ready") {
      return;
    }
    setLoading(true);
    try {
      setTeams(await listTeamSpaces(accessToken));
    } catch {
      toast.error(t("loadFailed"));
    } finally {
      setLoading(false);
    }
  }, [accessToken, t, userStatus]);

  React.useEffect(() => {
    void load();
  }, [load]);

  const createTeam = React.useCallback(async () => {
    if (!accessToken || !form.name.trim()) {
      return;
    }
    setPending(true);
    try {
      await createTeamSpace(accessToken, form);
      setForm({ name: "", description: "" });
      toast.success(t("toastCreated"));
      await load();
    } catch {
      toast.error(t("toastCreateFailed"));
    } finally {
      setPending(false);
    }
  }, [accessToken, form, load, t]);

  const updateMemberForm = React.useCallback((teamID: string, patch: Partial<{ login: string; role: "admin" | "member" }>) => {
    setMemberForms((prev) => ({
      ...prev,
      [teamID]: {
        login: "",
        role: "member",
        ...(prev[teamID] ?? {}),
        ...patch,
      },
    }));
  }, []);

  const addMember = React.useCallback(async (team: TeamSpaceDTO) => {
    if (!accessToken) {
      return;
    }
    const draft = memberForms[team.publicID] ?? { login: "", role: "member" as const };
    if (!draft.login.trim()) {
      return;
    }
    setPending(true);
    try {
      await addTeamMember(accessToken, team.publicID, draft);
      updateMemberForm(team.publicID, { login: "" });
      toast.success(t("toastMemberAdded"));
      await load();
    } catch {
      toast.error(t("toastMemberFailed"));
    } finally {
      setPending(false);
    }
  }, [accessToken, load, memberForms, t, updateMemberForm]);

  const removeMember = React.useCallback(async (team: TeamSpaceDTO, userID: number) => {
    if (!accessToken) {
      return;
    }
    setPending(true);
    try {
      await removeTeamMember(accessToken, team.publicID, userID);
      await load();
    } catch {
      toast.error(t("toastMemberFailed"));
    } finally {
      setPending(false);
    }
  }, [accessToken, load, t]);

  return (
    <main className="h-full min-h-0 overflow-auto bg-background text-foreground">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-4 md:px-6 md:py-6">
        <header className="flex flex-col gap-2 border-b border-border/60 pb-4">
          <div className="flex items-center gap-2">
            <Users className="size-4 text-muted-foreground" strokeWidth={1.8} />
            <h1 className="text-lg font-semibold tracking-tight">{t("title")}</h1>
          </div>
          <p className="text-sm text-muted-foreground">{t("description")}</p>
        </header>

        <section className="grid gap-4 lg:grid-cols-[360px_minmax(0,1fr)]">
          <Card className="border-border/70 bg-card">
            <CardHeader className="p-4">
              <CardTitle className="text-sm">{t("createTitle")}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 p-4 pt-0">
              <Label className="text-xs">{t("name")}</Label>
              <Input value={form.name} onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))} />
              <Label className="text-xs">{t("descriptionField")}</Label>
              <Textarea className="min-h-24" value={form.description} onChange={(event) => setForm((prev) => ({ ...prev, description: event.target.value }))} />
              <Button type="button" className="w-full" disabled={pending} onClick={() => void createTeam()}>
                {t("create")}
              </Button>
            </CardContent>
          </Card>

          <div className="min-w-0">
            {loading ? <TeamSkeleton /> : teams.length === 0 ? (
              <CenteredEmptyState title={t("emptyTitle")} description={t("emptyDescription")} />
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {teams.map((team) => {
                  const draft = memberForms[team.publicID] ?? { login: "", role: "member" as const };
                  return (
                    <Card key={team.publicID} className="border-border/70 bg-card">
                      <CardHeader className="p-4">
                        <CardTitle className="truncate text-sm">{team.name}</CardTitle>
                        {team.description ? <p className="line-clamp-2 text-xs text-muted-foreground">{team.description}</p> : null}
                      </CardHeader>
                      <CardContent className="space-y-3 p-4 pt-0">
                        <div className="space-y-2">
                          {team.members.map((member) => {
                            const label = member.role === "owner" ? t("owner") : member.role === "admin" ? t("admin") : t("member");
                            return (
                              <div key={member.userID} className="flex min-h-11 items-center gap-3 rounded-lg border border-border/70 px-3 py-2">
                                <Avatar className="size-8">
                                  <AvatarImage src={member.avatarURL} alt="" />
                                  <AvatarFallback>{(member.displayName || member.username).slice(0, 2).toUpperCase()}</AvatarFallback>
                                </Avatar>
                                <div className="min-w-0 flex-1">
                                  <div className="truncate text-sm">{member.displayName || member.username}</div>
                                  <div className="truncate text-xs text-muted-foreground">{member.username || member.email}</div>
                                </div>
                                <Badge variant="secondary">{label}</Badge>
                                {member.role !== "owner" ? (
                                  <Button type="button" variant="ghost" size="sm" className="min-h-9" disabled={pending} onClick={() => void removeMember(team, member.userID)}>
                                    {t("removeMember")}
                                  </Button>
                                ) : null}
                              </div>
                            );
                          })}
                        </div>
                        <div className="grid gap-2 rounded-lg border border-border/70 p-3">
                          <Label className="text-xs">{t("memberLogin")}</Label>
                          <Input value={draft.login} onChange={(event) => updateMemberForm(team.publicID, { login: event.target.value })} />
                          <Label className="text-xs">{t("memberRole")}</Label>
                          <Select value={draft.role} onValueChange={(role: "admin" | "member") => updateMemberForm(team.publicID, { role })}>
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="member">{t("member")}</SelectItem>
                              <SelectItem value="admin">{t("admin")}</SelectItem>
                            </SelectContent>
                          </Select>
                          <Button type="button" variant="outline" className="min-h-9" disabled={pending} onClick={() => void addMember(team)}>
                            {t("addMember")}
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  );
                })}
              </div>
            )}
          </div>
        </section>
      </div>
    </main>
  );
}
