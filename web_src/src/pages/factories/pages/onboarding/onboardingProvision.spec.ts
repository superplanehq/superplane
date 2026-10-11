import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, FactoriesFactoryIntake } from "@/api-client";

import {
  DEFAULT_LINE_NAME,
  GITHUB_INTAKE_SOURCE,
  JIRA_INTAKE_SOURCE,
  provisionEventApps,
  provisionGithubIntake,
  provisionJiraIntake,
  provisionLine,
  provisionOnboardingIntake,
} from "./onboardingProvision";

function boundJiraIntake(id: string, integrationId: string, resourceId: string): FactoriesFactoryIntake {
  return { id, source: JIRA_INTAKE_SOURCE, integrationId, resourceId } as FactoriesFactoryIntake;
}

beforeEach(() => {
  sessionStorage.clear();
});

describe("provisionLine", () => {
  it("reuses a line that already has the implementation entrypoint", async () => {
    const createLine = vi.fn();
    const updateOnboarding = vi.fn();
    const installFactory = vi.fn();
    const factory = {
      id: "factory-1",
      lines: [
        {
          id: "line-1",
          steps: [{ app: { app: "app-1", entrypoint: "onrun-implement" } }],
        },
      ],
    } as FactoriesFactory;

    const result = await provisionLine({
      factory,
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      defaultBranch: "main",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(result).toEqual({ lineId: "line-1", primaryAppId: "app-1" });
    expect(createLine).not.toHaveBeenCalled();
    expect(installFactory).not.toHaveBeenCalled();
  });

  it("installs implement and creates a line that runs it", async () => {
    const createLine = vi.fn().mockResolvedValue({ id: "line-new" });
    const updateOnboarding = vi.fn().mockResolvedValue({});
    const installFactory = vi.fn().mockImplementation(async ({ factoryId }: { factoryId: string }) => ({
      canvasId: `canvas-${factoryId}`,
      canvasName: factoryId,
    }));

    const result = await provisionLine({
      factory: { id: "factory-1" } as FactoriesFactory,
      selections: {},
      appRepository: "acme/app",
      backlogRepository: "acme/backlog",
      defaultBranch: "master",
      installFactory,
      createLine,
      updateOnboarding,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["line-implementation"]);
    expect(installFactory.mock.calls.map(([input]) => input.installParams)).toEqual([
      { appRepository: "acme/app", backlogRepository: "acme/backlog", defaultBranch: "master" },
    ]);
    expect(createLine).toHaveBeenCalledWith({
      name: DEFAULT_LINE_NAME,
      steps: [
        {
          type: "runApp",
          app: { app: "canvas-line-implementation", entrypoint: "onrun-implement" },
        },
      ],
    });
    expect(result.lineId).toBe("line-new");
    expect(updateOnboarding).toHaveBeenCalledWith({
      provisionedAppId: result.primaryAppId,
      provisionedLineId: "line-new",
    });
  });
});

describe("provisionEventApps", () => {
  const agentRewrite = {
    component: "runnerClaudeCode",
    model: "claude-sonnet-4-6",
    credentials: { source: "integration" as const, name: "acme-claude" },
  };
  const installParams = {
    appRepository: "acme/app",
    backlogRepository: "acme/backlog",
    defaultBranch: "staging",
  };

  function installFactoryMock() {
    return vi.fn().mockImplementation(async ({ factoryId }: { factoryId: string }) => ({
      canvasId: `canvas-${factoryId}`,
      canvasName: factoryId,
    }));
  }

  it("installs pull request closure, then merge confidence, for a GitHub workspace", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      agentRewrite,
      vcsProvider: "github",
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["pr-closure", "risk-score"]);
    for (const factoryId of ["pr-closure", "risk-score"]) {
      expect(installFactory).toHaveBeenCalledWith(
        expect.objectContaining({
          factoryId,
          workspaceFactoryId: "factory-1",
          installParams,
          agentRewrite,
        }),
      );
    }
    expect(installFactory.mock.calls[1][0].installParams).not.toHaveProperty("enabledChecks");
    expect(installFactory.mock.calls[1][0].installParams).not.toHaveProperty("riskRules");
  });

  it("does not install pull request closure when the workspace already has it", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([{ id: "app-1", name: "PR Closure" }]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["risk-score"]);
  });

  it("does not install pull request closure when it was renamed to PR Closure (2)", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([{ id: "app-1", name: "PR Closure (2)" }]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["risk-score"]);
  });

  it("does not install merge confidence when one is named Merge confidence", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([{ id: "app-1", name: "Merge confidence" }]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["pr-closure"]);
  });

  it("does not install merge confidence when one is named Merge confidence (2)", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([{ id: "app-1", name: "Merge confidence (2)" }]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["pr-closure"]);
  });

  it("installs both event apps next to an app with an unrelated name", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([{ id: "app-1", name: "Backlog" }]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      installFactory,
      listApps,
    });

    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["pr-closure", "risk-score"]);
  });

  it("installs neither event app for a Bitbucket workspace", async () => {
    const installFactory = installFactoryMock();
    const listApps = vi.fn().mockResolvedValue([]);

    await provisionEventApps({
      factoryId: "factory-1",
      selections: {},
      ...installParams,
      vcsProvider: "bitbucket",
      installFactory,
      listApps,
    });

    expect(installFactory).not.toHaveBeenCalled();
    expect(listApps).not.toHaveBeenCalled();
  });
});

