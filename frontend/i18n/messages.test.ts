import * as assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = process.cwd();
const messagesSource = readFileSync(join(root, "i18n/messages.ts"), "utf8");

const requiredNamespaces = ["arena", "checkin", "admin"] as const;

for (const namespace of requiredNamespaces) {
  for (const locale of ["en-US", "zh-CN"]) {
    assert.ok(
      existsSync(join(root, `i18n/messages/${locale}/${namespace}.json`)),
      `${locale}/${namespace}.json exists`,
    );
  }
}

for (const namespace of requiredNamespaces) {
  assert.match(
    messagesSource,
    new RegExp(`messages/en-US/${namespace}\\.json`),
    `${namespace} English messages are imported from en-US`,
  );
  assert.match(
    messagesSource,
    new RegExp(`messages/zh-CN/${namespace}\\.json`),
    `${namespace} zh-CN messages are imported`,
  );
}

for (const namespace of requiredNamespaces) {
  assert.match(messagesSource, new RegExp(`\\b${namespace}:`), `${namespace} is in DEFAULT_MESSAGES`);
}
