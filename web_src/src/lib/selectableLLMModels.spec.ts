import { describe, expect, it } from "bun:test";

import {
  hostedSelectableLLMModelKey,
  normalizeSuperPlaneModelValue,
  parseSelectableLLMModelKey,
  resolveSelectableLLMModelKey,
  selectableLLMModelKey,
  selectableLLMModelLabel,
  selectableLLMModelLabelFromKey,
  selectableLLMModelsFromResponse,
  selectableLLMModelsForProvider,
  byokRunnerModelOptions,
  defaultByokRunnerModel,
  sortSelectableLLMModels,
  type SelectableLLMModel,
} from "./selectableLLMModels";

function model(partial: Partial<SelectableLLMModel> & Pick<SelectableLLMModel, "key" | "label">): SelectableLLMModel {
  return {
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: "anthropic", name: "Anthropic" },
    model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
    ...partial,
  };
}

describe("selectableLLMModelKey", () => {
  it("encodes and parses source::provider::model", () => {
    expect(selectableLLMModelKey("hosted", "anthropic", "claude-sonnet-4-6")).toBe(
      "hosted::anthropic::claude-sonnet-4-6",
    );
    expect(parseSelectableLLMModelKey("byok::openrouter::moonshotai/kimi-k2.6")).toEqual({
      source: "byok",
      provider: "openrouter",
      model: "moonshotai/kimi-k2.6",
    });
    expect(hostedSelectableLLMModelKey("openai", "gpt-5")).toBe("hosted::openai::gpt-5");
  });
});

describe("selectableLLMModelLabel", () => {
  it("uses provider/model for native ids", () => {
    expect(selectableLLMModelLabel("anthropic", "claude-sonnet-4-6")).toBe("anthropic/claude-sonnet-4-6");
    expect(selectableLLMModelLabelFromKey("hosted::openrouter::moonshotai/kimi-k2.6")).toBe("moonshotai/kimi-k2.6");
  });
});

describe("selectableLLMModelsForProvider", () => {
  it("keeps only the requested provider", () => {
    const listed = [
      model({
        key: "byok::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
        source: { id: "byok", name: "Your keys" },
      }),
      model({
        key: "byok::openai::gpt-5",
        label: "openai/gpt-5",
        source: { id: "byok", name: "Your keys" },
        provider: { id: "openai", name: "OpenAI" },
        model: { id: "gpt-5", name: "gpt-5" },
      }),
    ];
    expect(selectableLLMModelsForProvider(listed, "anthropic").map((item) => item.model.id)).toEqual([
      "claude-sonnet-4-6",
    ]);
  });
});

describe("defaultByokRunnerModel", () => {
  const allowlist = ["claude-opus-4-6", "claude-sonnet-4-6", "claude-haiku-4-5"];

  it("replaces a Claude alias or the previous default with Opus 5.5", () => {
    const withOpus55 = [...allowlist, "claude-opus-5-5"];
    expect(defaultByokRunnerModel("sonnet", "anthropic", withOpus55)).toBe("claude-opus-5-5");
    expect(defaultByokRunnerModel("opus", "anthropic", withOpus55)).toBe("claude-opus-5-5");
    expect(defaultByokRunnerModel("claude-sonnet-4-6", "anthropic", withOpus55)).toBe("claude-opus-5-5");
    expect(defaultByokRunnerModel("claude-opus-4-6", "anthropic", withOpus55)).toBe("claude-opus-5-5");
  });

  it("prefers Claude Opus 5.5 when no model is selected", () => {
    expect(defaultByokRunnerModel("", "anthropic", [...allowlist, "claude-opus-5-5"])).toBe("claude-opus-5-5");
  });

  it("prefers a Sonnet id when the organization key has no Opus 5.5 model", () => {
    expect(defaultByokRunnerModel("sonnet", "anthropic", allowlist)).toBe("claude-sonnet-4-6");
    expect(defaultByokRunnerModel("", "anthropic", allowlist)).toBe("claude-sonnet-4-6");
  });

  it("uses the first allowlisted id when the key has no Sonnet model", () => {
    expect(defaultByokRunnerModel("sonnet", "anthropic", ["claude-opus-4-6", "claude-haiku-4-5"])).toBe(
      "claude-sonnet-4-6",
    );
  });

  it("keeps an allowlisted id and any other explicit id", () => {
    expect(defaultByokRunnerModel("claude-opus-4-6", "anthropic", allowlist)).toBeUndefined();
    expect(defaultByokRunnerModel("claude-custom", "anthropic", allowlist)).toBeUndefined();
  });

  it("uses Opus 5.5 when the organization key has no models", () => {
    expect(defaultByokRunnerModel("sonnet", "anthropic", [])).toBe("claude-opus-5-5");
  });
});

