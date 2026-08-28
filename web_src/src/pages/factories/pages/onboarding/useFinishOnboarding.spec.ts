import { describe, expect, it } from "vitest";

import { afterOnboardingPath, finishOnboardingError } from "./useFinishOnboarding";

const readyPlan = {
  providerId: "openrouter",
  component: "runnerOpenRouter",
  credentialsSource: "hosted",
  integrationName: "openrouter",
  harness: "AGENT_HARNESS_CLAUDE_CODE",
  model: "openai/gpt-4.1",
  planningModel: "openai/gpt-4.1",
} as const;

describe("finishOnboardingError", () => {
  it("allows finish when GitHub, repositories, name, and an agent plan are ready", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: "acme/web",
        workspaceName: "Web",
        githubReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
      }),
    ).toBeNull();
  });
});

describe("afterOnboardingPath", () => {
  it("opens the intake drawer on the provisioned line", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
      }),
    ).toBe("/org-1/workspaces/SP/lines/line-1?intake=1");
  });

  it("names the GitHub intake on the line URL", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
        githubIntakeId: "intake-github",
      }),
    ).toBe("/org-1/workspaces/SP/lines/line-1?intake=1&intakeId=intake-github");
  });
});
