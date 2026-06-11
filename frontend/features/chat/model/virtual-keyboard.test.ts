import { equal } from "node:assert/strict";

import { resolveVisualViewportKeyboardInset } from "./virtual-keyboard";

equal(resolveVisualViewportKeyboardInset({ layoutHeight: 800, visualHeight: 500, offsetTop: 0 }), 300);
equal(resolveVisualViewportKeyboardInset({ layoutHeight: 800, visualHeight: 500, offsetTop: 120 }), 180);
equal(resolveVisualViewportKeyboardInset({ layoutHeight: 800, visualHeight: 820, offsetTop: 0 }), 0);
equal(resolveVisualViewportKeyboardInset({ layoutHeight: 800, visualHeight: 500, offsetTop: -20 }), 300);
