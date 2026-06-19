import { authedRequest } from "@/shared/api/authed-client";
import { readAccessToken } from "@/shared/auth/session";

export type AlertingConfigView = {
  enabled: boolean;
  enabledNotifiers: string[];
  telegramConfigured: boolean;
  telegramChatId: string;
  webhookConfigured: boolean;
  debounceSeconds: number;
};

export type AlertingConfigUpdate = {
  enabled?: boolean;
  enabledNotifiers?: string[];
  telegramBotToken?: string;
  telegramChatId?: string;
  webhookUrl?: string;
  debounceSeconds?: number;
};

function token(): string {
  const t = readAccessToken();
  if (!t) throw new Error("Not authenticated");
  return t;
}

export async function getAlertingConfig(): Promise<AlertingConfigView> {
  return authedRequest<AlertingConfigView>("/api/v1/admin/alerting/config", {
    accessToken: token(),
  });
}

export async function updateAlertingConfig(payload: AlertingConfigUpdate): Promise<AlertingConfigView> {
  return authedRequest<AlertingConfigView>("/api/v1/admin/alerting/config", {
    method: "PATCH",
    accessToken: token(),
    body: payload,
  });
}

export async function testTelegramAlert(override?: { telegramBotToken?: string; telegramChatId?: string }): Promise<void> {
  await authedRequest<unknown>("/api/v1/admin/alerting/test-telegram", {
    method: "POST",
    accessToken: token(),
    body: override ?? {},
  });
}

export async function testWebhookAlert(override?: { webhookUrl?: string }): Promise<void> {
  await authedRequest<unknown>("/api/v1/admin/alerting/test-webhook", {
    method: "POST",
    accessToken: token(),
    body: override ?? {},
  });
}
