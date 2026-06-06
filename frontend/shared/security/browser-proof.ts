import { protectedAction } from "@/shared/security/protected-endpoints";
import { fetchPoWChallenge, solvePoW } from "@/shared/security/pow-client";
import { ensureBrowserKey } from "@/shared/security/browser-key-store";
import { signRequestProof } from "@/shared/security/request-signer";
import { attachFingerprintID } from "@/shared/security/fingerprint-reporter";
import { bytesToArrayBuffer } from "@/shared/security/encoding";

export const BROWSER_PROOF_HEADER = "X-DEEIX-Proof";

export function serializeRequestBody(body: unknown): string {
  if (typeof body === "string") {
    return body;
  }
  if (typeof body === "undefined" || body === null) {
    return "";
  }
  if (typeof FormData !== "undefined" && body instanceof FormData) {
    return "";
  }
  return JSON.stringify(body);
}

export type PreparedProofBody = {
  body: BodyInit | undefined;
  bodyBytes: Uint8Array;
  contentType?: string;
};

export async function prepareProofBody(body: unknown): Promise<PreparedProofBody> {
  if (typeof body === "undefined" || body === null) {
    return { body: undefined, bodyBytes: new Uint8Array() };
  }
  if (typeof body === "string") {
    return { body, bodyBytes: new TextEncoder().encode(body) };
  }
  if (typeof FormData !== "undefined" && body instanceof FormData) {
    const request = new Request("https://deeix.local/upload", {
      method: "POST",
      body,
    });
    const bytes = new Uint8Array(await request.arrayBuffer());
    return {
      body: bytesToArrayBuffer(bytes),
      bodyBytes: bytes,
      contentType: request.headers.get("content-type") || undefined,
    };
  }
  if (body instanceof ArrayBuffer) {
    return { body, bodyBytes: new Uint8Array(body) };
  }
  if (ArrayBuffer.isView(body)) {
    const view = body as ArrayBufferView;
    const bytes = new Uint8Array(view.buffer, view.byteOffset, view.byteLength);
    return { body: bytesToArrayBuffer(bytes), bodyBytes: bytes };
  }
  const serialized = serializeRequestBody(body);
  return { body: serialized, bodyBytes: new TextEncoder().encode(serialized) };
}

export async function attachBrowserProof(input: {
  path: string;
  method?: string;
  bodyBytes: Uint8Array;
  accessToken: string;
  sessionID: string;
  headers: Headers;
}): Promise<void> {
  if (typeof window === "undefined" || typeof crypto === "undefined" || !crypto.subtle || !input.accessToken || !input.sessionID) {
    return;
  }
  const method = (input.method || "GET").toUpperCase();
  const action = protectedAction(method, input.path);
  if (!action) {
    return;
  }

  const fingerprintPromise = attachFingerprintID(input.headers);
  const [challenge, key] = await Promise.all([
    fetchPoWChallenge(input.accessToken, action),
    ensureBrowserKey(input.accessToken, input.sessionID),
  ]);
  const powProof = await solvePoW(challenge);
  const proof = await signRequestProof({
    method,
    path: input.path,
    bodyBytes: input.bodyBytes,
    sessionID: input.sessionID,
    keyID: key.keyID,
    privateKey: key.keyPair.privateKey,
    powProof,
  });
  input.headers.set(BROWSER_PROOF_HEADER, JSON.stringify(proof));
  await fingerprintPromise;
}
