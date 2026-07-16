import { describe, expect, it } from "vitest";

import { DOLLAR_REWRITE_IDENTIFIER } from "./celExpr";
import { withCatalogRunNodesFallback } from "./useWidgetExpressionContext";

const runFields = [{ field: "$", types: ["object"], sample: "$" }];

describe("withCatalogRunNodesFallback", () => {
  it("replaces an empty live run-node map with catalog hints", () => {
    const row = withCatalogRunNodesFallback({ $: {}, [DOLLAR_REWRITE_IDENTIFIER]: {} }, runFields);

    expect(Object.keys(row[DOLLAR_REWRITE_IDENTIFIER] as Record<string, unknown>)).toEqual(["example-node"]);
    expect(row.$).toBe(row[DOLLAR_REWRITE_IDENTIFIER]);
  });

  it("preserves populated live run-node hints", () => {
    const runNodes = { Deploy: { outputs: {} } };
    const row = withCatalogRunNodesFallback({ $: runNodes, [DOLLAR_REWRITE_IDENTIFIER]: runNodes }, runFields);

    expect(row.$).toBe(runNodes);
    expect(row[DOLLAR_REWRITE_IDENTIFIER]).toBe(runNodes);
  });
});
