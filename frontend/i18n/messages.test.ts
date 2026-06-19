import * as assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = process.cwd();
const messagesSource = readFileSync(join(root, "i18n/messages.ts"), "utf8");

const requiredNamespaces = ["arena", "checkin", "admin"] as const;
const requiredNavigationKeys = ["newChat", "search", "recent", "checkin", "arena", "announcements", "bookmarks", "files"] as const;
const requiredAdminSectionKeys = [
  "dashboard",
  "accounts",
  "security",
  "channels",
  "models",
  "toolSettings",
  "billing",
  "checkin",
  "alerting",
  "arena",
  "announcements",
  "moderation",
  "branding",
  "logs",
  "loginSettings",
  "conversationSettings",
  "chatFiles",
  "about",
] as const;

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

for (const locale of ["en-US", "zh-CN"]) {
  const common = JSON.parse(readFileSync(join(root, `i18n/messages/${locale}/common.json`), "utf8")) as {
    navigation?: Record<string, string>;
  };
  for (const key of requiredNavigationKeys) {
    assert.ok(common.navigation?.[key], `${locale} common.navigation.${key} exists`);
    assert.notEqual(common.navigation?.[key], key, `${locale} common.navigation.${key} is translated`);
  }

  const adminUsers = JSON.parse(readFileSync(join(root, `i18n/messages/${locale}/admin-users.json`), "utf8")) as {
    sections?: Record<string, string>;
  };
  for (const key of requiredAdminSectionKeys) {
    assert.ok(adminUsers.sections?.[key], `${locale} adminUsers.sections.${key} exists`);
  }
}
