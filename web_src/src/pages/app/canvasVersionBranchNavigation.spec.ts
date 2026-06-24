import { describe, expect, it } from "vitest";
import { applyVersionSelectionSearchParams } from "./canvasVersionBranchNavigation";

describe("applyVersionSelectionSearchParams", () => {
  it("clears versions view when selecting a draft branch", () => {
    const next = applyVersionSelectionSearchParams(new URLSearchParams("view=versions&run=run-1"), {
      isCurrentLive: false,
      versionID: "draft-version",
      branchName: "drafts/abc",
    });

    expect(next.get("view")).toBeNull();
    expect(next.get("version")).toBe("draft-version");
    expect(next.get("branch")).toBe("drafts/abc");
    expect(next.get("run")).toBeNull();
  });

  it("keeps the current view when returning to live", () => {
    const next = applyVersionSelectionSearchParams(new URLSearchParams("view=versions"), {
      isCurrentLive: true,
      versionID: "",
      branchName: "",
    });

    expect(next.get("view")).toBeNull();
  });

  it("clears legacy versions view when previewing a published version", () => {
    const next = applyVersionSelectionSearchParams(new URLSearchParams("view=versions"), {
      isCurrentLive: false,
      versionID: "published-version",
      branchName: "",
    });

    expect(next.get("view")).toBeNull();
    expect(next.get("version")).toBe("published-version");
  });
});
