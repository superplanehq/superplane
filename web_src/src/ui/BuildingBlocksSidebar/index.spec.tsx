import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BuildingBlocksSidebar } from "./index";

const defaultProps = {
  isOpen: true,
  onToggle: vi.fn(),
  blocks: [],
  canvasZoom: 1,
};

describe("BuildingBlocksSidebar", () => {
  it("calls onToggle(false) when close button is clicked while disabled", () => {
    const onToggle = vi.fn();
    render(
      <BuildingBlocksSidebar
        {...defaultProps}
        onToggle={onToggle}
        disabled={true}
        disabledMessage="You don't have permission to edit this canvas."
      />,
    );

    const closeButton = screen.getByTestId("close-sidebar-button");
    fireEvent.click(closeButton);

    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("calls onToggle(false) when close button is clicked while not disabled", () => {
    const onToggle = vi.fn();
    render(<BuildingBlocksSidebar {...defaultProps} onToggle={onToggle} disabled={false} />);

    const closeButton = screen.getByTestId("close-sidebar-button");
    fireEvent.click(closeButton);

    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("renders the disabled overlay when disabled", () => {
    const { container } = render(
      <BuildingBlocksSidebar
        {...defaultProps}
        disabled={true}
        disabledMessage="You don't have permission to edit this canvas."
      />,
    );

    expect(container.querySelector(".cursor-not-allowed")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    const { container } = render(<BuildingBlocksSidebar {...defaultProps} isOpen={false} />);

    expect(container.querySelector('[data-testid="building-blocks-sidebar"]')).not.toBeInTheDocument();
  });
});
