import { describe, expect, it } from "vitest";

import type { IntegrationId } from "./onboardingFixtures";
import {
  firstWorkOrderAgentError,
  hostedCreditGrantCopy,
  hostedModelsQueriesLoading,
  isAgentStepReady,
  isHostedAgentReady,
  resolveOnboardingAgent,
  shouldShowHostedCreditGrant,
  type OnboardingAgentPlan,
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
  it("is ready when remaining hosted credit is greater than zero", () => {
    expect(isAgentStepReady(connected(), 4124)).toBe(true);
  });

  it("is ready when Anthropic, OpenAI, or OpenRouter is connected", () => {
    expect(isAgentStepReady(connected("claude"), 0)).toBe(true);
    expect(isAgentStepReady(connected("openai"), 0)).toBe(true);
    expect(isAgentStepReady(connected("openrouter"), 0)).toBe(true);
  });

  it("is not ready when credit is empty and no provider is connected", () => {
    expect(isAgentStepReady(connected(), 0)).toBe(false);
    expect(isAgentStepReady(connected("github"), 0)).toBe(false);
  });
});

describe("resolveOnboardingAgent", () => {
  it("uses a connected OpenRouter integration and an allowlisted model", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("openrouter"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["openai/gpt-4.1", "anthropic/claude-sonnet-4-6"] },
      }),
    ).toEqual({
      providerId: "openrouter",
      component: "runnerOpenRouter",
      credentialsSource: "integration",
      integrationName: "openrouter",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "anthropic/claude-sonnet-4-6",
      planningModel: "anthropic/claude-sonnet-4-6",
    });
  });

  it("gives planning an Opus id when the allowlist has one", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("openrouter"),
        remainingCreditCents: 5000,
        hostedModels: {
          ...noHostedModels,
          openrouter: ["anthropic/claude-opus-4-6", "anthropic/claude-sonnet-4-6"],
        },
      }),
    ).toMatchObject({
      model: "anthropic/claude-sonnet-4-6",
      planningModel: "anthropic/claude-opus-4-6",
    });
  });

  // With no allowlist to read, the agent CLI resolves the alias itself.
  it("gives planning the Opus alias when no allowlist applies", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("claude"),
        remainingCreditCents: 0,
        hostedModels: noHostedModels,
      }),
    ).toMatchObject({
      credentialsSource: "integration",
      model: "sonnet",
      planningModel: "opus",
    });
  });

  it("uses hosted OpenRouter and the selected allowlist when only credit remains", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected(),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["openai/gpt-4.1"] },
      }),
    ).toEqual({
      providerId: "openrouter",
      component: "runnerOpenRouter",
      credentialsSource: "hosted",
      integrationName: "openrouter",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "openai/gpt-4.1",
      planningModel: "openai/gpt-4.1",
    });
  });

  it("uses hosted OpenAI when that provider is the enabled hosted allowlist", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected(),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openai: ["gpt-5", "gpt-4.1"] },
      }),
    ).toEqual({
      providerId: "openai",
      component: "runnerCodex",
      credentialsSource: "hosted",
      integrationName: "openai",
      harness: "AGENT_HARNESS_CODEX",
      model: "gpt-5",
      planningModel: "gpt-5",
    });
  });

  it("prefers a connected provider over hosted credit", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected("openai"),
        remainingCreditCents: 5000,
        hostedModels: { ...noHostedModels, openrouter: ["anthropic/claude-sonnet-4-6"] },
      })?.providerId,
    ).toBe("openai");
  });

  it("returns undefined when credit remains but no hosted models are enabled", () => {
    expect(
      resolveOnboardingAgent({
        connected: connected(),
        remainingCreditCents: 5000,
        hostedModels: noHostedModels,
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

describe("isHostedAgentReady", () => {
  function plan(credentialsSource: OnboardingAgentPlan["credentialsSource"]): OnboardingAgentPlan {
    return {
      providerId: "openrouter",
      component: "runnerOpenRouter",
      credentialsSource,
      integrationName: "openrouter",
      harness: "AGENT_HARNESS_CLAUDE_CODE",
      model: "openai/gpt-4.1",
      planningModel: "openai/gpt-4.1",
    };
  }

  it("is ready when the plan runs on hosted credentials", () => {
    expect(isHostedAgentReady({ hostedModelsLoading: false, plan: plan("hosted") })).toBe(true);
  });

  it("is not ready when the plan needs a connected provider", () => {
    expect(isHostedAgentReady({ hostedModelsLoading: false, plan: plan("integration") })).toBe(false);
  });

  it("is not ready without a plan", () => {
    expect(isHostedAgentReady({ hostedModelsLoading: false, plan: undefined })).toBe(false);
  });

  it("is not ready while hosted models load, because the plan can still change", () => {
    expect(isHostedAgentReady({ hostedModelsLoading: true, plan: plan("hosted") })).toBe(false);
  });
});

describe("firstWorkOrderAgentError", () => {
  it("asks the user to connect a provider when credit is empty", () => {
    expect(
      firstWorkOrderAgentError({
        remainingCreditCents: 0,
        hostedModelsLoading: false,
        plan: undefined,
      }),
    ).toBe("Connect Anthropic, OpenAI, or OpenRouter, or use hosted credit.");
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
          planningModel: "openai/gpt-4.1",
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

  it("explains that remaining credit lets the user continue without keys", () => {
    expect(hostedCreditGrantCopy(5000)).toBe(
      "This organization has $50.00 of hosted credit. You can continue without connecting your own keys.",
    );
  });

  it("asks the user to connect a provider when remaining credit is empty", () => {
    expect(hostedCreditGrantCopy(0)).toBe("Hosted credit is empty. Connect a provider to continue.");
  });
});
