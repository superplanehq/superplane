import { afterEach, describe, expect, it, vi } from "vitest";
import { formatShortcutLabel, getShortcutModifierLabel, isMacPlatform } from "@/lib/keyboardShortcuts";

describe("keyboardShortcuts", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("detects mac platforms", () => {
    expect(isMacPlatform("MacIntel")).toBe(true);
    expect(isMacPlatform("Win32")).toBe(false);
  });

  it("returns platform-specific modifier labels", () => {
    expect(getShortcutModifierLabel("MacIntel")).toBe("⌘");
    expect(getShortcutModifierLabel("Win32")).toBe("Ctrl+");
  });

  it("formats shortcut labels for the current platform", () => {
    vi.stubGlobal("navigator", { platform: "MacIntel" });
    expect(formatShortcutLabel("B")).toBe("⌘B");

    vi.stubGlobal("navigator", { platform: "Win32" });
    expect(formatShortcutLabel("B")).toBe("Ctrl+B");
  });
});
