export function bytesToBase64URL(bytes: ArrayBuffer | Uint8Array): string {
  const view = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  let binary = "";
  for (const byte of view) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

export function bytesToHex(bytes: ArrayBuffer | Uint8Array): string {
  const view = bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  return Array.from(view, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export function randomBase64URL(byteLength = 16): string {
  const bytes = new Uint8Array(byteLength);
  crypto.getRandomValues(bytes);
  return bytesToBase64URL(bytes);
}

export function bytesToArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  const copy = new Uint8Array(bytes.byteLength);
  copy.set(bytes);
  return copy.buffer;
}

export function textToArrayBuffer(input: string): ArrayBuffer {
  return bytesToArrayBuffer(new TextEncoder().encode(input));
}

export async function sha256Base64URL(input: string): Promise<string> {
  if (!input) {
    return "";
  }
  const digest = await crypto.subtle.digest("SHA-256", textToArrayBuffer(input));
  return bytesToBase64URL(digest);
}

export async function sha256Base64URLBytes(input: Uint8Array): Promise<string> {
  if (!input.byteLength) {
    return "";
  }
  const digest = await crypto.subtle.digest("SHA-256", bytesToArrayBuffer(input));
  return bytesToBase64URL(digest);
}

export async function sha256Hex(input: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", textToArrayBuffer(input));
  return bytesToHex(digest);
}
