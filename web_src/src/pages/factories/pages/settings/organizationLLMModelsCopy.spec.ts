import { describe, expect, it } from "bun:test";

import { byokProviderProductName, providerKeyHeading, providerKeyIntegrationLink } from "./organizationLLMModelsCopy";

describe("byokProviderProductName", () => {
  it("uses Integrations product names", () => {
    expect(byokProviderProductName("anthropic")).toBe("Claude");
    expect(byokProviderProductName("openai")).toBe("OpenAI");
    expect(byokProviderProductName("openrouter")).toBe("OpenRouter");
  });
});

describe("providerKeyHeading", () => {
  it("names the key the models come from", () => {
    expect(providerKeyHeading("openrouter")).toBe("Your OpenRouter key");
    expect(providerKeyIntegrationLink("anthropic")).toBe("Open Claude integration");
  });
});
