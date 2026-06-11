import { equal } from "node:assert/strict";

import {
  normalizeSpeechText,
  resolveSpeechAction,
} from "./message-tts-utils";

equal(
  normalizeSpeechText("Hello **world**\n```ts\nconsole.log('x')\n```\n[link](https://example.test)"),
  "Hello world link",
);

equal(resolveSpeechAction({ supported: false, speaking: false, paused: false, messageKey: "a", activeMessageKey: null }), "unsupported");
equal(resolveSpeechAction({ supported: true, speaking: false, paused: false, messageKey: "a", activeMessageKey: null }), "play");
equal(resolveSpeechAction({ supported: true, speaking: true, paused: false, messageKey: "a", activeMessageKey: "a" }), "pause");
equal(resolveSpeechAction({ supported: true, speaking: true, paused: true, messageKey: "a", activeMessageKey: "a" }), "resume");
equal(resolveSpeechAction({ supported: true, speaking: true, paused: false, messageKey: "b", activeMessageKey: "a" }), "play");
