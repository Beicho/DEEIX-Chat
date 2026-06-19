import * as assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const source = readFileSync(join(process.cwd(), "features/arena/components/arena-page.tsx"), "utf8");

assert.match(source, /searchModels/, "arena has model search");
assert.match(source, /selectedModels/, "arena shows selected model count");
assert.match(source, /readyToVote/, "arena explains when voting is ready");
assert.match(source, /voteFailed/, "arena surfaces vote failures");
assert.match(source, /h-full min-h-0 overflow-y-auto/, "arena owns a vertical scroll root");
