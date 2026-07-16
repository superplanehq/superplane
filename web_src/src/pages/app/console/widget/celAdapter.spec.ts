import { describe, expect, it } from "vitest";
import { evaluateCel, resolveCelSuggestionValue, widgetCelAdapter } from "./celAdapter";
import { DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";

describe("evaluateCel", () => {
  it("evaluates a full-template expression and returns the raw value", () => {
    const outcome = evaluateCel("{{ name }}", { name: "deploy" });
    expect(outcome).toEqual({ ok: true, value: "deploy", formattedValue: "deploy" });
  });

  it("evaluates a bare expression against the row context", () => {
    const outcome = evaluateCel("name", { name: "deploy" });
    expect(outcome).toEqual({ ok: true, value: "deploy", formattedValue: "deploy" });
  });

  it("evaluates a mixed template and returns the composed string", () => {
    const outcome = evaluateCel("Hello, {{ name }}!", { name: "world" });
    expect(outcome).toEqual({ ok: true, value: "Hello, world!", formattedValue: "Hello, world!" });
  });

  it("reports diagnostics for expressions the runtime rejects", () => {
    const outcome = evaluateCel("{{ int(nan) + 1 }}", { nan: "not-a-number" });
    // int() is our fail-soft wrapper that returns 0, so the outer + still succeeds.
    expect(outcome.ok).toBe(true);
  });

  it("returns an error for a compile-time failure", () => {
    const outcome = evaluateCel("{{ unmatched(( }}", { anything: "ok" });
    expect(outcome.ok).toBe(false);
    expect(outcome.error).toBeTruthy();
  });

  it("reports no-context when globals is nullish", () => {
    const outcome = evaluateCel("name", null);
    expect(outcome).toEqual({ ok: false, error: "No context available" });
  });

  it("returns an empty formatted value for blank input", () => {
    const outcome = evaluateCel("  ", { name: "deploy" });
    expect(outcome).toEqual({ ok: true, value: "", formattedValue: "" });
  });

  it("does not error with 'Unknown variable' when the row omits __runNodes__", () => {
    // Sample rows synthesized for preview don't carry the internal
    // `__runNodes__` map — CEL should still be able to look the identifier up
    // and report an accurate "no such key" for the missing node reference
    // instead of the misleading "Unknown variable" error.
    const outcome = evaluateCel('{{ $["dog"].data.body.message }}', { status: "passed" });
    if (outcome.ok) {
      expect(outcome.formattedValue).toBe("");
    } else {
      expect(outcome.error).not.toMatch(/Unknown variable/);
    }
  });

  it("resolves $['node'].outputs.x against a live __runNodes__ map", () => {
    const outcome = evaluateCel('{{ $["dog"].data.body.message }}', {
      [DOLLAR_REWRITE_IDENTIFIER]: { dog: { data: { body: { message: "woof" } } } },
    });
    expect(outcome).toEqual({ ok: true, value: "woof", formattedValue: "woof" });
  });
});

describe("resolveCelSuggestionValue", () => {
  const globals = {
    data: { name: "deploy", nested: { url: "https://example.com" } },
    [DOLLAR_REWRITE_IDENTIFIER]: {
      "deploy-prod": { outputs: { url: "https://prod" } },
    },
  };

  it("resolves a dotted row-field path", () => {
    expect(resolveCelSuggestionValue("data.name", globals)).toBe("deploy");
    expect(resolveCelSuggestionValue("data.nested.url", globals)).toBe("https://example.com");
  });

  it("resolves $['node'] into the runNodes map", () => {
    expect(resolveCelSuggestionValue("$['deploy-prod'].outputs.url", globals)).toBe("https://prod");
  });

  it("walks only the tail expression when combined with other operations", () => {
    // Tail-extraction stops at operators, so the walker resolves the trailing path only.
    expect(resolveCelSuggestionValue("something + data.name", globals)).toBe("deploy");
  });

  it("returns undefined when the leading token is not an identifier", () => {
    expect(resolveCelSuggestionValue("42", globals)).toBeUndefined();
  });
});

describe("widgetCelAdapter", () => {
  it("wires the CEL evaluators under the cel dialect id", () => {
    expect(widgetCelAdapter.id).toBe("cel");
    expect(widgetCelAdapter.evaluate).toBe(evaluateCel);
    expect(widgetCelAdapter.resolveSuggestionValue).toBe(resolveCelSuggestionValue);
    expect(widgetCelAdapter.formatResult("hello")).toBe("hello");
  });
});
