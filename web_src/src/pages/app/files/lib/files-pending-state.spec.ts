import { describe, expect, it } from "vitest";

import { applyPendingContentUpdate } from "./files-pending-state";

describe("applyPendingContentUpdate", () => {
  it("records the first edit before the committed baseline has loaded", () => {
    const next = applyPendingContentUpdate({}, "README.md", "hello!", undefined);

    expect(next).toEqual({
      "README.md": { type: "modified", path: "README.md", content: "hello!" },
    });
  });

  it("does not create a pending change for empty content without a baseline", () => {
    expect(applyPendingContentUpdate({}, "README.md", "", undefined)).toEqual({});
  });
});
