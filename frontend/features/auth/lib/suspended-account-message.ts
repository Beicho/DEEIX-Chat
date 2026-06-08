import { ApiError } from "@/shared/api/http-client";

type SuspendedAccountDetails = {
  reason?: unknown;
  detail?: unknown;
};

export function isSuspendedAccountError(error: unknown): boolean {
  return error instanceof ApiError && error.errorCode === "auth.account_suspended";
}

export function getSuspendedAccountReason(error: unknown): string | null {
  if (!isSuspendedAccountError(error) || !(error instanceof ApiError)) {
    return null;
  }
  const details = error.details && typeof error.details === "object" ? error.details as SuspendedAccountDetails : {};
  const reason = typeof details.reason === "string" ? details.reason.trim() : "";
  const detail = typeof details.detail === "string" ? details.detail.trim() : "";
  return reason || detail || null;
}