describe("byokRunnerModelOptions", () => {
  it("stores the model id and keeps a value that is not on the list", () => {
    const listed = [
      model({
        key: "byok::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
        source: { id: "byok", name: "Your keys" },
      }),
    ];
    expect(byokRunnerModelOptions(listed, "")).toEqual([
      { value: "claude-sonnet-4-6", label: "anthropic/claude-sonnet-4-6" },
    ]);
    expect(byokRunnerModelOptions(listed, "opus")).toEqual([
      { value: "claude-opus-5-5", label: "claude-opus-5-5" },
      { value: "claude-sonnet-4-6", label: "anthropic/claude-sonnet-4-6" },
    ]);
  });
});

describe("selectableLLMModelsFromResponse", () => {
  it("drops incomplete rows and filters by source", () => {
    const listed = selectableLLMModelsFromResponse(
      [
        {
          source: { id: "hosted", name: "SuperPlane" },
          provider: { id: "anthropic", name: "Anthropic" },
          model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
          key: "hosted::anthropic::claude-sonnet-4-6",
          label: "anthropic/claude-sonnet-4-6",
        },
        {
          source: { id: "byok", name: "Your keys" },
          provider: { id: "anthropic", name: "Anthropic" },
          model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
          key: "byok::anthropic::claude-sonnet-4-6",
          label: "anthropic/claude-sonnet-4-6",
        },
        { key: "" },
      ],
      ["hosted"],
    );
    expect(listed.map((item) => item.key)).toEqual(["hosted::anthropic::claude-sonnet-4-6"]);
  });
});

describe("resolveSelectableLLMModelKey", () => {
  it("prefers the session key, then the instance SuperPlane agent model", () => {
    const listed = [
      model({
        key: "hosted::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
      }),
      model({
        key: "hosted::openai::gpt-5",
        label: "openai/gpt-5",
        provider: { id: "openai", name: "OpenAI" },
        model: { id: "gpt-5", name: "gpt-5" },
      }),
    ];
    expect(resolveSelectableLLMModelKey("hosted::openai::gpt-5", listed, "hosted::anthropic::claude-sonnet-4-6")).toBe(
      "hosted::openai::gpt-5",
    );
    expect(resolveSelectableLLMModelKey("", listed, "hosted::anthropic::claude-sonnet-4-6")).toBe(
      "hosted::anthropic::claude-sonnet-4-6",
    );
  });
});

describe("normalizeSuperPlaneModelValue", () => {
  it("upgrades two-part SuperPlane keys to hosted three-part keys", () => {
    expect(normalizeSuperPlaneModelValue("anthropic::claude-sonnet-4-6")).toBe("hosted::anthropic::claude-sonnet-4-6");
    expect(normalizeSuperPlaneModelValue("hosted::openai::gpt-5")).toBe("hosted::openai::gpt-5");
  });
});

describe("sortSelectableLLMModels", () => {
  it("keeps duplicate labels and sorts by source after label", () => {
    const listed = sortSelectableLLMModels([
      model({
        key: "hosted::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
        source: { id: "hosted", name: "SuperPlane" },
      }),
      model({
        key: "byok::anthropic::claude-sonnet-4-6",
        label: "anthropic/claude-sonnet-4-6",
        source: { id: "byok", name: "Your keys" },
      }),
    ]);
    expect(listed.map((item) => item.key)).toEqual([
      "byok::anthropic::claude-sonnet-4-6",
      "hosted::anthropic::claude-sonnet-4-6",
    ]);
  });
});
