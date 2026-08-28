import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const projectLayoutURL = new URL(
  "../features/layouts/components/sections/project-layout.tsx",
  import.meta.url,
);

test("project workspace keeps sidebar conversation consumers inside their provider", async () => {
  const source = await readFile(projectLayoutURL, "utf8");
  const importStatement =
    'import { SidebarConversationsProvider } from "@/entities/conversation";';
  const providerOpen = source.indexOf("<SidebarConversationsProvider");
  const workspaceContent = source.indexOf("<ProjectLayoutShell>");
  const providerClose = source.indexOf("</SidebarConversationsProvider>");

  assert.ok(source.includes(importStatement), "project layout must import SidebarConversationsProvider directly");
  assert.ok(providerOpen >= 0, "project layout must mount SidebarConversationsProvider");
  assert.ok(workspaceContent > providerOpen, "project workspace content must be inside SidebarConversationsProvider");
  assert.ok(providerClose > workspaceContent, "SidebarConversationsProvider must close after project workspace content");
});
