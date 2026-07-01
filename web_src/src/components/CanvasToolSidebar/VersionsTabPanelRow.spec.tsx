import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VersionRow } from "./VersionsTabPanelRow";

function makeCommit(id: string, message: string, ownerName: string) {
  return {
    metadata: {
      id,
      owner: { id: "user-1", name: ownerName },
      createdAt: "2026-05-18T12:00:00Z",
      commitMessage: message,
      commitSha: "abc1234567890",
    },
  };
}

describe("VersionRow", () => {
  it("shows the committer avatar with initials", () => {
    render(
      <VersionRow
        version={makeCommit("version-1", "Initial commit", "Alice Lovelace")}
        committer={{ name: "Alice Lovelace", initials: "AL" }}
        onUseVersion={vi.fn()}
        rowTestId="canvas-commit-row"
      />,
    );

    expect(screen.getByTestId("committer-avatar")).toBeInTheDocument();
    expect(screen.getByText("AL")).toBeInTheDocument();
    expect(screen.getByText("Initial commit")).toBeInTheDocument();
    expect(screen.getByText("abc1234")).toBeInTheDocument();
  });
});
