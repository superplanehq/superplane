export type ShortcutModifierLabel = "⌘" | "Ctrl+";

export function isMacPlatform(platform = getNavigatorPlatform()): boolean {
  return platform.toLowerCase().includes("mac");
}

export function getShortcutModifierLabel(platform = getNavigatorPlatform()): ShortcutModifierLabel {
  return isMacPlatform(platform) ? "⌘" : "Ctrl+";
}

export function formatShortcutLabel(key: string, platform = getNavigatorPlatform()): string {
  return `${getShortcutModifierLabel(platform)}${key}`;
}

function getNavigatorPlatform(): string {
  if (typeof navigator === "undefined") {
    return "";
  }

  return navigator.platform;
}
