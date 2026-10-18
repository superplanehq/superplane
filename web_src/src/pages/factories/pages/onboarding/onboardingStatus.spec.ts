import { describe, expect, it } from "bun:test";

import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";
import { syncSelectionsWithInstances, type IntegrationInstanceSummary } from "@/pages/home/homeIntegrationStatus";

import {
  apiIssuesSource,
  initialOnboardingSelections,
  initialWizardStep,
  isFactoryOnboardingComplete,
  localIssuesSource,
  ONBOARDING_CONNECTION_NAMES,
  ONBOARDING_MANUAL_CONNECTION_NAMES,
  restoreSavedAgentSelection,
} from "./onboardingStatus";

function githubInstance(id: string, name: string): OrganizationsIntegration {
  return {
    metadata: { id, name, integrationName: "github" },
    status: { state: "ready" },
  };
}

function connectionData(instances: OrganizationsIntegration[], provider = "github"): IntegrationInstanceSummary[] {
  return ONBOARDING_CONNECTION_NAMES.map((name) => ({
    name,
    allInstances: name === provider ? instances : [],
    readyInstances: name === provider ? instances : [],
  }));
}

describe("isFactoryOnboardingComplete", () => {
  it("returns false while completion time is absent", () => {
    expect(isFactoryOnboardingComplete({ onboarding: {} } as FactoriesFactory)).toBe(false);
  });

  it("returns true when onboarding is complete", () => {
    expect(
      isFactoryOnboardingComplete({
        onboarding: { completedAt: "2026-08-17T12:00:00Z" },
      } as FactoriesFactory),
    ).toBe(true);
  });
});

describe("initialWizardStep", () => {
  it("starts a new workspace at the VCS step", () => {
    expect(initialWizardStep({})).toBe("vcs");
  });

  it("treats the enum zero values the API sends as unanswered", () => {
    expect(
      initialWizardStep({
        vcsIntegrationId: "",
        appRepository: "",
        issuesSource: "ISSUES_SOURCE_UNSPECIFIED",
        agentHarness: "AGENT_HARNESS_UNSPECIFIED",
      }),
    ).toBe("vcs");
  });

  it("resumes at the first unanswered step", () => {
    expect(initialWizardStep({ vcsIntegrationId: "github-1" })).toBe("repo");
    expect(initialWizardStep({ vcsIntegrationId: "github-1", appRepository: "acme/web" })).toBe("issues");
    expect(
      initialWizardStep({
        vcsIntegrationId: "github-1",
        appRepository: "acme/web",
        issuesSource: "ISSUES_SOURCE_VCS",
      }),
    ).toBe("agent");
    expect(
      initialWizardStep({
        vcsIntegrationId: "github-1",
        appRepository: "acme/web",
        issuesSource: "ISSUES_SOURCE_VCS",
        agentHarness: "AGENT_HARNESS_CLAUDE_CODE",
      }),
    ).toBe("name");
  });
});

describe("issue source mapping", () => {
  it("maps API values to wizard choices", () => {
    expect(localIssuesSource("ISSUES_SOURCE_VCS")).toBe("vcs");
    expect(localIssuesSource("ISSUES_SOURCE_SKIP")).toBe("skip");
    expect(localIssuesSource("ISSUES_SOURCE_UNSPECIFIED")).toBeNull();
  });

  it("maps wizard choices back to API values", () => {
    expect(apiIssuesSource("vcs")).toBe("ISSUES_SOURCE_VCS");
    expect(apiIssuesSource(null)).toBe("ISSUES_SOURCE_UNSPECIFIED");
  });
});

describe("initialOnboardingSelections", () => {
  it("fills github and claude from saved integration IDs", () => {
    expect(
      initialOnboardingSelections({
        vcsIntegrationId: "github-1",
        agentIntegrationId: "claude-1",
      }),
    ).toEqual({
      github: { id: "github-1", name: "github-1", ready: false },
      claude: { id: "claude-1", name: "claude-1", ready: false },
    });
  });

  it("fills bitbucket when the saved provider is Bitbucket", () => {
    expect(
      initialOnboardingSelections({
        vcsIntegrationId: "bitbucket-1",
        vcsProvider: "bitbucket",
      }),
    ).toEqual({
      bitbucket: { id: "bitbucket-1", name: "bitbucket-1", ready: false },
    });
  });

  it.each(["github", "claude"] as const)("restores the saved %s installation after a reload", (provider) => {
    const saved = initialOnboardingSelections(
      provider === "github" ? { vcsIntegrationId: "saved-key" } : { agentIntegrationId: "saved-key" },
    );
    const synced = syncSelectionsWithInstances(
      connectionData(
        [
          { metadata: { id: "other-key", name: "other-key", integrationName: provider }, status: { state: "ready" } },
          { metadata: { id: "saved-key", name: "my-key", integrationName: provider }, status: { state: "ready" } },
        ],
        provider,
      ),
      saved,
      {},
      ONBOARDING_MANUAL_CONNECTION_NAMES,
    );

    expect(synced).toEqual({
      [provider]: { id: "saved-key", name: "my-key", ready: true },
    });
  });

  it("restores a saved OpenAI integration under OpenAI", () => {
    const saved = initialOnboardingSelections({
      agentIntegrationId: "saved-key",
      agentHarness: "AGENT_HARNESS_CODEX",
    });
    const synced = syncSelectionsWithInstances(
      connectionData(
        [{ metadata: { id: "saved-key", name: "my-openai", integrationName: "openai" }, status: { state: "ready" } }],
        "openai",
      ),
      saved,
      {},
      ONBOARDING_MANUAL_CONNECTION_NAMES,
    );

    expect(synced).toEqual({
      openai: { id: "saved-key", name: "my-openai", ready: true },
    });
  });

  it("restores a saved OpenRouter integration after the connection sync drops the wrong provider", () => {
    const saved = initialOnboardingSelections({
      agentIntegrationId: "saved-key",
      agentHarness: "AGENT_HARNESS_CLAUDE_CODE",
    });
    const instances = connectionData(
      [{ metadata: { id: "saved-key", name: "my-router", integrationName: "openrouter" }, status: { state: "ready" } }],
      "openrouter",
    );
    const synced = syncSelectionsWithInstances(instances, saved, {}, ONBOARDING_MANUAL_CONNECTION_NAMES) ?? saved;

    expect(saved.claude).toBeUndefined();
    expect(
      restoreSavedAgentSelection(
        synced,
        instances.flatMap((item) => item.allInstances),
        "saved-key",
      ),
    ).toEqual({
      openrouter: { id: "saved-key", name: "my-router", ready: true },
    });
  });

  it("does not replace a different agent the user already selected", () => {
    expect(
      restoreSavedAgentSelection(
        { openai: { id: "new-key", name: "new-openai", ready: true } },
        [
          {
            metadata: { id: "saved-key", name: "my-router", integrationName: "openrouter" },
            status: { state: "ready" },
          },
        ],
        "saved-key",
      ),
    ).toBeNull();
  });

  it("does not adopt a different GitHub connection when the workspace has none saved", () => {
    const synced = syncSelectionsWithInstances(
      connectionData([githubInstance("other-github", "github-other")]),
      {},
      {},
      ONBOARDING_MANUAL_CONNECTION_NAMES,
    );

    expect(synced).toBeNull();
  });
});
