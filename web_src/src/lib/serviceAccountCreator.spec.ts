import { describe, expect, it } from "vitest";
import { formatServiceAccountCreatorLabel } from "@/lib/serviceAccountCreator";

describe("formatServiceAccountCreatorLabel", () => {
  it("returns null when name is missing", () => {
    expect(formatServiceAccountCreatorLabel({})).toBeNull();
    expect(formatServiceAccountCreatorLabel({ createdByEmail: "only@email.test" })).toBeNull();
  });

  it("returns only the name when present, ignoring email", () => {
    expect(
      formatServiceAccountCreatorLabel({
        createdByName: "E2E User",
        createdByEmail: "e2e@superplane.local",
      }),
    ).toBe("E2E User");
  });

  it("trims whitespace on the name", () => {
    expect(formatServiceAccountCreatorLabel({ createdByName: "  Alice  " })).toBe("Alice");
  });
});
