import { deepEqual, equal } from "node:assert/strict";

import {
  findConversationMatches,
  nextConversationMatchIndex,
} from "./conversation-find-utils";

const messages = [
  { key: "a", content: "Alpha beta" },
  { key: "b", content: "No match" },
  { key: "c", content: "beta again beta" },
];

deepEqual(findConversationMatches(messages, "beta"), [
  { messageKey: "a", index: 6 },
  { messageKey: "c", index: 0 },
  { messageKey: "c", index: 11 },
]);

equal(nextConversationMatchIndex(0, 3, "next"), 1);
equal(nextConversationMatchIndex(2, 3, "next"), 0);
equal(nextConversationMatchIndex(0, 3, "previous"), 2);
equal(nextConversationMatchIndex(-1, 3, "next"), 0);
