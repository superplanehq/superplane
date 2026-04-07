import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { CanvasVersionNodeDiffDialog, type CanvasVersionNodeDiffContext } from "./CanvasVersionNodeDiffDialog";

function makeVersion(id: string): CanvasesCanvasVersion {
  return { metadata: { id }, spec: { nodes: [] } };
}

function makeContext(changeRequest?: CanvasesCanvasChangeRequest): CanvasVersionNodeDiffContext {
  return {
    version: makeVersion("v2"),
    previousVersion: makeVersion("v1"),
    changeRequest,
  };
}

const noop = () => {};

describe("CanvasVersionNodeDiffDialog", () => {
  it("renders the change request description as markdown", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-1",
        title: "My Change",
        description: "Hello **bold** and *italic*",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    expect(screen.getByText("bold")).toBeInTheDocument();
    const boldEl = screen.getByText("bold");
    expect(boldEl.tagName).toBe("STRONG");

    const italicEl = screen.getByText("italic");
    expect(italicEl.tagName).toBe("EM");
  });

  it("renders markdown links in the description", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-2",
        title: "Link test",
        description: "See [docs](https://example.com) for details",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    const link = screen.getByText("docs");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "https://example.com");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders markdown lists in the description", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-3",
        title: "List test",
        description: "Changes:\n- item one\n- item two\n- item three",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    expect(screen.getByText("item one")).toBeInTheDocument();
    expect(screen.getByText("item two")).toBeInTheDocument();
    expect(screen.getByText("item three")).toBeInTheDocument();
  });

  it("does not render the description section when description is empty", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-4",
        title: "No description",
        description: "",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    const { container } = render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    const descriptionBox = container.querySelector(".bg-slate-50.rounded-md");
    expect(descriptionBox).not.toBeInTheDocument();
  });

  it("does not render the description section when description is undefined", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-5",
        title: "Undefined description",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    const { container } = render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    const descriptionBox = container.querySelector(".bg-slate-50.rounded-md");
    expect(descriptionBox).not.toBeInTheDocument();
  });

  it("does not render the description section when description is whitespace only", () => {
    const cr: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-6",
        title: "Whitespace description",
        description: "   \n  ",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    const { container } = render(<CanvasVersionNodeDiffDialog context={makeContext(cr)} onOpenChange={noop} />);

    const descriptionBox = container.querySelector(".bg-slate-50.rounded-md");
    expect(descriptionBox).not.toBeInTheDocument();
  });

  it("uses liveChangeRequest description over context change request", () => {
    const contextCR: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-7",
        title: "Old title",
        description: "old description",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    const liveCR: CanvasesCanvasChangeRequest = {
      metadata: {
        id: "cr-7",
        title: "New title",
        description: "**live** description",
        status: "STATUS_OPEN",
        createdAt: "2026-04-01T00:00:00Z",
      },
    };

    render(
      <CanvasVersionNodeDiffDialog context={makeContext(contextCR)} onOpenChange={noop} liveChangeRequest={liveCR} />,
    );

    const liveEl = screen.getByText("live");
    expect(liveEl.tagName).toBe("STRONG");
    expect(screen.queryByText("old description")).not.toBeInTheDocument();
  });
});
