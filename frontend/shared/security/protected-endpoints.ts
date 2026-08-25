type ProtectedEndpoint = {
  method: string;
  pattern: RegExp;
  action: string;
};

const protectedEndpoints: ProtectedEndpoint[] = [
  { method: "POST", pattern: /^\/api\/v1\/conversations\/[^/]+\/messages$/, action: "send_message" },
  { method: "POST", pattern: /^\/api\/v1\/conversations\/[^/]+\/messages\/stream$/, action: "send_message" },
  { method: "POST", pattern: /^\/api\/v1\/conversations\/[^/]+\/media\/images\/generations\/stream$/, action: "generate_image" },
  { method: "POST", pattern: /^\/api\/v1\/conversations\/[^/]+\/media\/images\/edits\/stream$/, action: "generate_image" },
  { method: "POST", pattern: /^\/api\/v1\/conversations\/[^/]+\/media\/videos\/generations\/stream$/, action: "generate_video" },
  { method: "POST", pattern: /^\/api\/v1\/files$/, action: "upload_file" },
  { method: "POST", pattern: /^\/api\/v1\/conversation-runs\/[^/]+\/cancel$/, action: "cancel_generation" },
];

export function protectedAction(method: string | undefined, path: string): string | null {
  const normalizedMethod = (method || "GET").toUpperCase();
  const pathname = path.split("?")[0] || path;
  const match = protectedEndpoints.find((endpoint) => endpoint.method === normalizedMethod && endpoint.pattern.test(pathname));
  return match?.action ?? null;
}
