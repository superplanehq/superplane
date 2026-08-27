import { describe, expect, it } from "vitest";

import type { IntegrationId } from "./onboardingFixtures";
import {
  firstWorkOrderAgentError,
  hostedCreditGrantCopy,
  hostedModelsQueriesLoading,
  isAgentStepReady,
  resolveOnboardingAgent,
  shouldShowHostedCreditGrant,
} from "./onboardingAgentReadiness";

function connected(...ids: IntegrationId[]): Set<IntegrationId> {
  return new Set(ids);
}

const noHostedModels = {
  anthropic: [],
  openai: [],
  openrouter: [],
};

describe("isAgentStepReady", () => {
  it("is ready when SuperPlane agent has remaining hosted credit", () => {
    expect(isAgentStepReady(connected(), 4124, null)).toBe(true);
    expect(isAgentStepReady(connected("claude"), 4124, null)).toBe(true);
  });

  it("is not ready when SuperPlane agent has empty credit", () => {
    expect(isAgentStepReady(connected(), 0, null)).toBe(false);
    expect(isAgentStepReady(connected("claude"), 0, null)).toBe(false);
  });

  it("is ready for BYOK only when the selected provider is connected", () => {
    expect(isAgentStepReady(connected("claude"), 0, "claude")).toBe(true);
    expect(isAgentStepReady(connected("openai"), 0, "openai")).toBe(true);
    expect(isAgentStepReady(connected("openrouter"), 0, "openrouter")).toBe(true);
    expect(isAgentStepReady(connected("openai"), 0, "claude")).toBe(false);
    expect(isAgentStepReady(connected(), 5000, "claude")).toBe(false);
  });
});

describe("resolveOnboardingAgent", () => {
  it("uses hosted OpenRouter when SuperPlane agent is selected", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("claude"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["openai/gpt-4.1", "anthropic/claude-sonnet-4-6"] },
        keyProvider: null,
      }),
    ).toEqual({
      providerId: "openrouter",
      component: "runnerOpenRouter",
      credentialsSource: "hosted",
      integrationName: "openrouter",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "anthropic/claude-sonnet-4-6",
    });
  });

  it("does not let a connected Claude key steal the SuperPlane agent default", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("claude", "openai"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["anthropic/claude-sonnet-4-6"] },
        keyProvider: null,
      })?.providerId,
    ).toBe("openrouter");
  });

  it("uses a connected OpenRouter integration only after BYOK selection", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("openrouter"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["openai/gpt-4.1", "anthropic/claude-sonnet-4-6"] },
        keyProvider: "openrouter",
      }),
    ).toEqual({
      providerId: "openrouter",
      component: "runnerOpenRouter",
      credentialsSource: "integration",
      integrationName: "openrouter",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "anthropic/claude-sonnet-4-6",
    });
  });

  it("uses the selected BYOK Claude runner and installation", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("claude"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["anthropic/claude-sonnet-4-6"] },
        keyProvider: "claude",
      }),
    ).toEqual({
      providerId: "claude",
      component: "runnerClaudeCode",
      credentialsSource: "integration",
      integrationName: "claude",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "sonnet",
    });
  });

  it("uses the selected BYOK OpenAI runner", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("openai"),
        remainingCreditCents: 0,
        hostedModels: noHostedModels,
        keyProvider: "openai",
      })?.component,
    ).toBe("runnerCodex");
  });

  it("returns undefined when SuperPlane agent has credit but OpenRouter is not allowlisted", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected(),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openai: ["gpt-5"] },
        keyProvider: null,
      }),
    ).toBeUndefined();
  });

  it("returns undefined when credit remains but no hosted models are enabled", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected(),
        remainingCreditCents: 5000,
        hostedModels: noHostedModels,
        keyProvider: null,
      }),
    ).toBeUndefined();
  });
});

describe("hostedModelsQueriesLoading", () => {
  it("waits until every hosted model query has fetched when credit remains", () => {
    expect(hostedModelsQueriesLoading(true, [{ isFetched: true }, { isFetched: false }])).toBe(true);
    expect(hostedModelsQueriesLoading(true, [{ isFetched: true }, { isFetched: true }])).toBe(false);
  });

  it("does not wait when hosted models are not required", () => {
    expect(hostedModelsQueriesLoading(false, [{ isFetched: false }])).toBe(false);
  });
});

describe("firstWorkOrderAgentError", () => {
  it("asks the user to use SuperPlane agent credit or a key when credit is empty", () => {
    expect(
      firstWorkOrderAgentError({
        remainingCreditCents: 0,
        hostedModelsLoading: false,
        plan: undefined,
      }),
    ).toBe("Use SuperPlane agent credit, or connect Anthropic, OpenAI, or OpenRouter.");
  });

  it("asks the user to wait when hosted models are still loading", () => {
    expect(
      firstWorkOrderAgentError({
        remainingCreditCents: 5000,
        hostedModelsLoading: true,
        plan: undefined,
      }),
    ).toBe("Hosted models are still loading. Try again.");
  });

  it("asks an admin to enable hosted models when credit remains without an allowlist", () => {
    expect(
      firstWorkOrderAgentError({
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: undefined,
      }),
    ).toBe("Ask an installation admin to enable SuperPlane-hosted models.");
  });

  it("returns null when a plan is ready", () => {
    expect(
      firstWorkOrderAgentError({
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: {
          providerId: "openrouter",
          component: "runnerOpenRouter",
          credentialsSource: "hosted",
          integrationName: "openrouter",
          harness: "AGENT_HARNESS_CLAUDE_CODE",
          model: "openai/gpt-4.1",
        },
      }),
    ).toBeNull();
  });
});

describe("hosted credit grant copy", () => {
  it("hides the grant block when the organization has no grant", () => {
    expect(shouldShowHostedCreditGrant(0)).toBe(false);
  });

  it("shows the grant block when a grant exists, even if remaining credit is empty", () => {
    expect(shouldShowHostedCreditGrant(5000)).toBe(true);
  });

  it("explains that remaining credit lets SuperPlane run the agent", () => {
    expect(hostedCreditGrantCopy(5000)).toBe(
      "This organization has $50.00 of SuperPlane agent credit. SuperPlane can run the agent without your own key.",
    );
  });

  it("asks the user to connect a provider when remaining credit is empty", () => {
    expect(hostedCreditGrantCopy(0)).toBe("SuperPlane agent credit is empty. Connect a provider to continue.");
  });
});
