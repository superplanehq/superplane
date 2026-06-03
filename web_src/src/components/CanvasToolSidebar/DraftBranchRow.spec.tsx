import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DraftBranchRow } from "./DraftBranchRow";

const draft = {
  branchName: "drafts/user-1",
  displayName: "Draft #1",
  tipSha: "abc1234567890123456789012345678901234567890",
  owner: { id: "user-1", name: "Ada" },
  updatedAt: "2026-06-03T12:00:00.000Z",
  materializationStatus: "ready",
};

describe("DraftBranchRow", () => {
  it("shows ready to publish badge and blue background when active and ready", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive editStatus="ready" canUpdateCanvas onOpen={vi.fn()} />,
    );

    expect(screen.getByText("Ready to publish")).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-blue-50");
  });

  it("shows uncommitted changes badge and orange background when active and uncommitted", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive editStatus="uncommitted" canUpdateCanvas onOpen={vi.fn()} />,
    );

    expect(screen.getByText("Uncommitted changes")).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-orange-50");
  });

  it("shows gray badge and background when inactive with edit status", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive={false} editStatus="ready" canUpdateCanvas onOpen={vi.fn()} />,
    );

    const badge = screen.getByText("Ready to publish");
    expect(badge).toHaveClass("bg-slate-100");
    expect(badge).toHaveClass("text-slate-600");
    expect(container.firstChild).toHaveClass("bg-slate-50");
  });

  it("shows no changes badge and gray background when active with no changes", () => {
    const { container } = render(
      <DraftBranchRow draft={draft} isActive editStatus="no-changes" canUpdateCanvas onOpen={vi.fn()} />,
    );

    expect(screen.getByText("No changes")).toBeInTheDocument();
    expect(container.firstChild).toHaveClass("bg-blue-50");
    expect(screen.getByText("No changes")).toHaveClass("bg-blue-100");
    expect(screen.getByText("No changes")).toHaveClass("text-blue-800");
  });
});
