import { describe, expect, it } from "vitest";
import {
  EXPRESSION_DOUBLE_BRACE_TIP,
  EXPRESSION_PAYLOAD_TIP,
  expressionQuickTipForField,
  fieldShowsExpressionQuickTip,
  resolveExpressionQuickTip,
} from "./expressionQuickTip";

describe("fieldShowsExpressionQuickTip", () => {
  it("is true for expression-capable field types when expressions are allowed", () => {
    expect(fieldShowsExpressionQuickTip({ type: "string" }, true)).toBe(true);
    expect(fieldShowsExpressionQuickTip({ type: "text" }, true)).toBe(true);
    expect(fieldShowsExpressionQuickTip({ type: "expression" }, true)).toBe(true);
    expect(fieldShowsExpressionQuickTip({ type: "any-predicate-list" }, true)).toBe(true);
    expect(fieldShowsExpressionQuickTip({ type: "integration-resource" }, true)).toBe(true);
  });

  it("is false when expressions are disabled or the field type has no tip", () => {
    expect(fieldShowsExpressionQuickTip({ type: "string" }, false)).toBe(false);
    expect(fieldShowsExpressionQuickTip({ type: "boolean" }, true)).toBe(false);
    expect(fieldShowsExpressionQuickTip({ type: "number" }, true)).toBe(false);
  });
});

describe("resolveExpressionQuickTip", () => {
  it("returns the double-brace tip for string fields without a description", () => {
    expect(resolveExpressionQuickTip({ type: "string" }, true)).toBe(EXPRESSION_DOUBLE_BRACE_TIP);
  });

  it("returns the payload tip for expression fields without a description", () => {
    expect(resolveExpressionQuickTip({ type: "expression" }, true)).toBe(EXPRESSION_PAYLOAD_TIP);
  });

  it("returns null when the field already has a description", () => {
    expect(
      resolveExpressionQuickTip({ type: "string", description: "Choose a repository." }, true),
    ).toBeNull();
  });

  it("returns undefined when the field cannot show an expression tip", () => {
    expect(resolveExpressionQuickTip({ type: "boolean" }, true)).toBeUndefined();
    expect(resolveExpressionQuickTip({ type: "string" }, false)).toBeUndefined();
  });
});

describe("expressionQuickTipForField", () => {
  it("picks the tip text by field type", () => {
    expect(expressionQuickTipForField({ type: "expression" })).toBe(EXPRESSION_PAYLOAD_TIP);
    expect(expressionQuickTipForField({ type: "string" })).toBe(EXPRESSION_DOUBLE_BRACE_TIP);
  });
});
