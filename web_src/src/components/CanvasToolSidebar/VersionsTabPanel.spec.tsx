import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VersionsTabPanel } from "./VersionsTabPanel";

function makePublishedVersion(id: string, number: number) {
  return {
    metadata: {
      id,
      owner: { name: "Alice" },
      createdAt: "2026-05-18T12:00:00Z",
      state: "STATE_PUBLISHED",
      publishedAt: "2026-05-18T12:00:00Z",
    },
    version: number,
  };
}

describe("VersionsTabPanel", () => {
  it("shows the empty state when there is no published history", () => {
    render(
      <VersionsTabPanel
        liveVersions={[]}
        canUpdateCanvas={true}
        isTemplate={false}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        onVersionNodeDiffContextChange={vi.fn()}
      />,
    );

    expect(screen.getByText("No published history yet.")).toBeInTheDocument();
  });

  it("selects a version when a row is clicked", () => {
    const onUseVersion = vi.fn();

    render(
      <VersionsTabPanel
        liveCanvasVersionId="version-2"
        liveVersions={[makePublishedVersion("version-2", 2), makePublishedVersion("version-1", 1)]}
        canUpdateCanvas={true}
        isTemplate={false}
        canvasDeletedRemotely={false}
        onUseVersion={onUseVersion}
        onVersionNodeDiffContextChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByLabelText("Preview Published version"));

    expect(onUseVersion).toHaveBeenCalledWith("version-2");
  });

  it("keeps expanded versions visible when selecting a different version", () => {
    const liveVersions = Array.from({ length: 12 }, (_, index) => {
      const number = 12 - index;
      return makePublishedVersion(`version-${number}`, number);
    });

    const { rerender } = render(
      <VersionsTabPanel
        liveCanvasVersionId="version-12"
        selectedCanvasVersion={null}
        liveVersions={liveVersions}
        canUpdateCanvas={true}
        isTemplate={false}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        onVersionNodeDiffContextChange={vi.fn()}
      />,
    );

    // Starts collapsed to 5 versions so the first version ("v1") isn't visible.
    expect(screen.queryByText("v1")).not.toBeInTheDocument();

    // Expand twice (5 -> 10 -> 12) to reveal the first version row.
    fireEvent.click(screen.getByRole("button", { name: "Load older versions" }));
    fireEvent.click(screen.getByRole("button", { name: "Load older versions" }));
    expect(screen.getByText("v1")).toBeInTheDocument();

    // After selecting another version, the expanded view should remain.
    rerender(
      <VersionsTabPanel
        liveCanvasVersionId="version-12"
        selectedCanvasVersion={makePublishedVersion("version-9", 9)}
        liveVersions={liveVersions}
        canUpdateCanvas={true}
        isTemplate={false}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
        onVersionNodeDiffContextChange={vi.fn()}
      />,
    );

    // Version rows beyond the initial 5 should still be visible without clicking again.
    expect(screen.getByText("v1")).toBeInTheDocument();
  });
});
