import { equal } from "node:assert/strict";

import {
  clampImagePreviewZoom,
  getPointerDistance,
  resolveDoubleTapImageZoom,
  resolvePinchImageZoom,
} from "./image-preview-zoom";

equal(clampImagePreviewZoom(0.1), 0.5);
equal(clampImagePreviewZoom(1.2), 1.2);
equal(clampImagePreviewZoom(9), 2.5);

equal(resolveDoubleTapImageZoom(0.8), 1.6);
equal(resolveDoubleTapImageZoom(1.7), 0.8);

equal(getPointerDistance({ x: 0, y: 0 }, { x: 3, y: 4 }), 5);
equal(resolvePinchImageZoom({ startDistance: 100, currentDistance: 150, startZoom: 1 }), 1.5);
equal(resolvePinchImageZoom({ startDistance: 100, currentDistance: 10, startZoom: 1 }), 0.5);
equal(resolvePinchImageZoom({ startDistance: 100, currentDistance: 500, startZoom: 1 }), 2.5);
equal(resolvePinchImageZoom({ startDistance: 0, currentDistance: 120, startZoom: 1.1 }), 1.1);
