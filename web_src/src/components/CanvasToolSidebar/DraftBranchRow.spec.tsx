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
  it("shows ready to publish badge and blue background when active and ready", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive editStatus="ready" canUpdateCanvas onOpen={vi.fn()} onDelete={vi.fn()} />,
    );

    expect(screen.getByText("Ready to publish")).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-blue-50");
  });

  it("shows uncommitted changes badge and orange background when active and uncommitted", () => {
    const { container } = render(
      <DraftBranchRow
        draft={draft}
        isActive
        editStatus="uncommitted"
        canUpdateCanvas
        onOpen={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText("Uncommitted changes")).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-orange-50");
  });

  it("shows gray badge and background when inactive with edit status", () => {
    const { container } = render(
      <DraftBranchRow
        draft={draft}
        isActive={false}
        editStatus="ready"
        canUpdateCanvas
        onOpen={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    const badge = screen.getByText("Ready to publish");
    expect(badge).toHaveClass("bg-slate-100");
    expect(badge).toHaveClass("text-slate-600");
    expect(container.firstChild).toHaveClass("bg-slate-50");
  });

  it("shows gray no changes badge when the draft matches live", () => {
    const { container } = render(
      <DraftBranchRow
        draft={draft}
        isActive
        editStatus="no-changes"
        canUpdateCanvas
        onOpen={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByText("No changes")).toBeInTheDocument();
    expect(screen.getByText("No changes")).toHaveClass("bg-slate-100");
    expect(container.firstChild).toHaveClass("bg-blue-50");
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
