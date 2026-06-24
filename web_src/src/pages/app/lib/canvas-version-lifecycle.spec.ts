import { describe, expect, it } from "vitest";

import { shouldReactToCanvasVersionUpdated } from "./canvas-version-lifecycle";

describe("shouldReactToCanvasVersionUpdated", () => {
  it("ignores remote updates on a passive live-view tab", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-1",
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(false);
  });

  it("ignores version-less updates when the versions UI is closed", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: undefined,
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(false);
  });

  it("reacts when editing the affected draft", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-1",
        activeCanvasVersionId: "draft-1",
        isEditing: true,
        editSessionActive: true,
      }),
    ).toBe(true);
  });

  it("reacts when previewing the affected version", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "version-1",
        activeCanvasVersionId: "version-1",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(true);
  });

  it("reacts when the versions sidebar is open even for another draft", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-2",
        activeCanvasVersionId: "draft-1",
        isEditing: true,
        editSessionActive: true,
      }),
    ).toBe(true);
  });

  it("reacts to version-less updates while the versions UI is open", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: undefined,
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: true,
      }),
    ).toBe(true);
  });
});
