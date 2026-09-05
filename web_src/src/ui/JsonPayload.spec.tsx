import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { escapeJsonStringValue } from "@/lib/jsonViewTheme";

import { JsonPayload } from "./JsonPayload";

describe("escapeJsonStringValue", () => {
  it("keeps JSON string control characters visible", () => {
    expect(escapeJsonStringValue('first line\nsecond\t"quoted"\\slash')).toBe(
      'first line\\nsecond\\t\\"quoted\\"\\\\slash',
    );
  });
});

describe("JsonPayload", () => {
  it("renders a long string value in full, with no ellipsis", () => {
    // A realistic failure payload: the error is one long string. The JSON
    // viewer truncates strings after 30 chars by default, which hides the
    // part of the message the user actually needs.
    const longError =
      "error executing request: dial tcp 10.0.0.1:443 connect: connection blocked by policy rule policy-1234567890-abcdefghij";

    const { container } = render(<JsonPayload value={{ error: longError }} />, { wrapper: ThemeProvider });

    expect(container.textContent).toContain(longError);
    expect(container.textContent).not.toContain("...");
  });

  it("marks string values wrappable so long lines don't overflow the panel", () => {
    const { container } = render(<JsonPayload value={{ content: "x".repeat(120) }} />, { wrapper: ThemeProvider });

    expect(container.querySelector(".w-json-view-container")).toHaveClass("json-viewer-wrap-values");
  });
});
