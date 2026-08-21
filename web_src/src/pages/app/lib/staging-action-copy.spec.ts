import { describe, expect, it } from "vitest";
import { stagingCommitSuccessToast, stagingResetSuccessToast } from "./staging-action-copy";

describe("staging-action-copy", () => {
  it("uses Save/Discard wording in factory context", () => {
    expect(stagingCommitSuccessToast(true)).toBe("Changes saved");
    expect(stagingResetSuccessToast(true)).toBe("Changes discarded");
  });

  it("keeps commit wording for org canvases", () => {
    expect(stagingCommitSuccessToast(false)).toBe("Changes committed");
    expect(stagingResetSuccessToast(false)).toBe("Reverted to last commit");
  });
});
