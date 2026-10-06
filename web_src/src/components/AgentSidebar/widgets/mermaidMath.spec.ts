import { createRequire } from "node:module";
import { describe, expect, it } from "bun:test";
import mermaid from "mermaid";

const require = createRequire(import.meta.url);
const installedKatexVersion = require("katex/package.json").version as string;

const MATH_DIAGRAM = 'flowchart LR\n  A["$$a^2+b^2=c^2$$"]';

describe("mermaid math rendering", () => {
  it("renders diagram math with installed KaTeX 0.18.2", async () => {
    expect(installedKatexVersion).toBe("0.18.2");

    if (document.compatMode !== "CSS1Compat") {
      Object.defineProperty(document, "compatMode", {
        configurable: true,
        get: () => "CSS1Compat",
      });
    }

    if (typeof window.MathMLElement === "undefined") {
      Object.defineProperty(window, "MathMLElement", {
        configurable: true,
        value: class MathMLElement extends HTMLElement {},
      });
    }

    mermaid.initialize({
      startOnLoad: false,
      securityLevel: "loose",
    });

    const { svg } = await mermaid.render("mermaid-math-katex", MATH_DIAGRAM);

    expect(svg).toContain("<msup><mi>a</mi><mn>2</mn></msup>");
    expect(svg).toContain("<msup><mi>b</mi><mn>2</mn></msup>");
    expect(svg).toContain("<msup><mi>c</mi><mn>2</mn></msup>");
    expect(svg).not.toContain("$$");
    expect(svg).not.toContain("MathML is unsupported");
  });
});
