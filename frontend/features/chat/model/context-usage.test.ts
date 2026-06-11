import { equal } from "node:assert/strict";

import {
  CONTEXT_USAGE_APPROX_CHARS_PER_TOKEN,
  estimateConversationTokens,
  resolveContextUsageTone,
} from "./context-usage";

equal(estimateConversationTokens(["a".repeat(CONTEXT_USAGE_APPROX_CHARS_PER_TOKEN * 100)]), 100);
equal(resolveContextUsageTone(0.5), "default");
equal(resolveContextUsageTone(0.78), "warning");
equal(resolveContextUsageTone(0.92), "danger");
