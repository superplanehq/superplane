import { describe, expect, it } from "vitest";
import { escapeJsonStringValue } from "./runInspectorJson";

describe("escapeJsonStringValue", () => {
  it("keeps JSON string control characters visible", () => {
    expect(escapeJsonStringValue('first line\nsecond\t"quoted"\\slash')).toBe(
      'first line\\nsecond\\t\\"quoted\\"\\\\slash',
    );
  });
});
