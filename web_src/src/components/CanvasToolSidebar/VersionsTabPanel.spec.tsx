import { fireEvent, render, screen } from "@testing-library/react";
import type { CanvasesCanvasBranch, CanvasesCanvasVersion } from "@/api-client";
import { describe, expect, it, vi } from "vitest";
import { VersionsTabPanel } from "./VersionsTabPanel";

vi.mock("@/hooks/useOrganizationUserAvatars", () => ({
  useOrganizationUserAvatars: () => new Map(),
}));

function makeCommit(id: string, message: string, sha?: string): CanvasesCanvasVersion {
  return {
    metadata: {
      id,
      owner: { name: "Alice" },
      createdAt: "2026-05-18T12:00:00Z",
      commitMessage: message,
      commitSha: sha,
    },
  };
}

const branches: CanvasesCanvasBranch[] = [
  { id: "branch-main", name: "main", headVersionId: "version-2" },
  { id: "branch-feature", name: "feature/login", headVersionId: "version-feature" },
];

describe("VersionsTabPanel", () => {
  it("shows the empty state when there are no commits on the branch", () => {
    render(
      <VersionsTabPanel
        branchCommits={[]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        canvasBranches={branches}
        activeBranchName="main"
        onSelectBranch={vi.fn()}
      />,
    );

    expect(screen.getByText("No commits on this branch yet.")).toBeInTheDocument();
  });

  it("renders branch selector and commits section", () => {
    render(
      <VersionsTabPanel
        branchHeadVersionId="version-2"
        branchCommits={[
          makeCommit("version-2", "Latest change", "abc1234567890"),
          makeCommit("version-1", "Initial commit", "def0987654321"),
        ]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        canvasBranches={branches}
        activeBranchName="main"
        onSelectBranch={vi.fn()}
      />,
    );

    expect(screen.getByTestId("canvas-branch-selector")).toBeInTheDocument();
    expect(screen.getByText("Commits")).toBeInTheDocument();
    expect(screen.getByText("Latest change")).toBeInTheDocument();
    expect(screen.getByText("abc1234")).toBeInTheDocument();
    expect(screen.getAllByTestId("committer-avatar")).toHaveLength(2);
  });

  it("selects a commit when a row is clicked", () => {
    const onUseVersion = vi.fn();

    render(
      <VersionsTabPanel
        branchHeadVersionId="version-2"
        branchCommits={[makeCommit("version-2", "Latest change"), makeCommit("version-1", "Initial commit")]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={onUseVersion}
        canvasBranches={branches}
        activeBranchName="main"
        onSelectBranch={vi.fn()}
      />,
    );

    fireEvent.click(screen.getAllByTestId("canvas-commit-row")[1]);

    expect(onUseVersion).toHaveBeenCalledWith("version-1");
  });

  it("loads older commits when the sidebar scroll reaches the end", () => {
    const onLoadMoreBranchCommits = vi.fn();
    const branchCommits = [makeCommit("version-3", "Third"), makeCommit("version-2", "Second")];
    const props = {
      branchHeadVersionId: "version-3",
      branchCommits,
      canUpdateCanvas: true,
      canvasDeletedRemotely: false,
      onUseVersion: vi.fn(),
      canvasBranches: branches,
      activeBranchName: "main",
      onSelectBranch: vi.fn(),
    };

    const { rerender } = render(<VersionsTabPanel {...props} />);
    const scroller = screen.getByTestId("versions-sidebar-scroll");

    Object.defineProperties(scroller, {
      scrollHeight: { configurable: true, value: 1000 },
      clientHeight: { configurable: true, value: 300 },
      scrollTop: { configurable: true, writable: true, value: 0 },
    });

    rerender(
      <VersionsTabPanel
        {...props}
        onLoadMoreBranchCommits={onLoadMoreBranchCommits}
        loadMoreBranchCommitsDisabled={false}
        loadMoreBranchCommitsPending={false}
      />,
    );

    scroller.scrollTop = 860;
    fireEvent.scroll(scroller);

    expect(onLoadMoreBranchCommits).toHaveBeenCalledTimes(1);
  });
});
