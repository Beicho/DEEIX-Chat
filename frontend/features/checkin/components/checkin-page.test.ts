import * as assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const source = readFileSync(join(process.cwd(), "features/checkin/components/checkin-page.tsx"), "utf8");

assert.match(source, /h-full min-h-0 overflow-y-auto/, "check-in page owns a vertical scroll root");
assert.match(source, /overscroll-y-contain/, "check-in page contains touch overscroll inside the page");
assert.doesNotMatch(source, /alert\(/, "check-in page should not use blocking browser alerts");
