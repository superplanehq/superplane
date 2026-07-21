import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AnnotationComponent } from "./index";

// NodeResizeControl needs a ReactFlow provider we don't set up here; stub it out.
vi.mock("@xyflow/react", () => ({
  NodeResizeControl: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
}));

describe("AnnotationComponent duplicate action", () => {
  it("renders a duplicate button in edit mode and calls onDuplicate on click", () => {
    const onDuplicate = vi.fn();

    render(<AnnotationComponent title="Note" canvasMode="edit" onDuplicate={onDuplicate} />);

    const button = screen.getByTestId("node-action-duplicate");
    expect(button).toHaveAttribute("aria-label", "Duplicate note");

    fireEvent.click(button);
    expect(onDuplicate).toHaveBeenCalledTimes(1);
  });

  it("does not render the duplicate button in live mode", () => {
    render(<AnnotationComponent title="Note" canvasMode="live" onDuplicate={vi.fn()} />);

    expect(screen.queryByTestId("node-action-duplicate")).not.toBeInTheDocument();
  });

  it("does not render the duplicate button when onDuplicate is not provided", () => {
    render(<AnnotationComponent title="Note" canvasMode="edit" />);

    expect(screen.queryByTestId("node-action-duplicate")).not.toBeInTheDocument();
  });

  it("hides the duplicate button when actions are hidden", () => {
    render(<AnnotationComponent title="Note" canvasMode="edit" hideActionsButton onDuplicate={vi.fn()} />);

    expect(screen.queryByTestId("node-action-duplicate")).not.toBeInTheDocument();
  });
});
