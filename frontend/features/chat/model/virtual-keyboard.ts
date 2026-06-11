export function resolveVisualViewportKeyboardInset({
  layoutHeight,
  visualHeight,
  offsetTop = 0,
}: {
  layoutHeight: number;
  visualHeight: number;
  offsetTop?: number;
}) {
  return Math.max(0, Math.round(layoutHeight - visualHeight - Math.max(0, offsetTop)));
}
