import { fireEvent, render, screen } from "@testing-library/react";
import type { CanvasesCanvasVersion } from "@/api-client";
import { describe, expect, it, vi } from "vitest";
import { VersionsTabPanel } from "./VersionsTabPanel";

function makePublishedVersion(id: string): CanvasesCanvasVersion {
  return {
    metadata: {
      id,
      owner: { name: "Alice" },
      createdAt: "2026-05-18T12:00:00Z",
      state: "STATE_PUBLISHED",
      publishedAt: "2026-05-18T12:00:00Z",
    },
  };
}

describe("VersionsTabPanel", () => {
  it("shows the empty state when there is no published history", () => {
    render(
      <VersionsTabPanel
        liveVersions={[]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
      />,
    );

    expect(screen.getByText("No published history yet.")).toBeInTheDocument();
  });

  it("calls onCreateDraftBranch when the create-draft button is clicked", () => {
    const onCreateDraftBranch = vi.fn();

    render(
      <VersionsTabPanel
        liveVersions={[]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        onCreateDraftBranch={onCreateDraftBranch}
      />,
    );

    fireEvent.click(screen.getByTestId("canvas-create-draft-button"));

    expect(onCreateDraftBranch).toHaveBeenCalledTimes(1);
  });

  it("hides the create-draft button when the user cannot update the canvas", () => {
    render(
      <VersionsTabPanel
        liveVersions={[]}
        canUpdateCanvas={false}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        onCreateDraftBranch={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("canvas-create-draft-button")).not.toBeInTheDocument();
  });

  it("selects a version when a row is clicked", () => {
    const onUseVersion = vi.fn();

    render(
      <VersionsTabPanel
        liveCanvasVersionId="version-2"
        liveVersions={[makePublishedVersion("version-2"), makePublishedVersion("version-1")]}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={onUseVersion}
      />,
    );

    fireEvent.click(screen.getAllByTestId("canvas-live-version-row")[0]);

    expect(onUseVersion).toHaveBeenCalledWith("version-2");
  });

  it("keeps loaded versions visible when selecting a different version", () => {
    const liveVersions = Array.from({ length: 12 }, (_, index) => {
      const number = 12 - index;
      return makePublishedVersion(`version-${number}`);
    });

    const { rerender } = render(
      <VersionsTabPanel
        liveCanvasVersionId="version-12"
        selectedCanvasVersion={null}
        liveVersions={liveVersions}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
      />,
    );

    expect(screen.getByText("v1")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Load older versions" })).not.toBeInTheDocument();

    rerender(
      <VersionsTabPanel
        liveCanvasVersionId="version-12"
        selectedCanvasVersion={makePublishedVersion("version-9")}
        liveVersions={liveVersions}
        canUpdateCanvas={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
      />,
    );

    expect(screen.getByText("v1")).toBeInTheDocument();
  });

  it("restores the sidebar scroll position after remounting for the same canvas", () => {
    const liveVersions = Array.from({ length: 12 }, (_, index) => {
      const number = 12 - index;
      return makePublishedVersion(`version-${number}`);
    });
    const props = {
      scrollPersistenceKey: "canvas-1",
      liveCanvasVersionId: "version-12",
      liveVersions,
      canUpdateCanvas: true,
      canvasDeletedRemotely: false,
      onUseVersion: vi.fn(),
    };

    const { unmount } = render(<VersionsTabPanel {...props} />);
    const scroller = screen.getByTestId("versions-sidebar-scroll");

    scroller.scrollTop = 420;
    fireEvent.scroll(scroller);
    unmount();

    render(<VersionsTabPanel {...props} selectedCanvasVersion={makePublishedVersion("version-9")} />);

    expect(screen.getByTestId("versions-sidebar-scroll").scrollTop).toBe(420);
  });

  it("loads older versions when the sidebar scroll reaches the end", () => {
    const onLoadMoreLiveVersions = vi.fn();
    const liveVersions = [makePublishedVersion("version-3"), makePublishedVersion("version-2")];
    const props = {
      liveCanvasVersionId: "version-3",
      selectedCanvasVersion: null,
      liveVersions,
      canUpdateCanvas: true,
      canvasDeletedRemotely: false,
      onUseVersion: vi.fn(),
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
        onLoadMoreLiveVersions={onLoadMoreLiveVersions}
        loadMoreLiveVersionsDisabled={false}
        loadMoreLiveVersionsPending={false}
      />,
    );

    expect(onLoadMoreLiveVersions).not.toHaveBeenCalled();

    scroller.scrollTop = 860;
    fireEvent.scroll(scroller);

    expect(onLoadMoreLiveVersions).toHaveBeenCalledTimes(1);
  });
});
