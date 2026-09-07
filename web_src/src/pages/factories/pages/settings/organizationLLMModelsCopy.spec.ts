import { describe, expect, it } from "vitest";

import {
  byokProviderProductName,
  disconnectedProviderMessage,
  shouldShowLLMModelsLandingBanner,
} from "./organizationLLMModelsCopy";

describe("byokProviderProductName", () => {
  it("uses Integrations product names", () => {
    expect(byokProviderProductName("anthropic")).toBe("Claude");
    expect(byokProviderProductName("openai")).toBe("OpenAI");
    expect(byokProviderProductName("openrouter")).toBe("OpenRouter");
  });
});

describe("disconnectedProviderMessage", () => {
  it("tells the user to connect the provider on Integrations", () => {
    expect(disconnectedProviderMessage("openrouter")).toBe("Connect OpenRouter on Integrations, then select models.");
  });
});

describe("shouldShowLLMModelsLandingBanner", () => {
  it("hides the banner while any provider is loading", () => {
    expect(
      shouldShowLLMModelsLandingBanner([
        { isLoading: true, error: null, data: undefined },
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: false } },
      ]),
    ).toBe(false);
  });

  it("hides the banner when a provider is connected", () => {
    expect(
      shouldShowLLMModelsLandingBanner([
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: true } },
      ]),
    ).toBe(false);
  });

  it("hides the banner when a connected catalog fails to load", () => {
    expect(
      shouldShowLLMModelsLandingBanner([
        { isLoading: false, error: new Error("list failed"), data: undefined },
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: false } },
      ]),
    ).toBe(false);
  });

  it("shows the banner when every provider is disconnected", () => {
    expect(
      shouldShowLLMModelsLandingBanner([
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: false } },
        { isLoading: false, error: null, data: { connected: false } },
      ]),
    ).toBe(true);
  });
});
