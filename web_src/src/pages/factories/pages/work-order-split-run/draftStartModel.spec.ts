import { describe, expect, it } from "vitest";

import { DRAFT_START_MODEL_AUTO, draftStartModelPayload } from "./draftStartModel";

describe("draftStartModelPayload", () => {
  it("sends no model for Auto", () => {
    expect(draftStartModelPayload(DRAFT_START_MODEL_AUTO)).toBeUndefined();
    expect(draftStartModelPayload("")).toBeUndefined();
    expect(draftStartModelPayload("  ")).toBeUndefined();
  });

  it("sends the listed id", () => {
    expect(draftStartModelPayload("claude-opus-4-6")).toBe("claude-opus-4-6");
  });
});
