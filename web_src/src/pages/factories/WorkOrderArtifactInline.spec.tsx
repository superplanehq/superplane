import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

function renderedIcon() {
  const svg = document.querySelector("svg");
  expect(svg).toBeTruthy();
  return svg!;
}

describe("WorkOrderArtifactInline", () => {
  it("renders a branch artifact with a url as a clickable link", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch-with-url",
          type: "TYPE_BRANCH",
          data: {
            name: "feature/refund-retry",
            url: "https://github.com/example/repo/tree/feature/refund-retry",
          },
        }}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://github.com/example/repo/tree/feature/refund-retry");
    expect(link).toHaveTextContent("feature/refund-retry");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("links a branch stored with a repository but no url", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch-from-repository",
          type: "TYPE_BRANCH",
          data: { name: "feature/refund-retry", repository: "example/repo" },
        }}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://github.com/example/repo/tree/feature/refund-retry");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders a branch artifact with neither url nor repository as plain text", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch-without-url",
          type: "TYPE_BRANCH",
          data: { name: "feature/refund-retry" },
        }}
      />,
    );

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("feature/refund-retry")).toBeInTheDocument();
  });

  it("caps a long branch name at 25 letters and keeps the full name in the tooltip", () => {
    const name = "fix/bug-not-getting-notified-for-status-change-when-reopened";
    render(<WorkOrderArtifactInline artifact={{ id: "branch-long", type: "TYPE_BRANCH", data: { name } }} />);

    const label = screen.getByText("fix/bug-not-getting-notif…");
    expect(label).toBeInTheDocument();
    expect(label).toHaveAttribute("title", name);
  });
});

describe("WorkOrderArtifactInline (branch icon)", () => {
  it("renders a branch artifact muted", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{ id: "branch-1", type: "TYPE_BRANCH", data: { name: "feature/refund-retry" } }}
      />,
    );
    const svg = screen.getByText("feature/refund-retry").parentElement?.querySelector("svg");
    expect(svg).toHaveClass("text-muted-foreground");
  });
});

describe("WorkOrderArtifactInline (link artifacts)", () => {
  it("renders a link artifact with a url as a clickable link, titled from data.title", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "link-with-title",
          type: "TYPE_LINK",
          data: {
            url: "https://preview.example.com/pr-42",
            title: "Preview",
          },
        }}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://preview.example.com/pr-42");
    expect(link).toHaveTextContent("Preview");
    expect(renderedIcon()).toHaveClass("text-muted-foreground");
  });

  it("falls back to a compact url label when a link artifact has no title", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "link-without-title",
          type: "TYPE_LINK",
          data: { url: "https://preview.example.com/pr-42" },
        }}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveTextContent("pr-42");
  });

  it("renders a link artifact without a url as plain text, not a link", () => {
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "link-without-url",
          type: "TYPE_LINK",
          data: { title: "Preview" },
        }}
      />,
    );

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("Preview")).toBeInTheDocument();
  });
});
