import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Textarea } from "./textarea";

describe("Textarea", () => {
  it("uses a readable text color in dark mode", () => {
    render(<Textarea aria-label="Secret value" />);

    expect(screen.getByRole("textbox", { name: "Secret value" })).toHaveClass("dark:text-gray-100");
  });
});
