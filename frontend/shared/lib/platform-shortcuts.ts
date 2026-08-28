export function isApplePlatform(): boolean {
  if (typeof navigator === "undefined") return false;
  const nav = navigator as Navigator & { userAgentData?: { platform?: string } };
  const candidates = [
    nav.userAgentData?.platform,
    navigator.platform,
    navigator.userAgent,
  ].filter((value): value is string => Boolean(value));
  return candidates.some((value) => /Mac|iPhone|iPad|iPod|Darwin/i.test(value)) || (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
}

export function platformModifierLabel(): "Command" | "Ctrl" {
  return isApplePlatform() ? "Command" : "Ctrl";
}

export function hasPlatformModifierKey(event: { ctrlKey: boolean; metaKey: boolean }): boolean {
  if (isApplePlatform()) {
    return event.metaKey && !event.ctrlKey;
  }
  return event.ctrlKey && !event.metaKey;
}

function hasPlatformModifierKeyForEvent(
  event: { ctrlKey: boolean; metaKey: boolean },
  applePlatform: boolean,
): boolean {
  if (applePlatform) {
    return event.metaKey && !event.ctrlKey;
  }
  return event.ctrlKey && !event.metaKey;
}

function isEditableShortcutTarget(target: EventTarget | null | undefined, targetTagName?: string): boolean {
  if (target && target instanceof HTMLElement) {
    const tagName = target.tagName.toLowerCase();
    return target.isContentEditable || tagName === "input" || tagName === "textarea" || tagName === "select";
  }
  const normalizedTagName = String(targetTagName ?? "").trim().toLowerCase();
  return normalizedTagName === "input" || normalizedTagName === "textarea" || normalizedTagName === "select";
}

export function platformSendShortcut(): "ctrl_enter" | "meta_enter" {
  return isApplePlatform() ? "meta_enter" : "ctrl_enter";
}

export function isGlobalShortcutEvent(
  event: {
    key: string;
    shiftKey: boolean;
    altKey: boolean;
    ctrlKey: boolean;
    metaKey: boolean;
    target?: EventTarget | null;
    targetTagName?: string;
  },
  options: { applePlatform?: boolean } = {},
): boolean {
  if (event.key === "Escape" && !event.shiftKey && !event.altKey && !event.ctrlKey && !event.metaKey) {
    return true;
  }
  if (isEditableShortcutTarget(event.target, event.targetTagName)) {
    return false;
  }
  const applePlatform = options.applePlatform ?? isApplePlatform();
  return event.key === "/" && !event.altKey && hasPlatformModifierKeyForEvent(event, applePlatform);
}

export function isSendShortcutEvent(
  shortcut: "enter" | "ctrl_enter" | "meta_enter",
  event: {
    key: string;
    shiftKey: boolean;
    altKey: boolean;
    ctrlKey: boolean;
    metaKey: boolean;
  },
): boolean {
  if (event.key !== "Enter" || event.shiftKey || event.altKey) {
    return false;
  }
  if (shortcut === "enter") {
    return !event.ctrlKey && !event.metaKey;
  }
  return hasPlatformModifierKey(event);
}
