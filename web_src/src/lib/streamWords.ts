const STREAM_WORD_CLASS = "sp-stream-w";
const STREAMING_WORD_CLASS = "is-streaming";
const DEFAULT_STREAM_GAP_MS = 28;
const MAX_WORD_STREAM_DURATION_MS = 1800;
const MAX_STREAM_WORDS = 200;
const SKIP_PARENTS = new Set(["SCRIPT", "STYLE", "TEXTAREA", "NOSCRIPT"]);
const STRUCTURE_LINE = /^(?:[ \t]*#{1,6}[ \t]+|[ \t]*(?:[-*+]|\d+\.)[ \t]+)/;

export function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true;
}

export function streamGapMs(): number {
  const raw = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--stream-gap"));
  return Number.isFinite(raw) && raw > 0 ? raw : DEFAULT_STREAM_GAP_MS;
}

/**
 * Reveal units for a markdown file being written. Headings and list items stay
 * whole so markers do not appear before their text. Paragraphs resolve by word.
 */
export function streamUnits(content: string): string[] {
  const units: string[] = [];
  let pendingBlank = "";
  for (const chunk of content.split(/(?<=\n)/)) {
    if (/^[ \t]*\n$/.test(chunk)) {
      pendingBlank += chunk;
      continue;
    }
    if (STRUCTURE_LINE.test(chunk)) {
      units.push(pendingBlank + chunk);
      pendingBlank = "";
      continue;
    }
    const words = chunk.match(/\s*\S+/g);
    if (!words) {
      pendingBlank += chunk;
      continue;
    }
    const trailing = chunk.slice(words.join("").length);
    words[words.length - 1] += trailing;
    units.push(pendingBlank + words[0], ...words.slice(1));
    pendingBlank = "";
  }
  if (pendingBlank) {
    units.push(pendingBlank);
  }
  return units;
}

/** Prefix of the markdown after this many generate units. */
export function visibleGeneratedMarkdown(content: string, unitCount: number): string {
  if (!Number.isFinite(unitCount)) {
    return content;
  }
  return streamUnits(content).slice(0, Math.max(0, unitCount)).join("");
}

function acceptTextNode(node: Node): number {
  const parent = node.parentElement;
  if (!parent || SKIP_PARENTS.has(parent.tagName) || parent.closest(`.${STREAM_WORD_CLASS}`)) {
    return NodeFilter.FILTER_REJECT;
  }
  if (!node.textContent?.trim()) {
    return NodeFilter.FILTER_REJECT;
  }
  return NodeFilter.FILTER_ACCEPT;
}

/** Wrap each word in a `.sp-stream-w` span. Spaces stay as text nodes. */
export function wrapStreamWords(root: HTMLElement): HTMLElement[] {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, { acceptNode: acceptTextNode });
  const nodes: Text[] = [];
  while (walker.nextNode()) {
    nodes.push(walker.currentNode as Text);
  }

  const spans: HTMLElement[] = [];
  for (const node of nodes) {
    const parts = node.textContent?.split(/(\s+)/) ?? [];
    const fragment = document.createDocumentFragment();
    for (const part of parts) {
      if (!part) {
        continue;
      }
      if (/^\s+$/.test(part)) {
        fragment.appendChild(document.createTextNode(part));
        continue;
      }
      const span = document.createElement("span");
      span.className = STREAM_WORD_CLASS;
      span.textContent = part;
      fragment.appendChild(span);
      spans.push(span);
    }
    node.parentNode?.replaceChild(fragment, node);
  }
  return spans;
}

/** Resolve each word through opacity and a small blur with CSS-only staggering. */
export function playStreamWords(spans: HTMLElement[]): () => void {
  if (spans.length === 0) {
    return () => {};
  }

  if (prefersReducedMotion()) {
    for (const span of spans) {
      span.classList.add("is-in");
    }
    return () => {};
  }

  const gap = Math.min(streamGapMs(), MAX_WORD_STREAM_DURATION_MS / Math.max(1, spans.length - 1));
  for (const [index, span] of spans.entries()) {
    span.classList.remove("is-in");
    span.style.setProperty("--stream-delay", `${Math.round(index * gap)}ms`);
    span.classList.add(STREAMING_WORD_CLASS);
  }

  const lastSpan = spans.at(-1);
  let complete = false;
  const finish = () => {
    if (complete) return;
    complete = true;
    lastSpan?.removeEventListener("animationend", finish);
    for (const span of spans) {
      span.classList.remove(STREAMING_WORD_CLASS);
      span.classList.add("is-in");
      span.style.removeProperty("--stream-delay");
    }
  };
  lastSpan?.addEventListener("animationend", finish, { once: true });
  return finish;
}

/** Wrap once, or replay spans already in the tree (React Strict Mode). */
export function streamWordsIn(root: HTMLElement): () => void {
  if (prefersReducedMotion()) {
    return () => {};
  }

  const existing = [...root.querySelectorAll<HTMLElement>(`.${STREAM_WORD_CLASS}`)];
  const wordCount = existing.length || root.textContent?.match(/\S+/g)?.length || 0;
  if (wordCount > MAX_STREAM_WORDS) {
    existing.forEach((span) => span.classList.add("is-in"));
    root.classList.add("sp-stream-text");
    const finish = () => {
      root.removeEventListener("animationend", finish);
      root.classList.remove("sp-stream-text");
    };
    root.addEventListener("animationend", finish, { once: true });
    return finish;
  }
  return playStreamWords(existing.length > 0 ? existing : wrapStreamWords(root));
}
