import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BackdropContent } from "./BackdropContent";

describe("BackdropContent", () => {
  it("renders mention highlights without changing text metrics", () => {
    render(<BackdropContent text="@Create Release deploy" mentions={[{ label: "Create Release", startIndex: 0 }]} />);

    const mention = screen.getByText("@Create Release");
    expect(mention).toHaveClass("bg-blue-100", "text-blue-700");
    expect(mention.className).not.toMatch(/\bp[trblxy]?-/);
    expect(mention.className).not.toMatch(/\bfont-(?:medium|semibold|bold|extrabold|black)\b/);
  });
});
