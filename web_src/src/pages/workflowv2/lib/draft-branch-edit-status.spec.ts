import { describe, expect, it } from "vitest";
import { aggregateDraftTabIndicators } from "./draft-branch-edit-status";

describe("aggregateDraftTabIndicators", () => {
  it("shows uncommitted canvas when any draft has uncommitted canvas, suppressing ready dots", () => {
    const indicators = aggregateDraftTabIndicators({
      "drafts/a": {
        editStatus: "uncommitted",
        hasUncommittedCanvas: true,
        hasUncommittedConsole: false,
        hasUncommittedFiles: false,
        hasCommittedCanvasVersusLive: false,
        hasCommittedConsoleVersusLive: false,
      },
      "drafts/b": {
        editStatus: "ready",
        hasUncommittedCanvas: false,
        hasUncommittedConsole: false,
        hasUncommittedFiles: false,
        hasCommittedCanvasVersusLive: true,
        hasCommittedConsoleVersusLive: false,
      },
    });

    expect(indicators.hasUncommittedCanvasDraftChanges).toBe(true);
    expect(indicators.readyToPublishCanvasDraftChanges).toBe(false);
    expect(indicators.readyToPublishConsoleDraftChanges).toBe(false);
  });
});
