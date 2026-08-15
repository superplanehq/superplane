import type { Editor } from "@tiptap/core";
import "@tiptap/markdown";

const MARKDOWN_BLOCK_START = /^(#{1,6}\s|[-*+]\s|\d+\.\s|>\s|```)/m;
const MARKDOWN_PAIRED_ASTERISKS = /\*\*[^*\n]+\*\*/;
const MARKDOWN_CODE = /`[^`]+`/;
const MARKDOWN_UNDERSCORE_EMPHASIS = /(?:^|\s)__[^_\n]+__(?:\s|$)/;
// Same whitespace bounds as __ so increments like i++ then j++ stay plain text.
const MARKDOWN_UNDERLINE = /(?:^|\s)\+\+[^\n+]+\+\+(?:\s|$)/;

export function looksLikeMarkdown(text: string): boolean {
  return (
    MARKDOWN_BLOCK_START.test(text) ||
    MARKDOWN_PAIRED_ASTERISKS.test(text) ||
    MARKDOWN_CODE.test(text) ||
    MARKDOWN_UNDERSCORE_EMPHASIS.test(text) ||
    MARKDOWN_UNDERLINE.test(text)
  );
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
