import { describe, expect, it } from "vitest";
import { formatRunnerFleetLabel } from "./RunnerFleetFieldRenderer";

describe("formatRunnerFleetLabel", () => {
  it("shows product runner classes instead of provider instance sizes", () => {
    expect(
      formatRunnerFleetLabel({
        id: "aws-standard-amd64",
        provisioner: "aws",
        arch: "amd64",
        size: "t3.micro",
      }),
    ).toBe("Standard runner · amd64");
  });

  it("uses regulated size names from fleet identifiers", () => {
    expect(formatRunnerFleetLabel({ id: "aws-small-amd64", arch: "amd64" })).toBe("Small runner · amd64");
    expect(formatRunnerFleetLabel({ id: "aws-large-arm64", arch: "arm64" })).toBe("Large runner · arm64");
    expect(formatRunnerFleetLabel({ id: "aws-xlarge-amd64", arch: "amd64" })).toBe("Extra large runner · amd64");
  });

  it("defaults to standard without exposing unknown provider metadata", () => {
    expect(formatRunnerFleetLabel({ id: "local", size: "t3.micro" })).toBe("Standard runner");
  });
});
