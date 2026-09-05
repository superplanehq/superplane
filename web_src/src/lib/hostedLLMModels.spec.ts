import { describe, expect, it } from "vitest";

import {
  compareModelLabels,
  filterModelIds,
  hostedLLMModelKey,
  hostedLLMTechnicalName,
  hostedLLMTechnicalNameFromKey,
  hostedModelIds,
  parseHostedLLMModelKey,
  pickHostedAnthropicModel,
  pickHostedModel,
  uniqueSortedModelIds,
} from "./hostedLLMModels";

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

describe("pickHostedAnthropicModel", () => {
  it("prefers a Sonnet id from the allowlist", () => {
    expect(pickHostedAnthropicModel(["claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"])).toBe(
      "claude-sonnet-4-6",
    );
  });

  it("uses the first allowlisted id when no Sonnet id is present", () => {
    expect(pickHostedAnthropicModel(["claude-opus-4-6", "claude-haiku-4-5"])).toBe("claude-haiku-4-5");
  });

  it("returns undefined when the allowlist is empty", () => {
    expect(pickHostedAnthropicModel([])).toBeUndefined();
  });
});

describe("pickHostedModel", () => {
  it("prefers gpt-5 from the OpenAI allowlist", () => {
    expect(pickHostedModel("openai", ["gpt-4.1", "gpt-5", "o3"])).toBe("gpt-5");
  });

  it("prefers a Sonnet id from the OpenRouter allowlist", () => {
    expect(pickHostedModel("openrouter", ["openai/gpt-4.1", "anthropic/claude-sonnet-4-6"])).toBe(
      "anthropic/claude-sonnet-4-6",
    );
  });

  it("uses the first allowlisted id when no preferred id is present", () => {
    expect(pickHostedModel("openrouter", ["openai/gpt-4.1", "google/gemini-2.5-flash"])).toBe(
      "google/gemini-2.5-flash",
    );
  });
});

describe("hostedModelIds", () => {
  it("keeps non-empty model ids", () => {
    expect(hostedModelIds([{ id: "sonnet" }, { id: "" }, { id: null }, {}, { id: "opus" }])).toEqual([
      "sonnet",
      "opus",
    ]);
  });
});

describe("hostedLLMTechnicalName", () => {
  it("uses provider/model for native Anthropic and OpenAI ids", () => {
    expect(hostedLLMTechnicalName("anthropic", "claude-sonnet-4-6")).toBe("anthropic/claude-sonnet-4-6");
    expect(hostedLLMTechnicalName("openai", "gpt-5")).toBe("openai/gpt-5");
  });

  it("keeps OpenRouter ids as provider/model", () => {
    expect(hostedLLMTechnicalName("openrouter", "moonshotai/kimi-k2.6")).toBe("moonshotai/kimi-k2.6");
  });
});

describe("hostedLLMModelKey", () => {
  it("encodes and parses provider::model keys", () => {
    expect(hostedLLMModelKey("openrouter", "anthropic/claude-sonnet-4")).toBe("openrouter::anthropic/claude-sonnet-4");
    expect(parseHostedLLMModelKey("openrouter::anthropic/claude-sonnet-4")).toEqual({
      provider: "openrouter",
      model: "anthropic/claude-sonnet-4",
    });
    expect(hostedLLMTechnicalNameFromKey("anthropic::claude-sonnet-4-6")).toBe("anthropic/claude-sonnet-4-6");
  });
});
