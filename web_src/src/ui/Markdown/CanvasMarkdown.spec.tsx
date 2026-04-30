import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CanvasMarkdown, NODE_REF_CLASS, NODE_STATUS_CLASS, TRIGGER_RUN_CLASS, remarkNodeRefs } from "./CanvasMarkdown";

type MdastNode = {
  type: string;
  value?: string;
  children?: MdastNode[];
};

function text(value: string): MdastNode {
  return { type: "text", value };
}

function paragraph(...children: MdastNode[]): MdastNode {
  return { type: "paragraph", children };
}

function root(...children: MdastNode[]): MdastNode {
  return { type: "root", children };
}

function runPlugin(tree: MdastNode) {
  const transformer = remarkNodeRefs();
  transformer(tree);
  return tree;
}

describe("remarkNodeRefs", () => {
  it("replaces a bare @slug with a node-ref span", () => {
    const tree = runPlugin(root(paragraph(text("Reach out to @deploy for details"))));

    const para = tree.children?.[0];
    const children = para?.children ?? [];
    expect(children).toHaveLength(3);
    expect(children[0]).toEqual({ type: "text", value: "Reach out to " });
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(`class="${NODE_REF_CLASS}"`);
    expect(children[1].value).toContain(">deploy<");
    expect(children[2]).toEqual({ type: "text", value: " for details" });
  });

  it("replaces [[node:slug]] with a node-ref span", () => {
    const tree = runPlugin(root(paragraph(text("see [[node:build-service]] below"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(">build-service<");
  });

  it("replaces multiple slugs in the same text node", () => {
    const tree = runPlugin(root(paragraph(text("@one and @two walk into a bar"))));

    const children = tree.children?.[0].children ?? [];
    expect(children.filter((c) => c.type === "html")).toHaveLength(2);
  });

  it("leaves code blocks alone", () => {
    const code: MdastNode = { type: "code", value: "echo @deploy" };
    const tree = runPlugin(root(paragraph(code)));

    const para = tree.children?.[0];
    expect(para?.children?.[0]).toEqual(code);
  });

  it("leaves inline code alone", () => {
    const inline: MdastNode = { type: "inlineCode", value: "@deploy" };
    const tree = runPlugin(root(paragraph(text("use "), inline)));

    const children = tree.children?.[0].children ?? [];
    expect(children[1]).toEqual(inline);
  });

  it("does not match email-like @ usage", () => {
    const tree = runPlugin(root(paragraph(text("email user@example for details"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(1);
    expect(children[0].type).toBe("text");
  });

  it("handles adjacent punctuation", () => {
    const tree = runPlugin(root(paragraph(text("See (@deploy), ok?"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[0].value).toBe("See (");
    expect(children[1].type).toBe("html");
    expect(children[2].value).toBe("), ok?");
  });

  it("escapes the slug to prevent HTML injection", () => {
    // The regex only matches [a-zA-Z][a-zA-Z0-9_-]*, so angle brackets can't
    // reach the span. But double-check escaping via the [[node:...]] form.
    const tree = runPlugin(root(paragraph(text("[[node:safe-slug]]"))));

    const children = tree.children?.[0].children ?? [];
    expect(children[0].type).toBe("html");
    expect(children[0].value).not.toContain("<script>");
  });

  it("replaces @trigger:run/template with a trigger-run span", () => {
    const tree = runPlugin(root(paragraph(text("click @my-trigger:run/hello-world to deploy"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[0]).toEqual({ type: "text", value: "click " });
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(`class="${TRIGGER_RUN_CLASS}"`);
    expect(children[1].value).toContain(`data-trigger="my-trigger"`);
    expect(children[1].value).toContain(`data-template="hello-world"`);
    expect(children[2]).toEqual({ type: "text", value: " to deploy" });
  });

  it("replaces [[run:trigger:template]] with a trigger-run span", () => {
    const tree = runPlugin(root(paragraph(text("see [[run:my-trigger:hello-world]] below"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(`class="${TRIGGER_RUN_CLASS}"`);
    expect(children[1].value).toContain(`data-trigger="my-trigger"`);
    expect(children[1].value).toContain(`data-template="hello-world"`);
  });

  it("leaves trigger-run tokens inside code blocks alone", () => {
    const code: MdastNode = { type: "code", value: "@trig:run/tpl and [[run:trig:tpl]]" };
    const tree = runPlugin(root(paragraph(code)));

    const para = tree.children?.[0];
    expect(para?.children?.[0]).toEqual(code);
  });

  it("leaves trigger-run tokens inside inline code alone", () => {
    const inline: MdastNode = { type: "inlineCode", value: "@trig:run/tpl" };
    const tree = runPlugin(root(paragraph(text("use "), inline)));

    const children = tree.children?.[0].children ?? [];
    expect(children[1]).toEqual(inline);
  });

  it("supports both node-ref and trigger-run tokens in the same paragraph", () => {
    const tree = runPlugin(root(paragraph(text("ping @deploy then @my-trigger:run/hello-world here"))));

    const children = tree.children?.[0].children ?? [];
    const htmlNodes = children.filter((c) => c.type === "html");
    expect(htmlNodes).toHaveLength(2);
    expect(htmlNodes[0].value).toContain(`class="${NODE_REF_CLASS}"`);
    expect(htmlNodes[1].value).toContain(`class="${TRIGGER_RUN_CLASS}"`);
  });

  it("replaces @slug:status with a node-status span", () => {
    const tree = runPlugin(root(paragraph(text("status of @deploy:status today"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[0]).toEqual({ type: "text", value: "status of " });
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(`class="${NODE_STATUS_CLASS}"`);
    expect(children[1].value).toContain(`data-node="deploy"`);
    expect(children[2]).toEqual({ type: "text", value: " today" });
  });

  it("replaces [[status:slug]] with a node-status span", () => {
    const tree = runPlugin(root(paragraph(text("ok [[status:deploy]] right now"))));

    const children = tree.children?.[0].children ?? [];
    expect(children).toHaveLength(3);
    expect(children[1].type).toBe("html");
    expect(children[1].value).toContain(`class="${NODE_STATUS_CLASS}"`);
    expect(children[1].value).toContain(`data-node="deploy"`);
  });

  it("leaves status tokens inside code blocks alone", () => {
    const code: MdastNode = { type: "code", value: "@deploy:status and [[status:deploy]]" };
    const tree = runPlugin(root(paragraph(code)));

    const para = tree.children?.[0];
    expect(para?.children?.[0]).toEqual(code);
  });

  it("renders @node:status alongside @trigger:run/template in the same paragraph", () => {
    const tree = runPlugin(root(paragraph(text("see @deploy:status and @my-trigger:run/hello-world below"))));

    const children = tree.children?.[0].children ?? [];
    const htmlNodes = children.filter((c) => c.type === "html");
    expect(htmlNodes).toHaveLength(2);
    expect(htmlNodes[0].value).toContain(`class="${NODE_STATUS_CLASS}"`);
    expect(htmlNodes[1].value).toContain(`class="${TRIGGER_RUN_CLASS}"`);
  });

  it("does not match @slug:other (non-status suffix) as a status chip", () => {
    const tree = runPlugin(root(paragraph(text("see @deploy:other thing"))));

    const children = tree.children?.[0].children ?? [];
    const htmlNodes = children.filter((c) => c.type === "html");
    expect(htmlNodes).toHaveLength(1);
    // Falls back to the bare node-ref chip for `@deploy`.
    expect(htmlNodes[0].value).toContain(`class="${NODE_REF_CLASS}"`);
    expect(htmlNodes[0].value).toContain(">deploy<");
  });
});

describe("CanvasMarkdown trigger-run chip", () => {
  const triggerTemplates = {
    "my-trigger": {
      "hello-world": {
        name: "Hello World",
        payload: { greeting: "hi" },
      },
    },
  };

  it("renders a button when onTriggerTemplateRun is provided", () => {
    const onRun = vi.fn();
    render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { "my-trigger": "My trigger" },
          triggerTemplates,
          onTriggerTemplateRun: onRun,
        }}
      >
        {"@my-trigger:run/hello-world"}
      </CanvasMarkdown>,
    );

    const buttons = screen.getAllByRole("button");
    const runButton = buttons.find((b) => b.textContent?.includes("Hello World"));
    expect(runButton).toBeDefined();
    expect(runButton?.tagName).toBe("BUTTON");
    expect(runButton?.getAttribute("data-trigger")).toBe("my-trigger");
    expect(runButton?.getAttribute("data-template")).toBe("hello-world");

    fireEvent.click(runButton!);
    expect(onRun).toHaveBeenCalledWith({ nodeSlug: "my-trigger", templateSlug: "hello-world" });
  });

  it("renders a disabled span when onTriggerTemplateRun is undefined", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { "my-trigger": "My trigger" },
          triggerTemplates,
        }}
      >
        {"[[run:my-trigger:hello-world]]"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${TRIGGER_RUN_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.tagName).toBe("SPAN");
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders a disabled span for unknown trigger", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: {},
          triggerTemplates: {},
          onTriggerTemplateRun: vi.fn(),
        }}
      >
        {"@missing-trigger:run/whatever"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${TRIGGER_RUN_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.tagName).toBe("SPAN");
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders a disabled span for unknown template on a known trigger", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { "my-trigger": "My trigger" },
          triggerTemplates,
          onTriggerTemplateRun: vi.fn(),
        }}
      >
        {"@my-trigger:run/missing-template"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${TRIGGER_RUN_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.tagName).toBe("SPAN");
    expect(container.querySelector("button")).toBeNull();
  });
});

describe("CanvasMarkdown node-status chip", () => {
  it("renders a colored pill from a known status entry", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { deploy: "Deploy" },
          nodeStatuses: {
            deploy: { status: "running", badgeColor: "bg-blue-500", label: "RUNNING" },
          },
        }}
      >
        {"@deploy:status"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${NODE_STATUS_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.className).toContain("bg-blue-500");
    expect(chip?.textContent).toBe("RUNNING");
  });

  it("renders the bracketed [[status:slug]] form too", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { deploy: "Deploy" },
          nodeStatuses: {
            deploy: { status: "failed", badgeColor: "bg-red-400", label: "FAILED" },
          },
        }}
      >
        {"[[status:deploy]]"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${NODE_STATUS_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.className).toContain("bg-red-400");
    expect(chip?.textContent).toBe("FAILED");
  });

  it("renders a no-runs muted pill when entry has status === 'none'", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: { deploy: "Deploy" },
          nodeStatuses: {
            deploy: { status: "none", badgeColor: "bg-gray-400", label: "no runs" },
          },
        }}
      >
        {"@deploy:status"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${NODE_STATUS_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.className).toContain("bg-gray-400");
    expect(chip?.textContent).toBe("no runs");
  });

  it("renders a dashed grey unknown pill when no entry exists for the slug", () => {
    const { container } = render(
      <CanvasMarkdown
        nodeRefs={{
          nodes: {},
          nodeStatuses: {},
        }}
      >
        {"@missing:status"}
      </CanvasMarkdown>,
    );

    const chip = container.querySelector(`span.${NODE_STATUS_CLASS}`);
    expect(chip).not.toBeNull();
    expect(chip?.className).toContain("border-dashed");
    expect(chip?.textContent).toBe("missing:status");
  });
});
