import { equal } from "node:assert/strict";

import {
  buildQuotedDraft,
  formatSelectedQuote,
} from "./selection-quote";

equal(formatSelectedQuote("  first line\nsecond line  "), "> first line\n> second line");
equal(formatSelectedQuote("already\n\nspaced"), "> already\n>\n> spaced");
equal(formatSelectedQuote("   "), "");

equal(buildQuotedDraft("", "quoted text"), "> quoted text\n\n");
equal(buildQuotedDraft("Please explain.", "quoted text"), "> quoted text\n\nPlease explain.");
equal(buildQuotedDraft("Draft with tail\n", "quoted text"), "> quoted text\n\nDraft with tail");
