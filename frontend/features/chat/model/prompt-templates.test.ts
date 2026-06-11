import { deepEqual, equal } from "node:assert/strict";

import {
  buildDraftFromPromptTemplate,
  filterPromptTemplates,
  normalizePromptTemplate,
  recordPromptTemplateUsage,
} from "./prompt-templates";

const templates = [
  normalizePromptTemplate({ id: "summarize", title: "Summarize", body: "Summarize this:", category: "work" }),
  normalizePromptTemplate({ id: "rewrite", title: "Rewrite clearly", body: "Rewrite this text:", category: "writing" }),
  normalizePromptTemplate({ id: "plan", title: "Plan tasks", body: "Turn this into tasks:", category: "work" }),
];

deepEqual(filterPromptTemplates(templates, "rew").map((item) => item.id), ["rewrite"]);
deepEqual(filterPromptTemplates(templates, "work").map((item) => item.id), ["summarize", "plan"]);
deepEqual(filterPromptTemplates(templates, "/plan").map((item) => item.id), ["plan"]);
deepEqual(filterPromptTemplates(templates, " /rew").map((item) => item.id), ["rewrite"]);

equal(buildDraftFromPromptTemplate("", templates[0]), "Summarize this:");
equal(buildDraftFromPromptTemplate("Existing text", templates[0]), "Summarize this:\n\nExisting text");
equal(buildDraftFromPromptTemplate("/rew Existing text", templates[1]), "Rewrite this text:\n\nExisting text");
equal(buildDraftFromPromptTemplate("   /rew Existing text", templates[1]), "Rewrite this text:\n\nExisting text");

deepEqual(recordPromptTemplateUsage(["plan", "summarize"], "rewrite", 3), ["rewrite", "plan", "summarize"]);
deepEqual(recordPromptTemplateUsage(["plan", "summarize", "rewrite"], "summarize", 2), ["summarize", "plan"]);
