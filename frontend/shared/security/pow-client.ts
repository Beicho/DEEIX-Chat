import { resolveApiBaseURL } from "@/shared/api/http-client";
import { sha256Hex } from "@/shared/security/encoding";

export type PoWChallenge = {
  challenge: string;
  difficulty: number;
  action: string;
  expiresAt: string;
};

export type PoWProof = {
  challenge: string;
  nonce: string;
  hash: string;
  difficulty: number;
};

export async function fetchPoWChallenge(accessToken: string, action: string): Promise<PoWChallenge> {
  const response = await fetch(`${resolveApiBaseURL()}/api/v1/security/pow/challenge?action=${encodeURIComponent(action)}`, {
    method: "POST",
    credentials: "include",
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });
  if (!response.ok) {
    throw new Error("pow challenge failed");
  }
  const payload = (await response.json()) as { data?: PoWChallenge };
  if (!payload.data?.challenge) {
    throw new Error("pow challenge missing");
  }
  return payload.data;
}

export async function solvePoW(challenge: PoWChallenge): Promise<PoWProof> {
  if (typeof window !== "undefined" && typeof Worker !== "undefined") {
    try {
      return await solvePoWWithWorker(challenge);
    } catch {
      return solvePoWInline(challenge);
    }
  }
  return solvePoWInline(challenge);
}

function solvePoWWithWorker(challenge: PoWChallenge): Promise<PoWProof> {
  return new Promise((resolve, reject) => {
    const worker = new Worker(new URL("./pow.worker.ts", import.meta.url), { type: "module" });
    const timeout = window.setTimeout(() => {
      worker.terminate();
      reject(new Error("pow timeout"));
    }, 15000);
    worker.onmessage = (event: MessageEvent<{ nonce?: string; hash?: string }>) => {
      if (typeof event.data.nonce === "string" && typeof event.data.hash === "string") {
        window.clearTimeout(timeout);
        worker.terminate();
        resolve({
          challenge: challenge.challenge,
          nonce: event.data.nonce,
          hash: event.data.hash,
          difficulty: challenge.difficulty,
        });
      }
    };
    worker.onerror = () => {
      window.clearTimeout(timeout);
      worker.terminate();
      reject(new Error("pow worker failed"));
    };
    worker.postMessage({ challenge: challenge.challenge, difficulty: challenge.difficulty });
  });
}

async function solvePoWInline(challenge: PoWChallenge): Promise<PoWProof> {
  const target = "0".repeat(Math.max(1, challenge.difficulty));
  for (let nonce = 0; ; nonce += 1) {
    const nonceText = String(nonce);
    const hash = await sha256Hex(`${challenge.challenge}${nonceText}`);
    if (hash.startsWith(target)) {
      return {
        challenge: challenge.challenge,
        nonce: nonceText,
        hash,
        difficulty: challenge.difficulty,
      };
    }
    if (nonce > 0 && nonce % 1000 === 0) {
      await new Promise((resolve) => window.setTimeout(resolve, 0));
    }
  }
}
