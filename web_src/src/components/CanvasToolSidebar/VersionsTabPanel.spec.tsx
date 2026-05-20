import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VersionsTabPanel } from "./VersionsTabPanel";

function makeVersion(id: string, number: number) {
  return {
    metadata: {
      id,
      owner: { name: "Alice" },
      createdAt: "2026-05-18T12:00:00Z",
    },
    version: number,
    publishedAt: "2026-05-18T12:00:00Z",
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
        liveVersions={[makeVersion("version-2", 2), makeVersion("version-1", 1)]}
        canUpdateCanvas={true}
        isTemplate={false}
        canvasDeletedRemotely={false}
        onUseVersion={onUseVersion}
        onVersionNodeDiffContextChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByLabelText("Preview Draft version"));

    expect(onUseVersion).toHaveBeenCalledWith("version-2");
  });
});
