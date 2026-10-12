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
} from "./onboardingStatus";

function githubInstance(id: string, name: string): OrganizationsIntegration {
  return {
    metadata: { id, name, integrationName: "github" },
    status: { state: "ready" },
  };
}

function connectionData(instances: OrganizationsIntegration[]): IntegrationInstanceSummary[] {
  return ONBOARDING_CONNECTION_NAMES.map((name) => ({
    name,
    allInstances: name === "github" ? instances : [],
    readyInstances: name === "github" ? instances : [],
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

  it("marks the saved GitHub installation ready after a reload", () => {
    const saved = initialOnboardingSelections({ vcsIntegrationId: "saved-github" });
    const synced = syncSelectionsWithInstances(
      connectionData([githubInstance("saved-github", "github-acme"), githubInstance("other-github", "github-other")]),
      saved,
      {},
      ONBOARDING_MANUAL_CONNECTION_NAMES,
    );

    expect(synced).toEqual({
      github: { id: "saved-github", name: "github-acme", ready: true },
    });
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
