import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CategorySection } from "./CategorySection";
import type { BuildingBlockCategory } from "./types";

function createCategory(name: string): BuildingBlockCategory {
  return {
    name,
    blocks: [
      {
        name: "smtp.send",
        label: "Send Email",
        type: "component",
        integrationName: "smtp",
      },
    ],
  };
}

describe("CategorySection", () => {
  it("keeps a non-Core category collapsed by default", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} />);

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(container.querySelector("details")?.open).toBe(false);
  });

  it("expands the Core category by default", () => {
    const category = createCategory("Core");

    const { container } = render(<CategorySection category={category} />);

    expect(screen.getByText("Core")).toBeInTheDocument();
    expect(screen.getByText("Send Email")).toBeInTheDocument();
    expect(container.querySelector("details")?.open).toBe(true);
  });

  it("expands a non-Core category when a search term is present", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} searchTerm="send" />);

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Send Email")).toBeInTheDocument();
    expect(container.querySelector("details")?.open).toBe(true);
  });

  it("expands a non-Core category after it is manually opened", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} />);

    const details = container.querySelector("details");
    expect(details).toBeInTheDocument();
    expect(details?.open).toBe(false);

    details!.open = true;
    fireEvent(details!, new Event("toggle"));

    expect(details?.open).toBe(true);
    expect(screen.getByText("Send Email")).toBeInTheDocument();
  });
});
