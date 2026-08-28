import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const projectLayoutURL = new URL(
  "../features/layouts/components/sections/project-layout.tsx",
  import.meta.url,
);

test("project workspace keeps sidebar recents consumers inside their provider", async () => {
  const source = await readFile(projectLayoutURL, "utf8");
  const importStatement =
    'import { SidebarRecentsProvider } from "@/features/recent/context/sidebar-recents-context";';
  const providerOpen = source.indexOf("<SidebarRecentsProvider>");
  const workspaceContent = source.indexOf("<ProjectLayoutShell>");
  const providerClose = source.indexOf("</SidebarRecentsProvider>");

  assert.ok(source.includes(importStatement), "project layout must import SidebarRecentsProvider directly");
  assert.ok(providerOpen >= 0, "project layout must mount SidebarRecentsProvider");
  assert.ok(workspaceContent > providerOpen, "project workspace content must be inside SidebarRecentsProvider");
  assert.ok(providerClose > workspaceContent, "SidebarRecentsProvider must close after project workspace content");
});
