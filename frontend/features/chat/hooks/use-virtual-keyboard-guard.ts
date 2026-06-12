import * as React from "react";

import { resolveVisualViewportKeyboardInset } from "@/features/chat/model/virtual-keyboard";

export function useVirtualKeyboardGuard({
  composerRef,
  messageViewportRef,
  onScrollToLatest,
  disabled = false,
}: {
  composerRef: { current: HTMLElement | null };
  messageViewportRef: { current: HTMLElement | null };
  onScrollToLatest: () => void;
  disabled?: boolean;
}) {
  React.useEffect(() => {
    if (disabled || typeof window === "undefined" || typeof document === "undefined") {
      return;
    }

    const viewport = window.visualViewport;
    const animationFrames = new Set<number>();
    const timers = new Set<number>();

    const isComposerFocused = () => {
      const composer = composerRef.current;
      const activeElement = document.activeElement;
      return Boolean(composer && activeElement instanceof Node && composer.contains(activeElement));
    };

    const syncKeyboardInset = () => {
      const inset = viewport
        ? resolveVisualViewportKeyboardInset({
            layoutHeight: window.innerHeight,
            visualHeight: viewport.height,
            offsetTop: viewport.offsetTop,
          })
        : 0;
      document.documentElement.style.setProperty("--deeix-visual-keyboard-inset", `${inset}px`);
    };

    const keepComposerVisible = () => {
      if (!isComposerFocused()) {
        syncKeyboardInset();
        return;
      }
      syncKeyboardInset();
      const frame = window.requestAnimationFrame(() => {
        animationFrames.delete(frame);
        onScrollToLatest();
        messageViewportRef.current?.scrollTo({
          top: messageViewportRef.current.scrollHeight,
          behavior: "auto",
        });
        composerRef.current?.scrollIntoView({ block: "nearest", inline: "nearest", behavior: "auto" });
      });
      animationFrames.add(frame);
    };

    const scheduleKeepVisible = () => {
      keepComposerVisible();
      for (const delay of [80, 240]) {
        const timer = window.setTimeout(() => {
          timers.delete(timer);
          keepComposerVisible();
        }, delay);
        timers.add(timer);
      }
    };

    const handleFocusIn = (event: FocusEvent) => {
      const composer = composerRef.current;
      if (composer && event.target instanceof Node && composer.contains(event.target)) {
        scheduleKeepVisible();
      }
    };

    document.addEventListener("focusin", handleFocusIn);
    viewport?.addEventListener("resize", scheduleKeepVisible);
    viewport?.addEventListener("scroll", scheduleKeepVisible);

    return () => {
      document.removeEventListener("focusin", handleFocusIn);
      viewport?.removeEventListener("resize", scheduleKeepVisible);
      viewport?.removeEventListener("scroll", scheduleKeepVisible);
      for (const frame of animationFrames) {
        window.cancelAnimationFrame(frame);
      }
      for (const timer of timers) {
        window.clearTimeout(timer);
      }
      document.documentElement.style.removeProperty("--deeix-visual-keyboard-inset");
    };
  }, [composerRef, disabled, messageViewportRef, onScrollToLatest]);
}
