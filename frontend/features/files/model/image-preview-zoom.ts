export const IMAGE_PREVIEW_MIN_ZOOM = 0.5;
export const IMAGE_PREVIEW_MAX_ZOOM = 2.5;
export const IMAGE_PREVIEW_DEFAULT_ZOOM = 0.8;
export const IMAGE_PREVIEW_DOUBLE_TAP_ZOOM = 1.6;

export type Point = {
  x: number;
  y: number;
};

export function clampImagePreviewZoom(value: number): number {
  if (!Number.isFinite(value)) {
    return IMAGE_PREVIEW_DEFAULT_ZOOM;
  }
  return Math.min(IMAGE_PREVIEW_MAX_ZOOM, Math.max(IMAGE_PREVIEW_MIN_ZOOM, value));
}

export function getPointerDistance(first: Point, second: Point): number {
  return Math.hypot(second.x - first.x, second.y - first.y);
}

export function resolveDoubleTapImageZoom(currentZoom: number): number {
  return currentZoom >= IMAGE_PREVIEW_DOUBLE_TAP_ZOOM ? IMAGE_PREVIEW_DEFAULT_ZOOM : IMAGE_PREVIEW_DOUBLE_TAP_ZOOM;
}

export function resolvePinchImageZoom({
  startDistance,
  currentDistance,
  startZoom,
}: {
  startDistance: number;
  currentDistance: number;
  startZoom: number;
}): number {
  if (!Number.isFinite(startDistance) || !Number.isFinite(currentDistance) || startDistance <= 0) {
    return clampImagePreviewZoom(startZoom);
  }
  return clampImagePreviewZoom(startZoom * (currentDistance / startDistance));
}
