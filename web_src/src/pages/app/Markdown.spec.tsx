import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

  it.each([
    ["NOTE", "Note"],
    ["TIP", "Tip"],
    ["IMPORTANT", "Important"],
    ["WARNING", "Warning"],
    ["CAUTION", "Caution"],
  ] as const)("renders GitHub %s alerts with SuperPlane chrome", (type, label) => {
    render(<MarkdownContent content={`> [!${type}]\n> Useful ${label.toLowerCase()} details.`} />);

    const alert = screen.getByTestId(`markdown-alert-${type.toLowerCase()}`);
    expect(alert.tagName).toBe("ASIDE");
    expect(alert).toHaveTextContent(label);
    expect(alert).toHaveTextContent(`Useful ${label.toLowerCase()} details.`);
    expect(alert).not.toHaveTextContent(`[!${type}]`);
    expect(document.querySelector("blockquote")).toBeNull();
  });

  it("keeps unknown alert markers as plain blockquotes", () => {
    render(<MarkdownContent content={"> [!TODO]\n> Ship it later."} />);

    expect(screen.queryByTestId(/markdown-alert-/)).not.toBeInTheDocument();
    expect(screen.getByText(/Ship it later/)).toBeInTheDocument();
    expect(document.querySelector("blockquote")).toBeTruthy();
  });

  it("preserves nested markdown inside alert bodies", () => {
    render(<MarkdownContent content={"> [!TIP]\n> See [docs](https://example.com) and `sha`."} />);

    const alert = screen.getByTestId("markdown-alert-tip");
    expect(alert).toHaveTextContent("Tip");
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("href", "https://example.com");
    expect(screen.getByTestId("markdown-code")).toHaveTextContent("sha");
  });

  it("renders [!SECTION] blockquotes as collapsed accordions", async () => {
    const user = userEvent.setup();
    render(
      <MarkdownContent
        content={"> [!SECTION] Rules\n> Standing instructions for the agent.\n>\n> Keep them focused."}
      />,
    );

    const section = screen.getByTestId("markdown-section");
    expect(section).toHaveTextContent("Rules");
    expect(section).not.toHaveTextContent("[!SECTION]");
    expect(screen.queryByText(/Standing instructions/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Rules/i }));
    expect(screen.getByText(/Standing instructions for the agent/)).toBeInTheDocument();
    expect(screen.getByText(/Keep them focused/)).toBeInTheDocument();
  });

  it("renders optional section trailing meta after a middle dot", () => {
    render(<MarkdownContent content={"> [!SECTION] Rules · ~5,366\n> Body copy."} />);

    const section = screen.getByTestId("markdown-section");
    expect(section).toHaveTextContent("Rules");
    expect(section).toHaveTextContent("~5,366");
    expect(section).not.toHaveTextContent("[!SECTION]");
  });

  it("applies named section presets for icon and accent color", () => {
    render(<MarkdownContent content={"> [!SECTION:tools] Tool definitions · ~9,202\n> Body copy."} />);

    const section = screen.getByTestId("markdown-section");
    expect(section).toHaveAttribute("data-section-preset", "tools");
    expect(section).toHaveTextContent("Tool definitions");
    expect(section).toHaveTextContent("~9,202");
  });

  it("applies the integrations section preset", () => {
    render(<MarkdownContent content={"> [!SECTION:integrations] Connected integrations\n> Body copy."} />);

    expect(screen.getByTestId("markdown-section")).toHaveAttribute("data-section-preset", "integrations");
  });

  it("shows a count of direct nested sections beside the title", () => {
    render(
      <MarkdownContent
        content={
          "> [!SECTION:rules] Rules · ~5,366\n> Intro.\n>\n> > [!SECTION:folder] Project Rules\n> > One.\n>\n> > [!SECTION:folder] Cursor & User Rules\n> > Two."
        }
      />,
    );

    const section = screen.getByTestId("markdown-section");
    expect(section).toHaveAttribute("data-section-count", "2");
    expect(section).toHaveTextContent("Rules");
    expect(section).toHaveTextContent("2");
  });

  it("does not count sections nested inside alerts as direct children", () => {
    render(
      <MarkdownContent
        content={
          "> [!SECTION] Parent\n> Intro.\n>\n> > [!NOTE]\n> > Note body.\n> >\n> > > [!SECTION] Nested in note\n> > > Deep.\n>\n> > [!SECTION] Direct child\n> > Child body."
        }
      />,
    );

    const section = screen.getByTestId("markdown-section");
    expect(section).toHaveAttribute("data-section-count", "1");
    expect(screen.getByTestId("markdown-section-count")).toHaveTextContent("1");
  });

  it("preserves nested markdown inside section bodies", async () => {
    const user = userEvent.setup();
    render(<MarkdownContent content={"> [!SECTION] Tips\n> See [docs](https://example.com) and `sha`."} />);

    await user.click(screen.getByRole("button", { name: /Tips/i }));
    expect(screen.getByRole("link", { name: "docs" })).toHaveAttribute("href", "https://example.com");
    expect(screen.getByTestId("markdown-code")).toHaveTextContent("sha");
  });

  it("does not treat [!SECTION] without a title as a section", () => {
    render(<MarkdownContent content={"> [!SECTION]\n> Missing title stays a quote."} />);

    expect(screen.queryByTestId("markdown-section")).not.toBeInTheDocument();
    expect(document.querySelector("blockquote")).toBeTruthy();
  });
});