describe("provisionGithubIntake", () => {
  it("creates the GitHub intake for a workspace that has none", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" } as FactoriesFactoryIntake);

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
    expect(intake.id).toBe("intake-1");
  });

  it("leaves an existing GitHub intake alone so a retry adds no second copy", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-1", source: GITHUB_INTAKE_SOURCE }]);
    const createIntake = vi.fn();

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).not.toHaveBeenCalled();
    expect(intake.id).toBe("intake-1");
  });

  it("creates the GitHub intake next to an intake of another source", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-1", source: "SOURCE_SENTRY_EXCEPTIONS" }]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-2" } as FactoriesFactoryIntake);

    const intake = await provisionGithubIntake({ listIntakes, createIntake });

    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
    expect(intake.id).toBe("intake-2");
  });
});

describe("provisionJiraIntake", () => {
  it("creates the Jira intake with the selected connection and project", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    const intake = await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "PAY",
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: JIRA_INTAKE_SOURCE,
      integrationId: "jira-1",
      resourceId: "PAY",
    });
    expect(deleteIntake).not.toHaveBeenCalled();
    expect(intake.id).toBe("intake-jira");
  });

  it("creates the Jira intake with the chosen completion column", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "PAY",
      settings: { jiraMoveOnComplete: true, jiraCompletionColumn: "QA" },
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: JIRA_INTAKE_SOURCE,
      integrationId: "jira-1",
      resourceId: "PAY",
      settings: { jiraMoveOnComplete: true, jiraCompletionColumn: "QA" },
    });
  });

  it("reuses a Jira intake whose connection and project still match", async () => {
    const listIntakes = vi.fn().mockResolvedValue([boundJiraIntake("intake-1", "jira-1", "PAY")]);
    const createIntake = vi.fn();
    const deleteIntake = vi.fn();

    const intake = await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "PAY",
    });

    expect(createIntake).not.toHaveBeenCalled();
    expect(deleteIntake).not.toHaveBeenCalled();
    expect(intake.id).toBe("intake-1");
  });

  it("reuses a Jira intake created earlier in this session when the project is unchanged", async () => {
    const created = { id: "intake-1", source: JIRA_INTAKE_SOURCE } as FactoriesFactoryIntake;
    const listIntakes = vi
      .fn()
      .mockResolvedValueOnce([])
      .mockResolvedValue([{ id: "intake-1", source: JIRA_INTAKE_SOURCE }]);
    const createIntake = vi.fn().mockResolvedValue(created);
    const deleteIntake = vi.fn();

    await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "PAY",
    });
    const intake = await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "PAY",
    });

    expect(createIntake).toHaveBeenCalledTimes(1);
    expect(deleteIntake).not.toHaveBeenCalled();
    expect(intake.id).toBe("intake-1");
  });

  it("replaces a Jira intake when the project changed after a failed finish", async () => {
    const listIntakes = vi.fn().mockResolvedValue([boundJiraIntake("intake-old", "jira-1", "PAY")]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-new" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn().mockResolvedValue({});

    const intake = await provisionJiraIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      integrationId: "jira-1",
      resourceId: "CORE",
    });

    expect(deleteIntake).toHaveBeenCalledWith("intake-old");
    expect(createIntake).toHaveBeenCalledWith({
      source: JIRA_INTAKE_SOURCE,
      integrationId: "jira-1",
      resourceId: "CORE",
    });
    expect(intake.id).toBe("intake-new");
  });
});

describe("provisionOnboardingIntake", () => {
  it("creates the GitHub intake when the ticket source is GitHub", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    const intake = await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      issuesChoice: "vcs",
    });

    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
    expect(deleteIntake).not.toHaveBeenCalled();
    expect(intake?.id).toBe("intake-1");
  });

  it("creates a Jira intake when the ticket source is Jira", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    const intake = await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      issuesChoice: "jira",
      jira: { integrationId: "jira-1", projectId: "PAY" },
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: JIRA_INTAKE_SOURCE,
      integrationId: "jira-1",
      resourceId: "PAY",
    });
    expect(intake?.id).toBe("intake-jira");
  });

  it("removes a leftover GitHub intake when the retry selects Jira", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-github", source: GITHUB_INTAKE_SOURCE }]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn().mockResolvedValue({});

    await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      issuesChoice: "jira",
      jira: { integrationId: "jira-1", projectId: "PAY" },
    });

    expect(deleteIntake).toHaveBeenCalledWith("intake-github");
    expect(createIntake).toHaveBeenCalledWith({
      source: JIRA_INTAKE_SOURCE,
      integrationId: "jira-1",
      resourceId: "PAY",
    });
  });

  it("creates a Linear intake when the ticket source is Linear", async () => {
    const listIntakes = vi.fn().mockResolvedValue([]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn();

    const intake = await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      issuesChoice: "linear",
      linear: { integrationId: "linear-1", projectIds: ["project-1", "project-2"] },
    });

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      resourceId: "project-1,project-2",
      settings: { linearProjectIds: ["project-1", "project-2"] },
    });
    expect(intake?.id).toBe("intake-linear");
  });

  it("removes a leftover Jira intake when the retry selects GitHub", async () => {
    const listIntakes = vi.fn().mockResolvedValue([{ id: "intake-jira", source: JIRA_INTAKE_SOURCE }]);
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-github" } as FactoriesFactoryIntake);
    const deleteIntake = vi.fn().mockResolvedValue({});

    await provisionOnboardingIntake({
      listIntakes,
      createIntake,
      deleteIntake,
      issuesChoice: "vcs",
    });

    expect(deleteIntake).toHaveBeenCalledWith("intake-jira");
    expect(createIntake).toHaveBeenCalledWith({ source: GITHUB_INTAKE_SOURCE });
  });
});
