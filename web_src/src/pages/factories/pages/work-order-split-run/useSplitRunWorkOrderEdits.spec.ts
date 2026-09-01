import { describe, expect, it } from "vitest";

import { canEditSplitRunContent, canEditSplitRunDescription } from "./useSplitRunWorkOrderEdits";

describe("canEditSplitRunContent", () => {
  it("allows title edits until the task is done", () => {
    expect(canEditSplitRunContent("draft")).toBe(true);
    expect(canEditSplitRunContent("running")).toBe(true);
    expect(canEditSplitRunContent("waiting")).toBe(true);
    expect(canEditSplitRunContent("failed")).toBe(true);
    expect(canEditSplitRunContent("stopped")).toBe(true);
    expect(canEditSplitRunContent("done")).toBe(false);
    expect(canEditSplitRunContent("draft", false)).toBe(false);
  });
});

describe("canEditSplitRunDescription", () => {
  it("allows description edits only on drafts", () => {
    expect(canEditSplitRunDescription("draft")).toBe(true);
    expect(canEditSplitRunDescription("running")).toBe(false);
    expect(canEditSplitRunDescription("done")).toBe(false);
    expect(canEditSplitRunDescription("draft", false)).toBe(false);
  });
});
