import { describe, expect, it } from "vitest";

import { compareModelLabels, filterModelIds, uniqueSortedModelIds } from "./hostedLLMModels";

describe("uniqueSortedModelIds", () => {
  it("sorts model ids by name and drops blanks and duplicates", () => {
    expect(uniqueSortedModelIds(["z-model", " a-model ", "m-model", "a-model", "", "  "])).toEqual([
      "a-model",
      "m-model",
      "z-model",
    ]);
  });

  it("sorts numeric model names in natural order", () => {
    expect(uniqueSortedModelIds(["gpt-4", "gpt-10", "gpt-2"])).toEqual(["gpt-2", "gpt-4", "gpt-10"]);
  });
});

describe("filterModelIds", () => {
  it("keeps models whose names contain the query", () => {
    expect(filterModelIds(["anthropic/claude-sonnet-4", "openai/gpt-4.1", "google/gemini-2.5"], "GPT")).toEqual([
      "openai/gpt-4.1",
    ]);
  });

  it("returns all models when the query is blank", () => {
    const ids = ["b", "a"];
    expect(filterModelIds(ids, "   ")).toEqual(ids);
  });
});

describe("compareModelLabels", () => {
  it("compares labels without regard to case", () => {
    expect(compareModelLabels("Claude", "claude")).toBe(0);
    expect(compareModelLabels("anthropic/a", "OpenAI/b")).toBeLessThan(0);
  });
});
