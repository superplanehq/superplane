import React from "react";

/** Recursively flattens a React node tree into its plain text content. */
export function extractTextFromNode(node: React.ReactNode): string {
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }

  if (Array.isArray(node)) {
    return node.map(extractTextFromNode).join("");
  }

  if (React.isValidElement<{ children?: React.ReactNode }>(node)) {
    return extractTextFromNode(node.props.children);
  }

  return "";
}

/**
 * Extracts the raw code text (and optional language) from the children of a
 * markdown `pre` element, which react-markdown renders as a nested `code` node.
 */
export function extractCodeBlock(children: React.ReactNode): { code: string; language?: string } {
  const childArray = React.Children.toArray(children);

  const codeElement = childArray.find(
    (
      child,
    ): child is React.ReactElement<{
      className?: string;
      children?: React.ReactNode;
    }> => React.isValidElement(child) && child.type === "code",
  );

  if (!codeElement) {
    return { code: extractTextFromNode(children).replace(/\n$/, "") };
  }

  const className = codeElement.props.className;
  const language = className?.startsWith("language-") ? className.slice("language-".length) : undefined;

  return {
    code: extractTextFromNode(codeElement.props.children).replace(/\n$/, ""),
    language,
  };
}
