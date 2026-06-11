import { equal } from "node:assert/strict";

import {
  LONG_PASTE_TEXT_THRESHOLD,
  buildPastedTextFileName,
  shouldConvertPasteToTextFile,
} from "./paste-to-file";

equal(shouldConvertPasteToTextFile("a".repeat(LONG_PASTE_TEXT_THRESHOLD - 1)), false);
equal(shouldConvertPasteToTextFile("a".repeat(LONG_PASTE_TEXT_THRESHOLD)), true);
equal(shouldConvertPasteToTextFile(`  ${"a".repeat(LONG_PASTE_TEXT_THRESHOLD)}  `), true);
equal(buildPastedTextFileName(new Date("2026-06-10T12:34:56Z")), "pasted-text-20260610-123456.txt");
