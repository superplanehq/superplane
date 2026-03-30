import { describe, expect, it } from "vitest";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

describe("integrationDisplayName", () => {
  it("prefers known canonical display names", () => {
    expect(getIntegrationTypeDisplayName(undefined, "github")).toBe("GitHub");
  });

  it("uses the label when it is already properly capitalized", () => {
    expect(getIntegrationTypeDisplayName("Linear", "linear")).toBe("Linear");
  });

  it("capitalizes unknown names when the label is missing", () => {
    expect(getIntegrationTypeDisplayName(undefined, "customtool")).toBe("Customtool");
  });
});
