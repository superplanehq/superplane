import { describe, expect, it } from "vitest";
import { formatServiceAccountCreatorLabel } from "@/lib/serviceAccountCreator";

describe("formatServiceAccountCreatorLabel", () => {
  it("returns null when neither name nor email is present", () => {
    expect(formatServiceAccountCreatorLabel({})).toBeNull();
  });

  it("combines name and email when both are present", () => {
    expect(
      formatServiceAccountCreatorLabel({
        createdByName: "E2E User",
        createdByEmail: "e2e@superplane.local",
      }),
    ).toBe("E2E User (e2e@superplane.local)");
  });

  it("returns only name when email is missing", () => {
    expect(formatServiceAccountCreatorLabel({ createdByName: "Only Name" })).toBe("Only Name");
  });

  it("returns only email when name is missing", () => {
    expect(formatServiceAccountCreatorLabel({ createdByEmail: "only@email.test" })).toBe("only@email.test");
  });

  it("trims whitespace", () => {
    expect(
      formatServiceAccountCreatorLabel({
        createdByName: "  Alice  ",
        createdByEmail: "  a@b.co  ",
      }),
    ).toBe("Alice (a@b.co)");
  });
});
