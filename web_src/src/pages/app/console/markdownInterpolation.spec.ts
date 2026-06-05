import { describe, expect, it } from "vitest";

import { DOLLAR_REWRITE_IDENTIFIER } from "./widget/celExpr";
import { interpolateMarkdownTemplate } from "./markdownInterpolation";

describe("interpolateMarkdownTemplate", () => {
  it("returns empty for nullish input", () => {
    expect(interpolateMarkdownTemplate(undefined, {})).toBe("");
    expect(interpolateMarkdownTemplate("", {})).toBe("");
  });

  it("passes plain markdown through unchanged when no template segments are present", () => {
    const md = "# Hello\n\nThis is plain markdown.";
    expect(interpolateMarkdownTemplate(md, { foo: "bar" })).toBe(md);
  });

  it("substitutes a single variable field", () => {
    const out = interpolateMarkdownTemplate("Status: {{ run.status }}", { run: { status: "passed" } });
    expect(out).toBe("Status: passed");
  });

  it("renders multiple variables and preserves surrounding markdown", () => {
    const out = interpolateMarkdownTemplate(
      "## {{ recipe.title }}\n\n- run: {{ run.status }}\n- by: {{ run.nodeName }}",
      { recipe: { title: "Deploy prod" }, run: { status: "failed", nodeName: "build" } },
    );
    expect(out).toBe("## Deploy prod\n\n- run: failed\n- by: build");
  });

  it("renders the run dollar-node accessor via the CEL rewrite alias", () => {
    const dollar = { Deploy: { data: { url: "https://example.com" } } };
    const out = interpolateMarkdownTemplate('Deployed to {{ run.$["Deploy"].data.url }}', {
      run: { $: dollar, [DOLLAR_REWRITE_IDENTIFIER]: dollar },
    });
    expect(out).toBe("Deployed to https://example.com");
  });

  it("renders empty string for unresolved variables instead of throwing", () => {
    const out = interpolateMarkdownTemplate("Missing: {{ nope.field }}", {});
    expect(out).toBe("Missing: ");
  });

  it("serializes object values as JSON for inline insertion", () => {
    const out = interpolateMarkdownTemplate("Payload: {{ run.payload }}", { run: { payload: { pr: 7 } } });
    expect(out).toBe('Payload: {"pr":7}');
  });
});
