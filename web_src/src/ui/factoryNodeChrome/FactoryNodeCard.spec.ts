import { describe, expect, it } from "vitest";
import { shouldShowFactoryNodeStatusFooter } from "./shouldShowFactoryNodeStatusFooter";

describe("shouldShowFactoryNodeStatusFooter", () => {
  it("shows footer in edit mode for No run label", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "edit" })).toBe(true);
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "edit", showRuntimeStatus: false })).toBe(true);
  });

  it("shows footer on live canvases", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "live" })).toBe(true);
    expect(shouldShowFactoryNodeStatusFooter({})).toBe(true);
  });

  it("hides footer in compact view even when live", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "live", isCompactView: true })).toBe(false);
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "edit", isCompactView: true })).toBe(false);
  });

  it("hides footer when runtime status is suppressed on live", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "live", showRuntimeStatus: false })).toBe(false);
  });
});
