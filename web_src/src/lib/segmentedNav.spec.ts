import { describe, expect, it } from "vitest";
import {
  SEGMENTED_NAV_CLASSES,
  SEGMENTED_NAV_XS_CLASSES,
  segmentedNavClassName,
  segmentedNavTabClassName,
} from "./segmentedNav";

describe("segmentedNav", () => {
  it("uses the standard rounded segmented nav track", () => {
    expect(SEGMENTED_NAV_CLASSES).toContain("rounded-full");
    expect(SEGMENTED_NAV_CLASSES).toContain("bg-slate-100");
  });

  it("uses a smaller xs track", () => {
    expect(SEGMENTED_NAV_XS_CLASSES).toContain("h-6");
    expect(segmentedNavClassName("xs")).toBe(SEGMENTED_NAV_XS_CLASSES);
  });

  it("uses rounded tabs with 13px text", () => {
    const activeTab = segmentedNavTabClassName(true);
    expect(activeTab).toContain("rounded-full");
    expect(activeTab).toContain("text-[13px]");
    expect(activeTab).toContain("font-medium");
    expect(activeTab).not.toContain("font-bold");
  });

  it("uses xs tab sizing", () => {
    const activeTab = segmentedNavTabClassName(true, { size: "xs" });
    expect(activeTab).toContain("text-xs");
    expect(activeTab).toContain("px-2");
  });
});
