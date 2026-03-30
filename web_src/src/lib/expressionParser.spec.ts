import { describe, expect, it } from "vitest";
import { evaluateIndividualComparisons, parseExpression, substituteExpressionValues } from "@/lib/expressionParser";

describe("expressionParser", () => {
  it("parses expressions into styled badge groups", () => {
    expect(parseExpression('$.status == "active" and $.count >= 3')).toEqual([
      {
        badges: [
          { label: "$.status", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: "==", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: '"active"', bgColor: "bg-green-100", textColor: "text-green-700" },
          { label: "and", bgColor: "bg-gray-500", textColor: "text-white" },
        ],
      },
      {
        badges: [
          { label: "$.count", bgColor: "bg-purple-100", textColor: "text-purple-700" },
          { label: ">=", bgColor: "bg-gray-100", textColor: "text-gray-700" },
          { label: "3", bgColor: "bg-green-100", textColor: "text-green-700" },
        ],
      },
    ]);
  });

  it("substitutes root, previous, and current values into expressions", () => {
    expect(
      substituteExpressionValues(
        "$.status == root().status && $.count == previous(2).count",
        { status: "ok", count: 3 },
        {
          root: { status: "ok" },
          previousByDepth: { "2": { count: 3 } },
        },
      ),
    ).toBe('"ok" == "ok" && 3 == 3');
  });

  it("marks only the failed parts of basic comparisons", () => {
    expect(evaluateIndividualComparisons('"ok" == "ok" && 3 > 5')).toEqual(new Set(["3", ">", "5"]));
  });
});
