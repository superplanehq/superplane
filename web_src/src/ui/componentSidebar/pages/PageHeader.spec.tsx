import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PageHeader } from "./PageHeader";

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) => classes.filter(Boolean).join(" "),
}));

describe("PageHeader", () => {
  it("renders a compact back row for bottom layout", () => {
    const onBackToOverview = vi.fn();

    render(<PageHeader onBackToOverview={onBackToOverview} compact />);

    const backButton = screen.getByTestId("compact-page-header-back");
    expect(backButton).toHaveTextContent("Back");
    expect(backButton.className).toContain("h-9");

    fireEvent.click(backButton);
    expect(onBackToOverview).toHaveBeenCalledTimes(1);
  });

  it("renders the legacy back header in sidebar layout", () => {
    render(<PageHeader onBackToOverview={vi.fn()} />);

    expect(screen.queryByTestId("compact-page-header-back")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Back" })).toBeInTheDocument();
  });
});
