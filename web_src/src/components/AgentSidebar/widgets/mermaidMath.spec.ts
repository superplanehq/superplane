import { existsSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { describe, expect, it } from "bun:test";
import mermaid from "mermaid";

const require = createRequire(import.meta.url);

function installedKatexVersion(): string {
  const mermaidRoot = dirname(require.resolve("mermaid/package.json"));
  const candidates = [
    join(mermaidRoot, "node_modules", "katex", "package.json"),
    join(mermaidRoot, "..", "katex", "package.json"),
  ];

  for (const candidate of candidates) {
    if (!existsSync(candidate)) {
      continue;
    }

    const parsed = JSON.parse(readFileSync(candidate, "utf8")) as { version?: string };
    if (parsed.version) {
      return parsed.version;
    }
  }

  throw new Error("installed KaTeX package was not found");
}

const installedKatexVersionValue = installedKatexVersion();

const MATH_DIAGRAM = 'flowchart LR\n  A["$$a^2+b^2=c^2$$"]';

describe("mermaid math rendering", () => {
  it("renders diagram math with installed KaTeX 0.18.2", async () => {
    expect(installedKatexVersionValue).toBe("0.18.2");

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
