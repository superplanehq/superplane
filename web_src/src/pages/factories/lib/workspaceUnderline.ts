import Underline from "@tiptap/extension-underline";

/**
 * Store underline as HTML so CommonMark renderers can show it without a ++
 * delimiter that collides with increment operators and C++.
 */
export const WorkspaceUnderline = Underline.extend({
  renderMarkdown: (node, helpers) => `<u>${helpers.renderChildren(node)}</u>`,
  markdownTokenizer: {
    name: "underline",
    level: "inline",
    start(src) {
      return src.indexOf("<u>");
    },
    tokenize(src, _tokens, lexer) {
      const match = /^(<u>)([\s\S]+?)(<\/u>)/.exec(src);
      if (!match) {
        return undefined;
      }

      const innerContent = match[2];
      return {
        type: "underline",
        raw: match[0],
        text: innerContent,
        tokens: lexer.inlineTokens(innerContent),
      };
    },
  },
});
