import { describe, expect, it } from "vitest";
import { toTestId } from "@/lib/testID";

describe("testID", () => {
  it("normalizes labels into lowercase dash-separated ids", () => {
    expect(toTestId("Run Workflow Button")).toBe("run-workflow-button");
  });
});
