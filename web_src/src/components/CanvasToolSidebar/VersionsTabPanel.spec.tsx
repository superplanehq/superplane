import type { CanvasesCanvasVersion } from "@/api-client";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VersionsTabPanel } from "./VersionsTabPanel";

function makeVersion(id: string, commitMessage?: string): CanvasesCanvasVersion {
  return {
    metadata: {
      id,
      author: { name: "Alice" },
      createdAt: "2026-05-18T12:00:00Z",
      commitMessage,
    },
  };
}

describe("VersionsTabPanel", () => {
  it("shows the empty state when there is no commit history", () => {
    render(
      <VersionsTabPanel
        liveVersions={[]}
        canEditCanvasVersion={true}
        canvasDeletedRemotely={false}
        onUseVersion={vi.fn()}
      />,
    );

    expect(screen.getByText("No commit history yet.")).toBeInTheDocument();
  });

  it("selects a version when a row is clicked", () => {
    const onUseVersion = vi.fn();

    render(
      <VersionsTabPanel
        liveCanvasVersionId="version-2"
        liveVersions={[makeVersion("version-2", "Latest"), makeVersion("version-1", "Earlier")]}
        canEditCanvasVersion={true}
        canvasDeletedRemotely={false}
        onUseVersion={onUseVersion}
      />,
    );

    fireEvent.click(screen.getAllByTestId("canvas-live-version-row")[0]);

    expect(onUseVersion).toHaveBeenCalledWith("version-2");
  });
});
