import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

function prArtifact(state?: string) {
  return {
    id: "pr-1",
    type: "TYPE_PR",
    data: {
      url: "https://github.com/example/repo/pull/42",
      title: "Draft implementation",
      ...(state ? { state } : {}),
    },
  };
}

function renderedIcon() {
  const link = screen.getByRole("link");
  const svg = link.querySelector("svg");
  if (!svg) throw new Error("expected an icon <svg> inside the artifact link");
  return svg;
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
  });

  it("renders a branch artifact without a url as plain text, not a link", () => {
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
});

describe("WorkOrderArtifactInline (PR state icons)", () => {
  it("renders the open state in emerald", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact("open")} />);
    expect(renderedIcon()).toHaveClass("text-emerald-600");
  });

  it("renders the draft state muted (same look as the pre-state default)", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact("draft")} />);
    expect(renderedIcon()).toHaveClass("text-muted-foreground");
  });

  it("renders the closed state in red", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact("closed")} />);
    expect(renderedIcon()).toHaveClass("text-red-600");
  });

  it("renders the merged state in purple", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact("merged")} />);
    expect(renderedIcon()).toHaveClass("text-purple-600");
  });

  it("falls back to the open look when state is absent (back-compat)", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact()} />);
    expect(renderedIcon()).toHaveClass("text-emerald-600");
  });

  it("falls back to the open look when state is unrecognized", () => {
    render(<WorkOrderArtifactInline artifact={prArtifact("in_review")} />);
    expect(renderedIcon()).toHaveClass("text-emerald-600");
  });

  it("leaves non-PR artifacts muted, unaffected by the PR state map", () => {
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
