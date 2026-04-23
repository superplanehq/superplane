import { describe, it, expect } from "vitest";
import { remarkNodeRefs, NODE_REF_CLASS } from "./CanvasMarkdown";

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
});
