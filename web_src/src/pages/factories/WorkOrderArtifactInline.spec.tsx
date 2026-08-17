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

  it("renders a branch artifact without a url and no siblings as plain text", () => {
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

  it("makes a nameless-url branch clickable when a sibling PR points at the same repo", () => {
    // A branch is normally attached before the PR opens, so `data.url` is
    // usually missing. We derive a repo tree URL from the sibling PR so
    // the chip is clickable without the flow author remembering to pass
    // `url` at attach time.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch",
          type: "TYPE_BRANCH",
          data: { name: "feature/refund-retry" },
        }}
        relatedArtifacts={[{ type: "TYPE_PR", data: { url: "https://github.com/example/repo/pull/42" } }]}
      />,
    );

    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "https://github.com/example/repo/tree/feature/refund-retry");
    expect(link).toHaveTextContent("feature/refund-retry");
  });

  it("does not derive a link from a non-GitHub-style sibling URL", () => {
    // We only synthesize tree URLs for GitHub `/pull/{n}` shapes; GitLab
    // and Bitbucket use different path prefixes and would produce a
    // broken link.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "branch",
          type: "TYPE_BRANCH",
          data: { name: "feature/refund-retry" },
        }}
        relatedArtifacts={[
          {
            type: "TYPE_PR",
            data: { url: "https://gitlab.example.com/example/repo/-/merge_requests/7" },
          },
        ]}
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

  it("renders a GitHub-payload PR (state:closed, merged:true) as merged (purple)", () => {
    // GitHub reports merged PRs as `{ state: "closed", merged: true }`.
    // Without this, every merged PR that never got rewritten to SuperPlane's
    // `state: "merged"` displays as red.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "pr-github-merged",
          type: "TYPE_PR",
          data: {
            url: "https://github.com/example/repo/pull/42",
            state: "closed",
            merged: true,
          },
        }}
      />,
    );
    expect(renderedIcon()).toHaveClass("text-purple-600");
  });

  it("renders a PR with a leftover state:open + merged:true as merged", () => {
    // Some flows attach a PR eagerly with state:"open" and never flip the
    // state field — only the GitHub `merged` field is fresh. The chip
    // should still show purple, not green.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "pr-eager-open",
          type: "TYPE_PR",
          data: {
            url: "https://github.com/example/repo/pull/42",
            state: "open",
            merged: true,
          },
        }}
      />,
    );
    expect(renderedIcon()).toHaveClass("text-purple-600");
  });

  it("renders a GitHub-payload draft PR (state:open, draft:true) as muted draft", () => {
    // GitHub draft PRs are `{ state: "open", draft: true }`; the chip
    // needs the `draft` flag to look muted instead of green.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "pr-github-draft",
          type: "TYPE_PR",
          data: {
            url: "https://github.com/example/repo/pull/42",
            state: "open",
            draft: true,
          },
        }}
      />,
    );
    expect(renderedIcon()).toHaveClass("text-muted-foreground");
  });

  it('accepts a stringified merged flag ("true") from templated inputs', () => {
    // Flow inputs get resolved to strings; supporting "true"/"false" keeps
    // authors from needing a boolean cast in the expression.
    render(
      <WorkOrderArtifactInline
        artifact={{
          id: "pr-string-merged",
          type: "TYPE_PR",
          data: {
            url: "https://github.com/example/repo/pull/42",
            state: "closed",
            merged: "true",
          },
        }}
      />,
    );
    expect(renderedIcon()).toHaveClass("text-purple-600");
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
