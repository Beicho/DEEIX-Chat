import * as assert from "node:assert/strict";

import { isGlobalShortcutEvent } from "./platform-shortcuts";

assert.equal(
  isGlobalShortcutEvent(
    {
      key: "/",
      shiftKey: false,
      altKey: false,
      ctrlKey: true,
      metaKey: false,
      targetTagName: "body",
    },
    { applePlatform: false },
  ),
  true,
);

assert.equal(
  isGlobalShortcutEvent(
    {
      key: "/",
      shiftKey: false,
      altKey: false,
      ctrlKey: false,
      metaKey: true,
      targetTagName: "body",
    },
    { applePlatform: true },
  ),
  true,
);

assert.equal(
  isGlobalShortcutEvent(
    {
      key: "/",
      shiftKey: false,
      altKey: false,
      ctrlKey: true,
      metaKey: false,
      targetTagName: "textarea",
    },
    { applePlatform: false },
  ),
  false,
);

assert.equal(
  isGlobalShortcutEvent(
    {
      key: "Escape",
      shiftKey: false,
      altKey: false,
      ctrlKey: false,
      metaKey: false,
      targetTagName: "textarea",
    },
    { applePlatform: false },
  ),
  true,
);
