import { describe, expect, it } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import {
  apiIssuesSource,
  initialOnboardingSelections,
  initialWizardStep,
  isFactoryOnboardingComplete,
  localIssuesSource,
} from "./onboardingStatus";

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
});
