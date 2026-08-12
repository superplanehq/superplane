import { describe, expect, it } from "vitest";
import { shouldShowFactoryNodeStatusFooter } from "./shouldShowFactoryNodeStatusFooter";

describe("shouldShowFactoryNodeStatusFooter", () => {
  it("hides footer in edit mode", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "edit" })).toBe(false);
  });

  it("shows footer on live canvases", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "live" })).toBe(true);
    expect(shouldShowFactoryNodeStatusFooter({})).toBe(true);
  });

  it("hides footer in compact view even when live", () => {
    expect(shouldShowFactoryNodeStatusFooter({ canvasMode: "live", isCompactView: true })).toBe(false);
  });
});
