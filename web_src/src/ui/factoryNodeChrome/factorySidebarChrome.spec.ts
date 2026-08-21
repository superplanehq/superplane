import { describe, expect, it } from "vitest";
import {
  factorySidebarCloseButtonClassName,
  factorySidebarFontClassName,
  factorySidebarHeadingClassName,
  factorySidebarInputClassName,
  factorySidebarKindLabelClassName,
  factorySidebarSurfaceClassName,
} from "./factorySidebarChrome";

describe("factorySidebarChrome", () => {
  it("uses Inter and factory theme tokens", () => {
    expect(factorySidebarFontClassName).toContain("factory-sidebar");
    expect(factorySidebarFontClassName).toContain("font-inter");
    expect(factorySidebarSurfaceClassName).toContain("dark:bg-background");
    expect(factorySidebarSurfaceClassName).toContain("text-foreground");
    expect(factorySidebarHeadingClassName).toContain("text-foreground");
    expect(factorySidebarCloseButtonClassName).toContain("rounded-full");
    expect(factorySidebarInputClassName).toContain("dark:bg-background");
    expect(factorySidebarKindLabelClassName("trigger")).toContain("text-[#2563eb]");
    expect(factorySidebarKindLabelClassName("component")).toContain("text-[#16a34a]");
  });
});
