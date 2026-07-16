import { describe, expect, it } from "vitest";
import { exprLangAdapter, resolveExprLangSuggestionValue, evaluateExprLang } from "./exprLangAdapter";

describe("evaluateExprLang", () => {
  it("returns ok and formatted result for a valid expression", () => {
    const outcome = evaluateExprLang("root().data.name", { __root: { data: { name: "DCO" } } });
    expect(outcome.ok).toBe(true);
    expect(outcome.value).toBe("DCO");
    expect(outcome.formattedValue).toBe("DCO");
  });

  it("returns ok:false when the expression is invalid", () => {
    const outcome = evaluateExprLang("root().data.missing", { __root: { data: {} } });
    expect(outcome.ok).toBe(true);
    expect(outcome.value).toBeUndefined();
  });

  it("reports no context when globals is nullish", () => {
    const outcome = evaluateExprLang("1 + 1", null);
    expect(outcome.ok).toBe(false);
    expect(outcome.error).toBe("No context available");
  });

  it("captures thrown errors from the evaluator", () => {
    const outcome = evaluateExprLang("root() +", { __root: {} });
    expect(outcome.ok).toBe(false);
    expect(outcome.error).toBeTruthy();
  });
});

describe("resolveExprLangSuggestionValue", () => {
  const globals = {
    __root: { data: { name: "DCO", sha: "abc123" } },
    __previousByDepth: { "1": { data: { name: "prev" } } },
    "deploy-node": { outputs: { url: "https://example.com" } },
  };

  it("resolves root().data.name against the __root path", () => {
    expect(resolveExprLangSuggestionValue("root().data.name", globals)).toBe("DCO");
  });

  it("resolves previous(1).data via __previousByDepth", () => {
    expect(resolveExprLangSuggestionValue("previous(1).data.name", globals)).toBe("prev");
  });

  it("resolves $['deploy-node'].outputs.url", () => {
    expect(resolveExprLangSuggestionValue("$['deploy-node'].outputs.url", globals)).toBe("https://example.com");
  });

  it("returns undefined for invalid path syntax", () => {
    expect(resolveExprLangSuggestionValue("root().data.[bogus]", globals)).toBeUndefined();
  });

  it("returns undefined when previous() has an invalid depth", () => {
    expect(resolveExprLangSuggestionValue("previous(zero).data", globals)).toBeUndefined();
  });
});

describe("exprLangAdapter", () => {
  it("exposes the expr-lang id and wired functions", () => {
    expect(exprLangAdapter.id).toBe("expr-lang");
    expect(exprLangAdapter.evaluate).toBe(evaluateExprLang);
    expect(exprLangAdapter.resolveSuggestionValue).toBe(resolveExprLangSuggestionValue);
    expect(exprLangAdapter.formatResult("hello")).toBe("hello");
  });
});
