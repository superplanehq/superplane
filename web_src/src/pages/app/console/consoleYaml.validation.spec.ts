import { describe, expect, it } from "vitest";

import { validateConsoleContent, MAX_CONSOLE_PANELS } from "./consoleYaml";

describe("validateConsoleContent", () => {
  it("flags too many panels", () => {
    const panels = Array.from({ length: MAX_CONSOLE_PANELS + 1 }, (_, i) => ({
      id: `p${i}`,
      type: "markdown",
      content: {},
    }));
    expect(validateConsoleContent(panels, [])).toContain("Too many panels");
  });

  it("flags layout with non-positive size", () => {
    expect(
      validateConsoleContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: 0, y: 0, w: 0, h: 1 }]),
    ).toContain("positive width and height");
  });

  it("flags negative position", () => {
    expect(
      validateConsoleContent([{ id: "p", type: "markdown", content: {} }], [{ i: "p", x: -1, y: 0, w: 1, h: 1 }]),
    ).toContain("non-negative");
  });

  it("accepts a valid console", () => {
    expect(
      validateConsoleContent(
        [{ id: "p", type: "markdown", content: { body: "ok" } }],
        [{ i: "p", x: 0, y: 0, w: 1, h: 1 }],
      ),
    ).toBeNull();
  });
});
