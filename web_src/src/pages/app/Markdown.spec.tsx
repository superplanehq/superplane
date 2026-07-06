import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MarkdownContent } from "./Markdown";

vi.mock("@/components/AgentSidebar/widgets/MermaidWidget", () => ({
  MermaidWidget: ({ content }: { content: string }) => <div data-testid="mermaid-diagram">{content}</div>,
}));

vi.mock("@/components/AgentSidebar/widgets/NodeChip", () => ({
  NodeChipFromLink: ({
    nodeId,
    rawLabel,
    canvasId,
    organizationId,
  }: {
    nodeId: string;
    rawLabel?: string;
    canvasId: string;
    organizationId: string;
  }) => (
    <button type="button" data-testid="node-chip">
      {rawLabel}:{nodeId}:{canvasId}:{organizationId}
    </button>
  ),
}));

vi.mock("@/components/AgentSidebar/widgets/MarkdownCode", () => ({
  MarkdownCode: ({ children, className }: { children?: string; className?: string }) => (
    <div data-language={className?.replace("language-", "")} data-testid="markdown-code">
      {children}
    </div>
  ),
}));

describe("MarkdownContent", () => {
  it("renders mermaid code fences as diagrams", () => {
    render(<MarkdownContent content={"```mermaid\ngraph TD\n  A-->B\n```"} />);

    expect(screen.getByTestId("mermaid-diagram").textContent).toContain("graph TD\n  A-->B");
    expect(screen.getByTestId("mermaid-diagram").closest("pre")).not.toBeInTheDocument();
    expect(screen.queryByTestId("markdown-code")).not.toBeInTheDocument();
  });

  it("renders node links as chips when canvas context is available", () => {
    render(
      <MarkdownContent
        content={"Open [Deploy](node:deploy-node) before continuing."}
        canvasId="canvas-1"
        organizationId="org-1"
      />,
    );

    expect(screen.getByTestId("node-chip")).toHaveTextContent("Deploy:deploy-node:canvas-1:org-1");
  });

  it("keeps regular markdown links on native anchors", () => {
    render(<MarkdownContent content={'Open [docs](../docs "Local docs").'} />);

    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("href", "../docs");
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("title", "Local docs");
    expect(screen.getByRole("link", { name: "docs" })).not.toHaveAttribute("target");
  });

  it("keeps regular fenced code blocks as code", () => {
    render(<MarkdownContent content={"```yaml\nname: deploy\n```"} />);

    expect(screen.getByTestId("markdown-code")).toHaveTextContent("name: deploy");
    expect(screen.getByTestId("markdown-code")).toHaveAttribute("data-language", "yaml");
    expect(screen.getByTestId("markdown-code").closest("pre")).not.toBeInTheDocument();
  });

  it("keeps unlabeled fenced code blocks wrapped as blocks", () => {
    render(<MarkdownContent content={"```\nraw output\n```"} />);

    expect(screen.getByTestId("markdown-code")).toHaveTextContent("raw output");
    expect(screen.getByTestId("markdown-code").closest("pre")).toBeInTheDocument();
  });
});
