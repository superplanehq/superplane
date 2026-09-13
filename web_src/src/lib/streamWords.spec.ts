import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  playStreamWords,
  streamUnits,
  streamWordsIn,
  visibleGeneratedMarkdown,
  wrapStreamWords,
} from "@/lib/streamWords";

function reducedMotionMatchMedia(matches: boolean) {
  return (query: string) => ({
    matches: matches && query.includes("prefers-reduced-motion"),
    media: query,
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
  });
}

describe("streamWords", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    document.documentElement.style.setProperty("--stream-gap", "60ms");
    vi.stubGlobal("matchMedia", reducedMotionMatchMedia(false));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    document.documentElement.style.removeProperty("--stream-gap");
  });

  it("wraps each word in a stream span and keeps spaces between them", () => {
    const root = document.createElement("div");
    root.textContent = "Clearer empty state";

    const spans = wrapStreamWords(root);

    expect(spans.map((span) => span.textContent)).toEqual(["Clearer", "empty", "state"]);
    expect(spans.every((span) => span.className === "sp-stream-w")).toBe(true);
    expect(root.textContent).toBe("Clearer empty state");
  });

  it("wraps markdown text nodes without breaking headings", () => {
    const root = document.createElement("div");
    const heading = document.createElement("h3");
    heading.textContent = "Done when";
    const paragraph = document.createElement("p");
    paragraph.append("The empty view names the next action.");
    root.append(heading, paragraph);

    const spans = wrapStreamWords(root);

    expect(heading.querySelectorAll(".sp-stream-w")).toHaveLength(2);
    expect(heading.textContent).toBe("Done when");
    expect(spans[0]?.textContent).toBe("Done");
    expect(paragraph.textContent).toBe("The empty view names the next action.");
  });

  it("resolves words one by one through is-in", () => {
    const root = document.createElement("div");
    root.textContent = "one two three";
    const spans = wrapStreamWords(root);

    playStreamWords(spans);

    expect(spans[0]?.classList.contains("is-in")).toBe(true);
    expect(spans[1]?.classList.contains("is-in")).toBe(false);
    expect(spans[2]?.classList.contains("is-in")).toBe(false);

    vi.advanceTimersByTime(60);
    expect(spans[1]?.classList.contains("is-in")).toBe(true);
    expect(spans[2]?.classList.contains("is-in")).toBe(false);

    vi.advanceTimersByTime(60);
    expect(spans[2]?.classList.contains("is-in")).toBe(true);
  });

  it("shows every word at once when the user prefers reduced motion", () => {
    vi.stubGlobal("matchMedia", reducedMotionMatchMedia(true));
    const root = document.createElement("div");
    root.textContent = "one two";

    streamWordsIn(root);

    expect([...root.querySelectorAll(".sp-stream-w")].every((span) => span.classList.contains("is-in"))).toBe(true);
    vi.advanceTimersByTime(120);
  });

  it("keeps a heading or list line as one generated unit", () => {
    expect(streamUnits("### Goal\n\nExtend the demo.\n\n- First item.\n")).toEqual([
      "### Goal\n",
      "\nExtend",
      " the",
      " demo.\n",
      "\n- First item.\n",
    ]);
  });

  it("hides later markdown until earlier units are written", () => {
    const spec = "### Goal\n\nExtend the demo.\n\n- First item.\n- Second item.\n";

    expect(visibleGeneratedMarkdown(spec, 1)).toBe("### Goal\n");
    expect(visibleGeneratedMarkdown(spec, 2)).toBe("### Goal\n\nExtend");
    expect(visibleGeneratedMarkdown(spec, 5)).toBe("### Goal\n\nExtend the demo.\n\n- First item.\n");
    expect(visibleGeneratedMarkdown(spec, Number.POSITIVE_INFINITY)).toBe(spec);
  });

  it("replays existing spans instead of wrapping them twice", () => {
    const root = document.createElement("div");
    root.textContent = "one two";
    streamWordsIn(root);
    vi.advanceTimersByTime(60);

    streamWordsIn(root);

    expect(root.querySelectorAll(".sp-stream-w")).toHaveLength(2);
    expect(root.querySelectorAll(".sp-stream-w.is-in")).toHaveLength(1);
    vi.advanceTimersByTime(60);
    expect(root.querySelectorAll(".sp-stream-w.is-in")).toHaveLength(2);
  });
});
