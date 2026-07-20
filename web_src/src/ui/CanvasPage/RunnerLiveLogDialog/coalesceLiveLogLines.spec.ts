import { describe, expect, it } from "vitest";
import { coalesceLiveLogLines } from "./coalesceLiveLogLines";

describe("coalesceLiveLogLines", () => {
  it("rejoins mid-word Claude stream chunks", () => {
    expect(
      coalesceLiveLogLines([
        "The bi",
        "ggest file tracked in the `superplane` repository is:",
        "**`web_src/src/pages/app/__fixtures__/console/docsReviewer.json`** — about **1.5 MB",
        "** (1,508,998 bytes)",
      ]),
    ).toEqual([
      "The biggest file tracked in the `superplane` repository is:**`web_src/src/pages/app/__fixtures__/console/docsReviewer.json`** — about **1.5 MB** (1,508,998 bytes)",
    ]);
  });

  it("keeps tool markers and bash output on separate lines", () => {
    expect(
      coalesceLiveLogLines([
        "Looking around.",
        "-> [Bash] ls -la",
        "     README.md",
        "     src",
      ]),
    ).toEqual(["Looking around.", "-> [Bash] ls -la", "     README.md", "     src"]);
  });

  it("preserves paragraph breaks inside Claude prose", () => {
    expect(coalesceLiveLogLines(["First paragraph.", "", "Sec", "ond paragraph."])).toEqual([
      "First paragraph.\n\nSecond paragraph.",
    ]);
  });

  it("leaves non-Claude command output untouched", () => {
    expect(coalesceLiveLogLines(["file_a", "file_b", "file_c"])).toEqual(["file_a", "file_b", "file_c"]);
  });

  it("hides legacy Claude and tool-result labels", () => {
    expect(
      coalesceLiveLogLines([
        "Claude",
        "Looking around.",
        "-> [Bash] ls -la",
        "← tool result",
        "     README.md",
        "← tool result (empty)",
      ]),
    ).toEqual(["Looking around.", "-> [Bash] ls -la", "     README.md"]);
  });
});
