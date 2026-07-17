import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Textarea } from "./textarea";

describe("Textarea", () => {
  it("applies a dark-mode text color so typed values stay visible on dark backgrounds", () => {
    render(<Textarea data-testid="textarea" />);

    expect(screen.getByTestId("textarea").className).toContain("dark:text-gray-100");
  });

  it("keeps caller-provided classes when applying base theme styles", () => {
    render(<Textarea data-testid="textarea" className="font-mono" />);

    const textarea = screen.getByTestId("textarea");
    expect(textarea.className).toContain("font-mono");
    expect(textarea.className).toContain("dark:text-gray-100");
  });
});
