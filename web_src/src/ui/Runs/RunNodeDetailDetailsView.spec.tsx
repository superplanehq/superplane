import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";

describe("RunNodeDetailDetailsView", () => {
  it("collapses long summary values and expands them on demand", () => {
    const longValue = `Hello, World!\n\n${"long output ".repeat(15)}end`;

    render(<RunNodeDetailDetailsView details={{ message: longValue }} />);

    expect(screen.getByText(/^Hello, World!/)).toHaveTextContent("...");
    expect(screen.queryByText(/ end$/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Expand" }));

    expect(screen.getByText(/^Hello, World!/)).toHaveTextContent("end");
    expect(screen.getByRole("button", { name: "Collapse" })).toBeInTheDocument();
  });

  it("does not show expand controls for short summary values", () => {
    render(<RunNodeDetailDetailsView details={{ Host: "root@192.241.150.61" }} />);

    expect(screen.getByText("root@192.241.150.61")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Expand" })).not.toBeInTheDocument();
  });

  it("opens app run links in the same tab", () => {
    render(
      <MemoryRouter>
        <RunNodeDetailDetailsView details={{ Run: "/org-1/apps/child-app?run=child-run" }} />
      </MemoryRouter>,
    );

    const link = screen.getByRole("link", { name: "See run" });
    expect(link).toHaveAttribute("href", "/org-1/apps/child-app?run=child-run");
    expect(link).not.toHaveAttribute("target", "_blank");
  });

  it("still opens external URLs in a new tab", () => {
    render(
      <MemoryRouter>
        <RunNodeDetailDetailsView details={{ URL: "https://example.com/run" }} />
      </MemoryRouter>,
    );

    const link = screen.getByRole("link", { name: "https://example.com/run" });
    expect(link).toHaveAttribute("target", "_blank");
  });
});
