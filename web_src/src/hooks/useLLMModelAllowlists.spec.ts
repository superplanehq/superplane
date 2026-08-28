import { describe, expect, it } from "vitest";

import { factoryLLMModelsQueryKey, isFactoryBYOKModelsQuery } from "./useLLMModelAllowlists";

describe("isFactoryBYOKModelsQuery", () => {
  it("matches factory BYOK model list keys for the organization and provider", () => {
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-1", "factory-1", "anthropic", "byok"),
        "org-1",
        "anthropic",
      ),
    ).toBe(true);
  });

  it("rejects hosted factory keys and other organizations", () => {
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-1", "factory-1", "anthropic", "hosted"),
        "org-1",
        "anthropic",
      ),
    ).toBe(false);
    expect(
      isFactoryBYOKModelsQuery(
        factoryLLMModelsQueryKey("org-2", "factory-1", "anthropic", "byok"),
        "org-1",
        "anthropic",
      ),
    ).toBe(false);
  });
});
