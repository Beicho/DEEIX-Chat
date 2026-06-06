import type { PoWProof } from "@/shared/security/pow-client";
import { bytesToBase64URL, randomBase64URL, sha256Base64URLBytes, textToArrayBuffer } from "@/shared/security/encoding";

export type RequestProof = {
  keyId: string;
  sessionId: string;
  timestamp: number;
  nonce: string;
  bodyHash: string;
  signature: string;
  powProof?: PoWProof;
};

export async function signRequestProof(input: {
  method: string;
  path: string;
  bodyBytes: Uint8Array;
  sessionID: string;
  keyID: string;
  privateKey: CryptoKey;
  powProof?: PoWProof;
}): Promise<RequestProof> {
  const timestamp = Date.now();
  const nonce = randomBase64URL(16);
  const { pathname, query } = splitPath(input.path);
  const bodyHash = await sha256Base64URLBytes(input.bodyBytes);
  const canonical = [
    "DEEIX-PROOF-v1",
    input.method.toUpperCase(),
    pathname,
    query,
    String(timestamp),
    nonce,
    bodyHash,
    input.sessionID,
    input.keyID,
  ].join("\n");
  const signature = await crypto.subtle.sign(
    { name: "ECDSA", hash: "SHA-256" },
    input.privateKey,
    textToArrayBuffer(canonical),
  );

  return {
    keyId: input.keyID,
    sessionId: input.sessionID,
    timestamp,
    nonce,
    bodyHash,
    signature: bytesToBase64URL(signature),
    powProof: input.powProof,
  };
}

function splitPath(path: string): { pathname: string; query: string } {
  const queryStart = path.indexOf("?");
  if (queryStart === -1) {
    return { pathname: path, query: "" };
  }
  return {
    pathname: path.slice(0, queryStart),
    query: path.slice(queryStart + 1),
  };
}
