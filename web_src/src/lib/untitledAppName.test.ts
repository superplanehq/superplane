import { describe, expect, it } from "vitest";
import { generateUntitledAppName } from "./untitledAppName";

describe("generateUntitledAppName", () => {
  it("returns Untitled App 1 when no untitled apps exist", () => {
    expect(generateUntitledAppName([])).toBe("Untitled App 1");
    expect(generateUntitledAppName(["My Workflow", "FanOut"])).toBe("Untitled App 1");
  });

  it("increments the highest existing untitled app number", () => {
    expect(generateUntitledAppName(["Untitled App 1"])).toBe("Untitled App 2");
    expect(generateUntitledAppName(["Untitled App 1", "Untitled App 3", "Other"])).toBe("Untitled App 4");
  });
});
