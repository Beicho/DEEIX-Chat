import { resolveLocalizedErrorMessage } from "@/i18n/resolve-error-message";
import type { SettingsGrouped } from "@/shared/api/settings.types";

export type ModerationFieldType = "int" | "bool" | "string" | "password" | "textarea" | "select";

export type ModerationSettingsField = {
  namespace: "moderation";
  key:
    | "enabled"
    | "mode"
    | "fail_strategy"
    | "base_url"
    | "api_key"
    | "model"
    | "threshold"
    | "action"
    | "timeout_seconds"
    | "classifier_template"
    | "auto_window_hours"
    | "auto_limit_threshold"
    | "auto_suspend_threshold"
    | "auto_limit_rpm"
    | "auto_limit_duration_minutes"
    | "output_window_chars";
  label: string;
  description: string;
  type: ModerationFieldType;
  placeholder?: string;
  options?: Array<{ label: string; value: string }>;
};

type ModerationSettingsTranslator = (key: string) => string;

export function buildModerationSettingsFields(t: ModerationSettingsTranslator): ModerationSettingsField[] {
  return [
    {
      namespace: "moderation",
      key: "enabled",
      label: t("settings.fields.enabled.label"),
      description: t("settings.fields.enabled.description"),
      type: "bool",
    },
    {
      namespace: "moderation",
      key: "mode",
      label: t("settings.fields.mode.label"),
      description: t("settings.fields.mode.description"),
      type: "select",
      options: [
        { label: t("settings.mode.moderations"), value: "moderations" },
        { label: t("settings.mode.chatClassifier"), value: "chat_classifier" },
      ],
    },
    {
      namespace: "moderation",
      key: "fail_strategy",
      label: t("settings.fields.failStrategy.label"),
      description: t("settings.fields.failStrategy.description"),
      type: "select",
      options: [
        { label: t("settings.failStrategy.failOpen"), value: "fail_open" },
        { label: t("settings.failStrategy.failClose"), value: "fail_close" },
      ],
    },
    {
      namespace: "moderation",
      key: "base_url",
      label: t("settings.fields.baseURL.label"),
      description: t("settings.fields.baseURL.description"),
      type: "string",
      placeholder: t("settings.fields.baseURL.placeholder"),
    },
    {
      namespace: "moderation",
      key: "api_key",
      label: t("settings.fields.apiKey.label"),
      description: t("settings.fields.apiKey.description"),
      type: "password",
    },
    {
      namespace: "moderation",
      key: "model",
      label: t("settings.fields.model.label"),
      description: t("settings.fields.model.description"),
      type: "string",
      placeholder: "omni-moderation-latest",
    },
    {
      namespace: "moderation",
      key: "threshold",
      label: t("settings.fields.threshold.label"),
      description: t("settings.fields.threshold.description"),
      type: "string",
      placeholder: "0.5",
    },
    {
      namespace: "moderation",
      key: "action",
      label: t("settings.fields.action.label"),
      description: t("settings.fields.action.description"),
      type: "select",
      options: [{ label: t("settings.action.block"), value: "block" }],
    },
    {
      namespace: "moderation",
      key: "timeout_seconds",
      label: t("settings.fields.timeoutSeconds.label"),
      description: t("settings.fields.timeoutSeconds.description"),
      type: "int",
      placeholder: "10",
    },
    {
      namespace: "moderation",
      key: "classifier_template",
      label: t("settings.fields.classifierTemplate.label"),
      description: t("settings.fields.classifierTemplate.description"),
      type: "textarea",
      placeholder: t("settings.fields.classifierTemplate.placeholder"),
    },
    {
      namespace: "moderation",
      key: "auto_window_hours",
      label: t("settings.fields.autoWindowHours.label"),
      description: t("settings.fields.autoWindowHours.description"),
      type: "int",
      placeholder: "24",
    },
    {
      namespace: "moderation",
      key: "auto_limit_threshold",
      label: t("settings.fields.autoLimitThreshold.label"),
      description: t("settings.fields.autoLimitThreshold.description"),
      type: "int",
      placeholder: "0",
    },
    {
      namespace: "moderation",
      key: "auto_suspend_threshold",
      label: t("settings.fields.autoSuspendThreshold.label"),
      description: t("settings.fields.autoSuspendThreshold.description"),
      type: "int",
      placeholder: "0",
    },
    {
      namespace: "moderation",
      key: "auto_limit_rpm",
      label: t("settings.fields.autoLimitRPM.label"),
      description: t("settings.fields.autoLimitRPM.description"),
      type: "int",
      placeholder: "5",
    },
    {
      namespace: "moderation",
      key: "auto_limit_duration_minutes",
      label: t("settings.fields.autoLimitDurationMinutes.label"),
      description: t("settings.fields.autoLimitDurationMinutes.description"),
      type: "int",
      placeholder: "60",
    },
    {
      namespace: "moderation",
      key: "output_window_chars",
      label: t("settings.fields.outputWindowChars.label"),
      description: t("settings.fields.outputWindowChars.description"),
      type: "int",
      placeholder: "800",
    },
  ];
}

export function moderationFieldID(field: ModerationSettingsField): string {
  return `${field.namespace}.${field.key}`;
}

export function flattenModerationSettings(grouped: SettingsGrouped): Record<string, string> {
  const result: Record<string, string> = {};
  for (const item of grouped.moderation ?? []) {
    result[`moderation.${item.key}`] = item.value ?? "";
  }
  return applyModerationDefaults(result);
}

export function applyModerationDefaults(settings: Record<string, string>): Record<string, string> {
  const result = { ...settings };
  const defaults: Record<string, string> = {
    "moderation.enabled": "false",
    "moderation.mode": "moderations",
    "moderation.fail_strategy": "fail_open",
    "moderation.model": "omni-moderation-latest",
    "moderation.threshold": "0.5",
    "moderation.action": "block",
    "moderation.timeout_seconds": "10",
    "moderation.auto_window_hours": "24",
    "moderation.auto_limit_threshold": "0",
    "moderation.auto_suspend_threshold": "0",
    "moderation.auto_limit_rpm": "5",
    "moderation.auto_limit_duration_minutes": "60",
    "moderation.output_window_chars": "800",
  };
  for (const [key, value] of Object.entries(defaults)) {
    if (!(result[key] ?? "").trim()) {
      result[key] = value;
    }
  }
  return result;
}

export function resolveModerationSettingsError(error: unknown): string {
  return resolveLocalizedErrorMessage(error);
}

export function toModerationEditorField(field: ModerationSettingsField) {
  return {
    id: moderationFieldID(field),
    label: field.label,
    description: field.description,
    type: field.type,
    placeholder: field.placeholder,
    options: field.options,
  } as const;
}
