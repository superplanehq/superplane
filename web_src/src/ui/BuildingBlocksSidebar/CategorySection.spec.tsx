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
  it("does not render the ItemGroup for a non-Core category that is collapsed by default", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} />);

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(container.querySelector('[data-slot="item-group"]')).not.toBeInTheDocument();
  });

  it("renders the ItemGroup for the Core category, which is expanded by default", () => {
    const category = createCategory("Core");

    const { container } = render(<CategorySection category={category} />);

    expect(screen.getByText("Core")).toBeInTheDocument();
    expect(screen.getByText("Send Email")).toBeInTheDocument();
    expect(container.querySelector('[data-slot="item-group"]')).toBeInTheDocument();
  });

  it("renders the ItemGroup for a non-Core category when a search term is present", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} searchTerm="send" />);

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Send Email")).toBeInTheDocument();
    expect(container.querySelector('[data-slot="item-group"]')).toBeInTheDocument();
  });

  it("renders the ItemGroup for a non-Core category after it is manually opened", () => {
    const category = createCategory("Email");

    const { container } = render(<CategorySection category={category} />);

    const details = container.querySelector("details");
    expect(details).toBeInTheDocument();
    expect(container.querySelector('[data-slot="item-group"]')).not.toBeInTheDocument();

    details!.open = true;
    fireEvent(details!, new Event("toggle"));

    expect(screen.getByText("Send Email")).toBeInTheDocument();
    expect(container.querySelector('[data-slot="item-group"]')).toBeInTheDocument();
  });
});
