import * as assert from "node:assert/strict";

import {
  filterModelPickerGroups,
  recordRecentModelSelection,
  resolveRecentModelOptions,
} from "./model-picker-utils";

const groups = [
  {
    vendor: "openai",
    label: "OpenAI",
    icon: "openai",
    items: [
      { platformModelName: "gpt-5.4", vendor: "openai", icon: "openai" },
      { platformModelName: "gpt-image-2", vendor: "openai", icon: "openai" },
    ],
  },
  {
    vendor: "anthropic",
    label: "Anthropic",
    icon: "claude",
    items: [
      { platformModelName: "claude-opus-4-8", vendor: "anthropic", icon: "claude" },
    ],
  },
];

assert.deepEqual(
  filterModelPickerGroups(groups, "image").map((group) => ({
    vendor: group.vendor,
    items: group.items.map((item) => item.platformModelName),
  })),
  [{ vendor: "openai", items: ["gpt-image-2"] }],
);

assert.deepEqual(
  filterModelPickerGroups(groups, "anth").map((group) => ({
    vendor: group.vendor,
    items: group.items.map((item) => item.platformModelName),
  })),
  [{ vendor: "anthropic", items: ["claude-opus-4-8"] }],
);

assert.deepEqual(
  recordRecentModelSelection(["gpt-5.4", "claude-opus-4-8"], "gpt-image-2", 3),
  ["gpt-image-2", "gpt-5.4", "claude-opus-4-8"],
);

assert.deepEqual(
  recordRecentModelSelection(["gpt-5.4", "claude-opus-4-8", "gpt-image-2"], "claude-opus-4-8", 2),
  ["claude-opus-4-8", "gpt-5.4"],
);

assert.deepEqual(
  resolveRecentModelOptions(groups.flatMap((group) => group.items), ["missing", "claude-opus-4-8", "gpt-5.4"]).map(
    (item) => item.platformModelName,
  ),
  ["claude-opus-4-8", "gpt-5.4"],
);
