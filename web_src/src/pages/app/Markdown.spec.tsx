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

vi.mock("@/components/AgentSidebar/widgets/IntegrationButton", () => ({
  IntegrationButton: ({ integrationRef, label }: { integrationRef: string; label?: string }) => (
    <button type="button" data-testid="integration-chip">
      {label}:{integrationRef}
    </button>
  ),
}));

vi.mock("@/components/AgentSidebar/widgets/MarkdownCode", () => ({
  MarkdownCode: ({ children, className }: { children?: string; className?: string }) => (
    <code data-language={className?.replace("language-", "")} data-testid="markdown-code">
      {children}
    </code>
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

  it("renders integration links as chips", () => {
    render(<MarkdownContent content={"Connect [GitHub](integration:github) to continue."} />);

    expect(screen.getByTestId("integration-chip")).toHaveTextContent("GitHub:github");
    expect(screen.queryByRole("link", { name: "GitHub" })).not.toBeInTheDocument();
  });

  it("keeps regular markdown links on native anchors", () => {
    render(<MarkdownContent content={'Open [docs](../docs "Local docs").'} />);

    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("href", "../docs");
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("title", "Local docs");
    expect(screen.getByRole("link", { name: "docs" })).not.toHaveAttribute("target");
  });

  it("applies shared console link and inline-code styles", () => {
    const { container } = render(<MarkdownContent content={"See [docs](https://example.com) and `sha`."} />);

    expect(container.firstChild).toHaveClass("[&_a]:text-sky-600");
    expect(container.firstChild).toHaveClass("[&_a]:no-underline");
    expect(container.firstChild).toHaveClass("[&_code]:bg-gray-950/5");
    expect(container.firstChild).not.toHaveClass("[&_a]:underline");
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

  it("renders bold markdown with semibold weight", () => {
    const { container } = render(<MarkdownContent content={"**Claude Managed Agent**"} />);
    expect(container.firstChild).toHaveClass("[&_strong]:font-semibold");
    expect(screen.getByText("Claude Managed Agent").tagName).toBe("STRONG");
  });

  it("applies section spacing classes directly on headings", () => {
    render(<MarkdownContent content={"Intro\n\n## Section title\n\nBody copy."} />);
    const h2 = screen.getByRole("heading", { level: 2, name: "Section title" });
    expect(h2.className).toContain("my-4");
    expect(h2.className).toContain("first:mt-0");
  });

  it("uses matching borders and semibold headers in tables", () => {
    render(<MarkdownContent content={"| Stage | Note |\n| --- | --- |\n| Build | **Failed** |\n"} />);
    const th = screen.getByRole("columnheader", { name: "Stage" });
    const td = screen.getByRole("cell", { name: "Failed" });
    expect(th.className).toContain("border-slate-200");
    expect(th.className).toContain("font-semibold");
    expect(td.className).toContain("border-slate-200");
    expect(td.className).not.toContain("border-slate-100");
    expect(screen.getByText("Failed").tagName).toBe("STRONG");
    expect(screen.getByText("Failed").className).toContain("font-semibold");
  });
});
