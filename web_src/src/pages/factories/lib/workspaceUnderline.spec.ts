import { Editor } from "@tiptap/core";
import { Markdown } from "@tiptap/markdown";
import StarterKit from "@tiptap/starter-kit";
import { afterEach, describe, expect, it } from "vitest";

import { WorkspaceUnderline } from "./workspaceUnderline";

function createEditor(content: string): Editor {
  return new Editor({
    extensions: [
      StarterKit.configure({
        underline: false,
      }),
      WorkspaceUnderline,
      Markdown,
    ],
    content,
    contentType: "markdown",
  });
}

function hasMark(node: { marks?: { type: string }[]; content?: unknown[] }, type: string): boolean {
  if (node.marks?.some((mark) => mark.type === type)) {
    return true;
  }
  return (node.content ?? []).some((child) => hasMark(child as { marks?: { type: string }[]; content?: unknown[] }, type));
}

describe("WorkspaceUnderline", () => {
  let editor: Editor | undefined;

  afterEach(() => {
    editor?.destroy();
  });

  it("stores underline as HTML instead of ++ delimiters", () => {
    editor = createEditor("Hello world");
    editor.commands.selectAll();
    editor.commands.toggleUnderline();

    expect(editor.getMarkdown()).toContain("<u>Hello world</u>");
    expect(editor.getMarkdown()).not.toContain("++");
  });

  it("round-trips underline next to punctuation and nested marks", () => {
    editor = createEditor("See **<u>word</u>**.");

    expect(hasMark(editor.getJSON(), "underline")).toBe(true);
    expect(editor.getMarkdown()).toContain("<u>word</u>");
    expect(editor.getMarkdown()).not.toContain("++");
  });

  it("does not parse increment operators as underline", () => {
    editor = createEditor("i++ then j++");

    expect(hasMark(editor.getJSON(), "underline")).toBe(false);
    expect(editor.getMarkdown()).toContain("i++ then j++");
    expect(editor.getMarkdown()).not.toContain("<u>");
  });
});
