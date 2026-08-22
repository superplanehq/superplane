import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { escapeJsonStringValue } from "./runInspectorJson";
import { JsonPayload } from "./RunInspectorTimelineCard";

describe("escapeJsonStringValue", () => {
  it("keeps JSON string control characters visible", () => {
    expect(escapeJsonStringValue('first line\nsecond\t"quoted"\\slash')).toBe(
      'first line\\nsecond\\t\\"quoted\\"\\\\slash',
    );
  });
});

describe("JsonPayload", () => {
  it("allows long string values to wrap inside the viewer", () => {
    const longValue = "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=".repeat(4);
    const { container } = render(<JsonPayload value={{ content: longValue }} jsonViewStyle={{}} collapsed={false} />);
    const viewer = container.querySelector(".w-json-view-container");

    expect(viewer).toHaveClass("json-viewer-wrap-values");
  });

  it("renders long string values in full instead of truncating them", () => {
    const longValue = "error executing request dial tcp connection blocked by policy rule policy-1234567890-abcdefghij";
    const { container } = render(<JsonPayload value={{ error: longValue }} jsonViewStyle={{}} collapsed={false} />);

    expect(container.textContent).toContain(longValue);
    expect(container.textContent).not.toContain("...");
  });
});
