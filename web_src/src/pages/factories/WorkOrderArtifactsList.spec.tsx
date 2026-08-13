import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderArtifactsList } from "./WorkOrderArtifactsList";

vi.mock("@/pages/app/Markdown", () => ({
  MarkdownContent: ({ content }: { content: string }) => <div>{content}</div>,
}));

describe("WorkOrderArtifactsList", () => {
  it("left-aligns linked, branch, and markdown artifacts to the same row width", () => {
    render(
      <WorkOrderArtifactsList
        isLoading={false}
        artifacts={[
          {
            id: "pr",
            type: "TYPE_PR",
            data: { title: "#42", url: "https://example.com/pull/42" },
          },
          {
            id: "note",
            type: "TYPE_MARKDOWN",
            data: { title: "Release note", body: "Ready" },
          },
          {
            id: "branch",
            type: "TYPE_BRANCH",
            data: {
              name: "feature/refund-retry",
              url: "https://github.com/example/repo/tree/feature/refund-retry",
            },
          },
        ]}
      />,
    );

    const pullRequest = screen.getByRole("link", { name: "#42" });
    const branch = screen.getByRole("link", { name: "feature/refund-retry" });
    const note = screen.getByRole("button", { name: "Release note" });

    expect(pullRequest).toHaveClass("inline-flex", "w-full", "justify-start");
    expect(branch).toHaveAttribute("href", "https://github.com/example/repo/tree/feature/refund-retry");
    expect(branch).toHaveAttribute("target", "_blank");
    expect(branch).toHaveAttribute("rel", "noopener noreferrer");
    expect(branch).toHaveClass("inline-flex", "w-full", "justify-start");
    expect(note).toHaveClass("inline-flex", "w-full", "justify-start", "p-0");
    expect(note).not.toHaveAttribute("data-slot", "button");

    fireEvent.click(note);
    expect(screen.getByRole("dialog").parentElement).toBe(document.body);
  });

  it("does not link a branch artifact without a URL", () => {
    render(
      <WorkOrderArtifactsList
        isLoading={false}
        artifacts={[
          {
            id: "branch",
            type: "TYPE_BRANCH",
            data: { name: "feature/refund-retry" },
          },
        ]}
      />,
    );

    expect(screen.getByText("feature/refund-retry")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "feature/refund-retry" })).not.toBeInTheDocument();
  });
});
