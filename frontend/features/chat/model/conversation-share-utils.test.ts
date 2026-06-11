import { deepEqual, equal } from "node:assert/strict";

import {
  buildConversationNativeShareData,
  isNativeShareAbortError,
} from "./conversation-share-utils";

deepEqual(
  buildConversationNativeShareData({
    title: "  Project notes  ",
    text: "  Project notes - 3 snapshot messages  ",
    url: "https://example.test/share/abc",
  }),
  {
    title: "Project notes",
    text: "Project notes - 3 snapshot messages",
    url: "https://example.test/share/abc",
  },
);

deepEqual(
  buildConversationNativeShareData({
    title: "",
    text: "",
    url: "https://example.test/share/abc",
  }),
  {
    title: "DEEIX Chat",
    url: "https://example.test/share/abc",
  },
);

equal(isNativeShareAbortError(new DOMException("cancelled", "AbortError")), true);
equal(isNativeShareAbortError(new Error("failed")), false);
