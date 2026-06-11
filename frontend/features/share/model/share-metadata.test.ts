import { equal } from "node:assert/strict";

import { buildShareMetadataDescription, resolveShareCanonicalPath } from "./share-metadata";

equal(resolveShareCanonicalPath("abc123"), "/share/abc123");
equal(resolveShareCanonicalPath(""), "/share");

equal(
  buildShareMetadataDescription({
    fallback: "Shared conversation",
    messages: [
      { role: "system", content: "hidden" },
      { role: "user", content: "# Hello\n\nThis is **visible** text." },
    ],
  }),
  "Hello This is visible text.",
);

equal(
  buildShareMetadataDescription({
    fallback: "Shared conversation",
    messages: [{ role: "assistant", content: "x".repeat(220) }],
  }).length,
  160,
);

equal(
  buildShareMetadataDescription({
    fallback: "Shared conversation",
    messages: [],
  }),
  "Shared conversation",
);
