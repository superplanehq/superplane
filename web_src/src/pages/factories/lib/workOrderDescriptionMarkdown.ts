import type { Editor } from "@tiptap/core";
import "@tiptap/markdown";

const MARKDOWN_BLOCK_START = /^(#{1,6}\s|[-*+]\s|\d+\.\s|>\s|```)/m;
const MARKDOWN_INLINE = /(?:\*\*|__|`[^`]+`)/;

export function looksLikeMarkdown(text: string): boolean {
  return MARKDOWN_BLOCK_START.test(text) || MARKDOWN_INLINE.test(text);
}

export function pasteMarkdownFromClipboard(editor: Editor, event: ClipboardEvent): boolean {
  const text = event.clipboardData?.getData("text/plain") ?? "";
  if (!text || !looksLikeMarkdown(text)) {
    return false;
  }

  if (editor.isEmpty) {
    return editor.commands.setContent(text, { contentType: "markdown" });
  }

  return editor.commands.insertContent(text, { contentType: "markdown" });
}
