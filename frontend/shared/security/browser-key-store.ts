import { resolveApiBaseURL } from "@/shared/api/http-client";

type StoredBrowserKey = {
  sessionID: string;
  keyID: string;
  keyPair: CryptoKeyPair;
  createdAt: number;
};

const DB_NAME = "deeix-browser-proof";
const STORE_NAME = "browser-keys";
// 服务端密钥默认保存 7 天；提前轮换，避免本地仍缓存着 Redis 已过期的密钥。
const BROWSER_KEY_MAX_AGE_MS = 6 * 24 * 60 * 60 * 1000;

let dbPromise: Promise<IDBDatabase> | null = null;
const ensurePromises = new Map<string, Promise<StoredBrowserKey>>();

export function ensureBrowserKey(accessToken: string, sessionID: string): Promise<StoredBrowserKey> {
  const existingPromise = ensurePromises.get(sessionID);
  if (existingPromise) {
    return existingPromise;
  }
  const promise = ensureBrowserKeyInternal(accessToken, sessionID).finally(() => {
    ensurePromises.delete(sessionID);
  });
  ensurePromises.set(sessionID, promise);
  return promise;
}

async function ensureBrowserKeyInternal(accessToken: string, sessionID: string): Promise<StoredBrowserKey> {
  const existing = await loadBrowserKey(sessionID);
  if (existing && isBrowserKeyFresh(existing)) {
    return existing;
  }
  if (existing) {
    await deleteBrowserKey(sessionID);
  }

  const keyPair = await crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    false,
    ["sign", "verify"],
  );
  const publicKeyJwk = await crypto.subtle.exportKey("jwk", keyPair.publicKey);
  const keyID = await bootstrapBrowserKey(accessToken, sessionID, publicKeyJwk);
  const stored = { sessionID, keyID, keyPair, createdAt: Date.now() };
  await saveBrowserKey(stored);
  return stored;
}

export async function invalidateBrowserKey(sessionID: string): Promise<void> {
  if (!sessionID) {
    return;
  }
  await deleteBrowserKey(sessionID);
}

function isBrowserKeyFresh(item: StoredBrowserKey): boolean {
  return Number.isFinite(item.createdAt) && item.createdAt > 0 && Date.now() - item.createdAt < BROWSER_KEY_MAX_AGE_MS;
}

async function bootstrapBrowserKey(accessToken: string, sessionID: string, publicKeyJwk: JsonWebKey): Promise<string> {
  const response = await fetch(`${resolveApiBaseURL()}/api/v1/security/browser-key/bootstrap`, {
    method: "POST",
    credentials: "include",
    headers: {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      sessionId: sessionID,
      publicKeyJwk: JSON.stringify(publicKeyJwk),
    }),
  });
  if (!response.ok) {
    throw new Error("browser key bootstrap failed");
  }
  const payload = (await response.json()) as { data?: { keyId?: string } };
  const keyID = payload.data?.keyId || "";
  if (!keyID) {
    throw new Error("browser key id missing");
  }
  return keyID;
}

async function openDB(): Promise<IDBDatabase> {
  if (dbPromise) {
    return dbPromise;
  }
  dbPromise = new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "sessionID" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("indexeddb open failed"));
  });
  return dbPromise;
}

async function loadBrowserKey(sessionID: string): Promise<StoredBrowserKey | null> {
  if (typeof indexedDB === "undefined" || !sessionID) {
    return null;
  }
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readonly");
    const request = tx.objectStore(STORE_NAME).get(sessionID);
    request.onsuccess = () => resolve((request.result as StoredBrowserKey | undefined) ?? null);
    request.onerror = () => reject(request.error || new Error("indexeddb read failed"));
  });
}

async function saveBrowserKey(item: StoredBrowserKey): Promise<void> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readwrite");
    tx.objectStore(STORE_NAME).put(item);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error || new Error("indexeddb write failed"));
  });
}

async function deleteBrowserKey(sessionID: string): Promise<void> {
  if (typeof indexedDB === "undefined" || !sessionID) {
    return;
  }
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readwrite");
    tx.objectStore(STORE_NAME).delete(sessionID);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error || new Error("indexeddb delete failed"));
  });
}
