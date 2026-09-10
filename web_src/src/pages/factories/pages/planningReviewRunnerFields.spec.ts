import { describe, expect, it } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";

import { planningReviewModelUsedField } from "./planningReviewRunnerFields";

const superPlaneCatalogField: ConfigurationField = {
  name: "model",
  label: "Model",
  type: "hosted-model",
  placeholder: "Instance SuperPlane agent model",
  typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } },
};

describe("planningReviewModelUsedField", () => {
  it("uses the catalog model field when the automation supplies one", () => {
    const field = planningReviewModelUsedField("runnerSuperPlane", [superPlaneCatalogField]);

    expect(field).toMatchObject({
      name: "model",
      label: "Model used",
      type: "hosted-model",
      typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } },
    });
  });

  it("uses the Run SuperPlane Agent hosted-model field when the catalog is empty", () => {
    const field = planningReviewModelUsedField("runnerSuperPlane");

    expect(field?.type).toBe("hosted-model");
    expect(field?.typeOptions?.hostedModel?.provider).toBe(HOSTED_MODEL_ALL_PROVIDERS);
    expect(field?.label).toBe("Model used");
    expect(field?.typeOptions?.select?.options).toBeUndefined();
  });

  it("keeps the Claude aliases when the agent is not Run SuperPlane Agent", () => {
    const field = planningReviewModelUsedField("runnerClaudeCode");

    expect(field?.type).toBe("select");
    expect(field?.typeOptions?.select?.options?.map((option) => option.label)).toEqual([
      "Claude Sonnet",
      "Claude Opus",
      "Claude Haiku",
    ]);
  });
});
