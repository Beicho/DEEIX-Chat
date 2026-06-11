import type { ProfileDraft } from "@/features/settings/types/settings";
import { DEFAULT_LOCALE } from "@/i18n/config";
import { resolveLocalizedErrorMessage } from "@/i18n/resolve-error-message";
import type { UserDTO } from "@/shared/api/auth.types";

function normalizeString(value: unknown, fallback = ""): string {
  if (typeof value !== "string") {
    return fallback;
  }

  const normalizedValue = value.trim();
  return normalizedValue || fallback;
}

export function createDraftFromUser(user?: UserDTO | null): ProfileDraft {
  return {
    avatarUrl: normalizeString(user?.avatarURL),
    displayName: normalizeString(user?.displayName),
    timezone: normalizeString(user?.timezone, "Etc/UTC"),
    locale: normalizeString(user?.locale, DEFAULT_LOCALE),
    profilePreferences: normalizeString(user?.profilePreferences),
  };
}

export function isProfileDraftEqual(left: ProfileDraft, right: ProfileDraft): boolean {
  return (
    left.avatarUrl === right.avatarUrl &&
    left.displayName === right.displayName &&
    left.timezone === right.timezone &&
    left.locale === right.locale &&
    left.profilePreferences === right.profilePreferences
  );
}

export function resolveSettingsErrorMessage(error: unknown): string {
  return resolveLocalizedErrorMessage(error);
}
