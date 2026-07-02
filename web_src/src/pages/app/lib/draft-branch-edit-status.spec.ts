import { describe, expect, it } from "vitest";

import {
  draftBranchStatusBadge,
  draftEditTabToneFromStaging,
  resolveDraftBranchEditStatus,
} from "./draft-branch-edit-status";

describe("draft-branch-edit-status", () => {
  it("maps staging and publishable state to edit status", () => {
    expect(resolveDraftBranchEditStatus(true, true)).toBe("uncommitted");
    expect(resolveDraftBranchEditStatus(true, false)).toBe("uncommitted");
    expect(resolveDraftBranchEditStatus(false, true)).toBe("ready");
    expect(resolveDraftBranchEditStatus(false, false)).toBe("no-changes");
    expect(draftEditTabToneFromStaging(true, true)).toBe("uncommitted");
    expect(draftEditTabToneFromStaging(false, true)).toBe("ready");
    expect(draftEditTabToneFromStaging(false, false)).toBe("neutral");
  });

  it("uses orange badge styling for active uncommitted drafts", () => {
    const badge = draftBranchStatusBadge("uncommitted", true);
    expect(badge.label).toBe("Uncommitted changes");
    expect(badge.className).toContain("bg-orange-100");
    expect(badge.className).toContain("text-orange-800");
  });

  it("uses blue badge styling for active ready drafts", () => {
    const badge = draftBranchStatusBadge("ready", true);
    expect(badge.label).toBe("Ready to publish");
    expect(badge.className).toContain("bg-blue-100");
    expect(badge.className).toContain("text-blue-800");
  });

  it("uses gray badge styling for drafts with no changes", () => {
    const activeBadge = draftBranchStatusBadge("no-changes", true);
    const inactiveBadge = draftBranchStatusBadge("no-changes", false);

    expect(activeBadge.label).toBe("No changes");
    expect(activeBadge.className).toContain("bg-slate-100");
    expect(activeBadge.className).toContain("text-slate-600");
    expect(inactiveBadge.className).toBe(activeBadge.className);
  });
});
