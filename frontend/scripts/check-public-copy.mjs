import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = join(fileURLToPath(new URL("..", import.meta.url)));
const MESSAGE_ROOT = join(ROOT, "i18n", "messages");

const PUBLIC_NAMESPACES = new Set([
  "announcements.json",
  "chat.json",
  "common.json",
  "errors.json",
  "files.json",
  "guide.json",
  "login.json",
  "recent.json",
  "settings.json",
  "share.json",
]);

const BANNED_PATTERNS = [
  /\bRAG\b/i,
  /\bMCP\b/i,
  /\bMIME\b/i,
  /\bEmbedding\b/i,
  /nanousd/i,
  /service_tier/i,
  /routedBindingCode/i,
  /upstream/i,
  /allowlist/i,
  /开发验证码/,
  /调试/,
  /后台/,
  /上游/,
  /白名单/,
  /向量/,
  /全文注入/,
  /语义向量/,
];

function collectStrings(value, path, out) {
  if (typeof value === "string") {
    out.push({ path, value });
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => {
      collectStrings(item, `${path}[${index}]`, out);
    });
    return;
  }
  if (value && typeof value === "object") {
    for (const [key, child] of Object.entries(value)) {
      collectStrings(child, path ? `${path}.${key}` : key, out);
    }
  }
}

const violations = [];

for (const locale of readdirSync(MESSAGE_ROOT)) {
  const localeDir = join(MESSAGE_ROOT, locale);
  for (const fileName of readdirSync(localeDir)) {
    if (!PUBLIC_NAMESPACES.has(fileName)) {
      continue;
    }
    const filePath = join(localeDir, fileName);
    const parsed = JSON.parse(readFileSync(filePath, "utf8"));
    const strings = [];
    collectStrings(parsed, "", strings);
    for (const entry of strings) {
      for (const pattern of BANNED_PATTERNS) {
        if (pattern.test(entry.value)) {
          violations.push(`${locale}/${fileName}:${entry.path}: ${entry.value}`);
          break;
        }
      }
    }
  }
}

if (violations.length > 0) {
  console.error("Public copy contains internal terms:");
  for (const violation of violations) {
    console.error(`- ${violation}`);
  }
  process.exit(1);
}

console.log("Public copy internal-term check passed.");
