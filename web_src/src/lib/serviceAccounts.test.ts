import { describe, expect, it } from "vitest";
import { serviceAccountCreatorLabel } from "./serviceAccounts";

describe("serviceAccountCreatorLabel", () => {
  it("returns creator name when present", () => {
    expect(
      serviceAccountCreatorLabel({
        createdBy: "uuid-1",
        createdByUser: { id: "uuid-1", name: "Alice" },
      }),
    ).toBe("Alice");
  });

  it("returns Unknown when creator id exists but name is missing", () => {
    expect(
      serviceAccountCreatorLabel({
        createdBy: "uuid-1",
        createdByUser: { id: "uuid-1" },
      }),
    ).toBe("Unknown");
  });

  it("returns em dash when no creator", () => {
    expect(serviceAccountCreatorLabel({})).toBe("—");
  });
});
