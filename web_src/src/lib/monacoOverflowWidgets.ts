/**
 * Body-level host for Monaco's overflow widgets (suggest, hover, parameter
 * hints). Mounted outside the configuration sidebar so widgets escape its
 * `overflow: hidden` and `transform`-induced containing block.
 *
 * Fixes #1804.
 */

const OVERFLOW_WIDGETS_NODE_ID = "monaco-overflow-widgets-host";

let cachedNode: HTMLElement | null = null;

export function getMonacoOverflowWidgetsNode(): HTMLElement | undefined {
  if (typeof document === "undefined") {
    return undefined;
  }

  if (cachedNode && document.body.contains(cachedNode)) {
    return cachedNode;
  }

  const existing = document.getElementById(OVERFLOW_WIDGETS_NODE_ID);
  if (existing instanceof HTMLElement) {
    cachedNode = existing;
    return cachedNode;
  }

  const node = document.createElement("div");
  node.id = OVERFLOW_WIDGETS_NODE_ID;
  // `monaco-editor` class is required so Monaco's widget styles still scope
  // correctly when widgets render outside the original editor.
  node.className = "monaco-editor";
  node.style.position = "absolute";
  node.style.top = "0";
  node.style.left = "0";
  node.style.zIndex = "10000";

  document.body.appendChild(node);
  cachedNode = node;
  return cachedNode;
}
