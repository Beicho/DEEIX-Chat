import { resolveApiBaseURL } from "@/shared/api/http-client";
import { collectDeviceFingerprint } from "@/shared/security/fingerprint-collector";

const REPORT_KEY = "deeix:fingerprint:last-report";
const REPORT_INTERVAL_MS = 24 * 60 * 60 * 1000;

let reportPromise: Promise<void> | null = null;

export async function reportFingerprintOnce(accessToken: string): Promise<void> {
  if (typeof window === "undefined" || !accessToken || !shouldReport()) {
    return;
  }
  if (!reportPromise) {
    reportPromise = reportFingerprint(accessToken).finally(() => {
      reportPromise = null;
    });
  }
  return reportPromise;
}

export async function attachFingerprintID(headers: Headers): Promise<void> {
  if (typeof window === "undefined" || !headers) {
    return;
  }
  try {
    const fp = await collectDeviceFingerprint();
    if (fp.fingerprintId) {
      headers.set("X-DEEIX-Fingerprint-ID", fp.fingerprintId);
    }
  } catch {
    // Fingerprinting is advisory. Do not block protected user requests.
  }
}

function shouldReport(): boolean {
  try {
    const raw = window.localStorage.getItem(REPORT_KEY);
    const last = raw ? Number.parseInt(raw, 10) : 0;
    return !last || Date.now() - last > REPORT_INTERVAL_MS;
  } catch {
    return true;
  }
}

async function reportFingerprint(accessToken: string): Promise<void> {
  try {
    const fp = await collectDeviceFingerprint();
    const response = await fetch(`${resolveApiBaseURL()}/api/v1/security/fingerprint`, {
      method: "POST",
      credentials: "include",
      headers: {
        Authorization: `Bearer ${accessToken}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(fp),
    });
    if (response.ok) {
      window.localStorage.setItem(REPORT_KEY, String(Date.now()));
    }
  } catch {
    // Retry on the next authenticated page load.
  }
}

