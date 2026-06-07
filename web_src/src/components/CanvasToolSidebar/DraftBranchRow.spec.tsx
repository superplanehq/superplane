import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasVersion } from "@/api-client";
import { DraftBranchRow } from "./DraftBranchRow";

const draft: CanvasesCanvasVersion = {
  metadata: {
    id: "version-1",
    branchName: "drafts/user-1",
    displayName: "Draft #1",
    owner: { id: "user-1", name: "Ada" },
    updatedAt: "2026-06-03T12:00:00.000Z",
  },
};

describe("DraftBranchRow", () => {
  it("renders draft metadata and highlights the active row", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive canUpdateCanvas onOpen={vi.fn()} onDelete={vi.fn()} />,
    );

    expect(screen.getByText("Draft #1")).toBeInTheDocument();
    expect(screen.getByText("drafts/user-1")).toBeInTheDocument();
    expect(screen.getByText(/Ada ·/)).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-blue-50");
    expect(screen.queryByText("Ready to publish")).not.toBeInTheDocument();
  });

  it("opens and deletes the draft branch", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    const onDelete = vi.fn();

    render(<DraftBranchRow draft={draft} isActive={false} canUpdateCanvas onOpen={onOpen} onDelete={onDelete} />);

    await user.click(screen.getByText("Draft #1"));
    await user.click(screen.getByRole("button", { name: "Delete Draft #1" }));

    expect(onOpen).toHaveBeenCalledWith("drafts/user-1");
    expect(onDelete).toHaveBeenCalledWith("version-1");
  });
});
