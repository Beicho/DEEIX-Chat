async function sha256Hex(input: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", textToArrayBuffer(input));
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function textToArrayBuffer(input: string): ArrayBuffer {
  const bytes = new TextEncoder().encode(input);
  const copy = new Uint8Array(bytes.byteLength);
  copy.set(bytes);
  return copy.buffer;
}

self.onmessage = async (event: MessageEvent<{ challenge: string; difficulty: number }>) => {
  const { challenge, difficulty } = event.data;
  const target = "0".repeat(Math.max(1, difficulty));
  for (let nonce = 0; ; nonce += 1) {
    const nonceText = String(nonce);
    const hash = await sha256Hex(`${challenge}${nonceText}`);
    if (hash.startsWith(target)) {
      self.postMessage({ nonce: nonceText, hash });
      return;
    }
    if (nonce > 0 && nonce % 1000 === 0) {
      self.postMessage({ progress: nonce });
    }
  }
};

export {};
