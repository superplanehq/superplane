import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Textarea } from "./textarea";

describe("Textarea", () => {
  it("applies a dark-mode text color so typed values stay visible on dark backgrounds", () => {
    // Regression test for #6137: the base Textarea hardcoded near-black text
    // but was missing a dark-mode override, making values invisible in dark mode.
    render(<Textarea data-testid="textarea" />);

    const textarea = screen.getByTestId("textarea");
    expect(textarea.className).toContain("dark:text-gray-100");
  });

  it("merges caller-provided classes with the base styles", () => {
    render(<Textarea data-testid="textarea" className="font-mono" />);

    const textarea = screen.getByTestId("textarea");
    expect(textarea.className).toContain("font-mono");
    expect(textarea.className).toContain("dark:text-gray-100");
  });
});
