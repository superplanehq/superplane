import { describe, expect, it } from "vitest";

import { hostedDefaultModelOptions } from "./hostedLLMDefaultModel";
import type { HostedLLMProvider, ProviderForm } from "./hostedLLMSettingsApi";

const openRouter = (overrides: Partial<HostedLLMProvider> = {}): HostedLLMProvider => ({
  provider: "openrouter",
  enabled: false,
  api_key_configured: true,
  base_url: "",
  allowed_models: ["openai/gpt-4.1", "anthropic/claude-sonnet-4"],
  ...overrides,
});

const emptyForm = (overrides: Partial<ProviderForm> = {}): ProviderForm => ({
  enabled: false,
  apiKey: "",
  baseURL: "",
  allowedModels: [],
  listedModels: [],
  ...overrides,
});

describe("hostedDefaultModelOptions", () => {
  it("lists allowlisted OpenRouter models when the provider switch is off", () => {
    expect(hostedDefaultModelOptions([openRouter()])).toEqual([
      { value: "openrouter::openai/gpt-4.1", label: "OpenRouter - openai/gpt-4.1" },
      { value: "openrouter::anthropic/claude-sonnet-4", label: "OpenRouter - anthropic/claude-sonnet-4" },
    ]);
  });

  it("omits a provider that has no API key", () => {
    expect(hostedDefaultModelOptions([openRouter({ api_key_configured: false })])).toEqual([]);
  });

  it("lists models from the live form before the allowlist is saved", () => {
    expect(
      hostedDefaultModelOptions([openRouter({ allowed_models: [] })], {
        openrouter: emptyForm({
          apiKey: "sk-or-live",
          allowedModels: ["moonshotai/kimi-k2.6"],
        }),
      }),
    ).toEqual([{ value: "openrouter::moonshotai/kimi-k2.6", label: "OpenRouter - moonshotai/kimi-k2.6" }]);
  });
});
