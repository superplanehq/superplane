import { describe, expect, it } from "vitest";

import { shouldReadStagedCanvasVersion } from "./live-edit-session";

describe("shouldReadStagedCanvasVersion", () => {
  it("reads staged content only while editing the live version", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: true,
        activeCanvasVersionId: "live-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(true);
  });

  it("reads committed content outside edit mode", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: false,
        activeCanvasVersionId: "live-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(false);
  });

  it("reads committed content when previewing a historical version in edit mode", () => {
    expect(
      shouldReadStagedCanvasVersion({
        editSessionActive: true,
        activeCanvasVersionId: "old-version",
        effectiveLiveCanvasVersionId: "live-version",
        liveCanvasVersionId: "live-version",
      }),
    ).toBe(false);
  });
});
