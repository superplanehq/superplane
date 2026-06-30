import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { CanvasesCanvasBranch } from "@/api-client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CanvasBranchSelector } from "./CanvasBranchSelector";

const branches: CanvasesCanvasBranch[] = [
  { id: "branch-main", name: "main", headVersionId: "version-main" },
  { id: "branch-feature", name: "feature/login", headVersionId: "version-feature" },
];

describe("CanvasBranchSelector", () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView ??= () => {};
  });

  it("renders the active branch like the project switcher trigger", () => {
    render(<CanvasBranchSelector branches={branches} value="main" onValueChange={vi.fn()} />);

    expect(screen.getByTestId("canvas-branch-selector")).toHaveTextContent("main");
  });

  it("selects a different branch from the searchable menu", async () => {
    const onValueChange = vi.fn();

    render(<CanvasBranchSelector branches={branches} value="main" onValueChange={onValueChange} />);

    fireEvent.click(screen.getByTestId("canvas-branch-selector"));

    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search branches")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("feature/login"));

    expect(onValueChange).toHaveBeenCalledWith("feature/login");
  });
});
