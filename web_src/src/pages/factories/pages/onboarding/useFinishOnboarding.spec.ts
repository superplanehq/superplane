import { describe, expect, it, vi } from "bun:test";

import {
  afterOnboardingPath,
  afterWorkspaceProvisioned,
  finishOnboardingError,
  MissingRepositoryBranchError,
  onboardingDefaultBranch,
  provisionWorkspace,
} from "./useFinishOnboarding";

const readyPlan = {
  component: "runnerSuperPlane",
  credentialsSource: "hosted",
  harness: "AGENT_HARNESS_SUPERPLANE",
  model: "",
  planningModel: "",
} as const;

describe("finishOnboardingError", () => {
  it("allows finish when GitHub, repositories, name, and an agent plan are ready", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: "acme/web",
        workspaceName: "Web",
        vcsReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
      }),
    ).toBeNull();
  });

  it("requires a Jira connection and project when the ticket source is Jira", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: "acme/web",
        workspaceName: "Web",
        vcsReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
        issuesChoice: "jira",
        jiraReady: false,
        jiraProjectId: "",
      }),
    ).toBe("Connect Jira, then choose a project.");
  });

  it("requires a Linear connection and project when the ticket source is Linear", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: "acme/web",
        workspaceName: "Web",
        vcsReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
        issuesChoice: "linear",
        linearReady: false,
        linearProjectIds: [],
      }),
    ).toBe("Connect Linear, then choose a project.");
  });

  it("names Bitbucket when a Bitbucket workspace has no connection", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: "acme/web",
        workspaceName: "Web",
        vcsReady: false,
        vcsHost: "bitbucket",
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
      }),
    ).toBe("Connect Bitbucket before you continue.");
  });

  it("asks for repositories when GitHub is connected and a repository is missing", () => {
    expect(
      finishOnboardingError({
        appRepository: "acme/web",
        backlogRepository: null,
        workspaceName: "Web",
        vcsReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
      }),
    ).toBe("Select the code repository and the backlog repository.");
  });
});

// Regression: provisioning used to read the issues answer off `setup`, which
// a caller in the middle of the same click (the ticket screen's Analyze
// action) can still hold at its pre-click value. An explicit `issuesChoice`
// argument removes that dependency, so a repository with zero issues (or
// any repository, since the ticket screen always answers "vcs") saves the
// answer it was just given instead of overwriting it with
// ISSUES_SOURCE_UNSPECIFIED.
function provisionArgs(overrides: Partial<Parameters<typeof provisionWorkspace>[0]> = {}) {
  return {
    organizationId: "org-1",
    factoryId: "factory-1",
    factory: null,
    selections: {},
    updateFactory: vi.fn().mockResolvedValue({}),
    updateOnboarding: vi.fn().mockResolvedValue({}),
    installFactory: vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" }),
    createLine: vi.fn().mockResolvedValue({ id: "line-1" }),
    listIntakes: vi.fn().mockResolvedValue([]),
    createIntake: vi.fn().mockResolvedValue({ id: "intake-1" }),
    updateIntake: vi.fn().mockResolvedValue({ id: "intake-1" }),
    deleteIntake: vi.fn().mockResolvedValue({}),
    listApps: vi.fn().mockResolvedValue([]),
    workspaceName: "Payments Service",
    takenNames: [],
    appRepository: "acme/payments-service",
    backlogRepository: "acme/payments-service",
    issuesChoice: "vcs" as const,
    resolveDefaultBranch: vi.fn().mockResolvedValue("main"),
    vcs: { id: "github-1", provider: "github" as const },
    agentPlan: readyPlan,
    agentRewrite: {
      component: "runnerSuperPlane",
      model: "",
      planningModel: "",
    },
    ...overrides,
  };
}

describe("provisionWorkspace", () => {
  it.each(["vcs", "jira", "linear"] as const)(
    "can skip the initial %s import without skipping intake creation",
    async (issuesChoice) => {
      const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" });
      await provisionWorkspace(
        provisionArgs({
          issuesChoice,
          skipInitialImport: true,
          createIntake,
          jira: { integrationId: "jira-1", projectId: "project-1" },
          linear: { integrationId: "linear-1", projectIds: ["project-1"] },
        }),
      );
      expect(createIntake).toHaveBeenCalledWith(expect.objectContaining({ skipInitialImport: true }));
    },
  );

  it.each(["vcs", "jira", "linear"] as const)(
    "starts the %s intake paused when automatic intake is off",
    async (issuesChoice) => {
      const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" });
      await provisionWorkspace(
        provisionArgs({
          issuesChoice,
          paused: true,
          createIntake,
          jira: { integrationId: "jira-1", projectId: "project-1" },
          linear: { integrationId: "linear-1", projectIds: ["project-1"] },
        }),
      );
      expect(createIntake).toHaveBeenCalledWith(expect.objectContaining({ paused: true }));
      expect(createIntake).not.toHaveBeenCalledWith(expect.objectContaining({ skipInitialImport: true }));
    },
  );

  it("saves the issues choice it was given, not a value read off setup state", async () => {
    const updateOnboarding = vi.fn().mockResolvedValue({});
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-1" });

    await provisionWorkspace(provisionArgs({ issuesChoice: "vcs", updateOnboarding, createIntake }));

    expect(createIntake).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    const issuesSourceCalls = updateOnboarding.mock.calls
      .map(([input]) => input.issuesSource)
      .filter((value) => value !== undefined);
    expect(issuesSourceCalls).toEqual(["ISSUES_SOURCE_VCS"]);
  });

  it("still provisions a repository with no issues, because zero issues is not a blocker", async () => {
    const updateOnboarding = vi.fn().mockResolvedValue({});

    const result = await provisionWorkspace(
      provisionArgs({ issuesChoice: "vcs", backlogRepository: "acme/quiet-repo", updateOnboarding }),
    );

    expect(result).toEqual({ lineId: "line-1" });
    const completeCall = updateOnboarding.mock.calls.find(([input]) => input.complete);
    expect(completeCall?.[0]).toMatchObject({ complete: true });
  });

  it("creates a Jira intake instead of GitHub issues when the ticket source is Jira", async () => {
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" });
    const updateOnboarding = vi.fn().mockResolvedValue({});

    await provisionWorkspace(
      provisionArgs({
        issuesChoice: "jira",
        createIntake,
        updateOnboarding,
        jira: { integrationId: "jira-1", projectId: "PAY" },
      }),
    );

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_JIRA_ISSUES",
      integrationId: "jira-1",
      resourceId: "PAY",
    });
    const issuesSourceCalls = updateOnboarding.mock.calls
      .map(([input]) => input.issuesSource)
      .filter((value) => value !== undefined);
    expect(issuesSourceCalls).toEqual(["ISSUES_SOURCE_JIRA"]);
  });

  it("creates a Jira intake with the chosen completion column", async () => {
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-jira" });

    await provisionWorkspace(
      provisionArgs({
        issuesChoice: "jira",
        createIntake,
        jira: {
          integrationId: "jira-1",
          projectId: "PAY",
          settings: { jiraMoveOnComplete: true, jiraCompletionColumn: "QA" },
        },
      }),
    );

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_JIRA_ISSUES",
      integrationId: "jira-1",
      resourceId: "PAY",
      settings: { jiraMoveOnComplete: true, jiraCompletionColumn: "QA" },
    });
  });

  it("creates a Linear intake instead of GitHub issues when the ticket source is Linear", async () => {
    const createIntake = vi.fn().mockResolvedValue({ id: "intake-linear" });
    const updateOnboarding = vi.fn().mockResolvedValue({});

    await provisionWorkspace(
      provisionArgs({
        issuesChoice: "linear",
        createIntake,
        updateOnboarding,
        linear: { integrationId: "linear-1", projectIds: ["project-1"] },
      }),
    );

    expect(createIntake).toHaveBeenCalledWith({
      source: "SOURCE_LINEAR_ISSUES",
      integrationId: "linear-1",
      resourceId: "project-1",
      settings: { linearProjectIds: ["project-1"] },
    });
    const issuesSourceCalls = updateOnboarding.mock.calls
      .map(([input]) => input.issuesSource)
      .filter((value) => value !== undefined);
    expect(issuesSourceCalls).toEqual(["ISSUES_SOURCE_LINEAR"]);
  });

  it("provisions the Bitbucket Implement app and no GitHub intake for a Bitbucket workspace", async () => {
    const createIntake = vi.fn();
    const installFactory = vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" });
    const listApps = vi.fn().mockResolvedValue([]);

    await provisionWorkspace(
      provisionArgs({
        vcs: { id: "bitbucket-1", provider: "bitbucket" },
        createIntake,
        installFactory,
        listApps,
      }),
    );

    expect(createIntake).not.toHaveBeenCalled();
    expect(installFactory.mock.calls.map(([input]) => input.factoryId)).toEqual(["line-implementation", "pr-closure"]);
    for (const [input] of installFactory.mock.calls) {
      expect(input.vcsProvider).toBe("bitbucket");
    }
  });

  it("does not create a comments handler during workspace setup", async () => {
    await provisionWorkspace(provisionArgs());
  });

  it("does not create a GitHub intake when the ticket source is Jira without a project", async () => {
    const createIntake = vi.fn();

    await expect(provisionWorkspace(provisionArgs({ issuesChoice: "jira", createIntake }))).rejects.toThrow(
      "Connect Jira, then choose a project.",
    );

    expect(createIntake).not.toHaveBeenCalled();
  });
});

describe("afterWorkspaceProvisioned", () => {
  it("opens the factory directly without skipping the import", async () => {
    const navigate = vi.fn();
    const onProvisioned = vi.fn();
    await afterWorkspaceProvisioned({
      factory: null,
      organizationId: "org",
      factoryId: "factory",
      factoryKey: "PAY",
      lineId: "line",
      invalidateAccountOrganizations: vi.fn(),
      navigate,
      onProvisioned,
      bypassAnalysis: true,
      skipInitialImport: false,
    });
    expect(navigate).toHaveBeenCalled();
    expect(onProvisioned).not.toHaveBeenCalled();
  });
  it("opens the board directly when the user skipped importing", async () => {
    const navigate = vi.fn();
    const onProvisioned = vi.fn();
    await afterWorkspaceProvisioned({
      factory: null,
      organizationId: "org",
      factoryId: "factory-1",
      factoryKey: "SP",
      lineId: "line-1",
      invalidateAccountOrganizations: vi.fn(),
      navigate,
      onProvisioned,
      skipInitialImport: true,
    });
    expect(onProvisioned).not.toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith("/org/workspaces/sp/lines/line-1", { replace: true });
  });

  it("renames the organization, refreshes the switcher, then opens the new slug", async () => {
    const updateOrganization = vi.fn().mockResolvedValue("acme-org");
    const invalidateAccountOrganizations = vi.fn();
    const navigate = vi.fn();

    await afterWorkspaceProvisioned({
      factory: { onboarding: { initial: true } },
      owner: "Acme Org",
      organizationId: "test-test",
      factoryId: "factory-1",
      factoryKey: "SP",
      lineId: "line-1",
      updateOrganization,
      invalidateAccountOrganizations,
      navigate,
    });

    expect(updateOrganization).toHaveBeenCalledWith({ name: "Acme Org", slug: "acme-org" });
    expect(invalidateAccountOrganizations).toHaveBeenCalledTimes(1);
    expect(navigate).toHaveBeenCalledWith("/acme-org/workspaces/sp/lines/line-1", { replace: true });
  });

  it("hands the renamed organization to onProvisioned instead of navigating", async () => {
    const updateOrganization = vi.fn().mockResolvedValue("acme-org");
    const invalidateAccountOrganizations = vi.fn();
    const navigate = vi.fn();
    const onProvisioned = vi.fn();

    await afterWorkspaceProvisioned({
      factory: { onboarding: { initial: true } },
      owner: "Acme Org",
      organizationId: "test-test",
      factoryId: "factory-1",
      factoryKey: "SP",
      lineId: "line-1",
      updateOrganization,
      invalidateAccountOrganizations,
      navigate,
      onProvisioned,
    });

    expect(navigate).not.toHaveBeenCalled();
    expect(onProvisioned).toHaveBeenCalledWith({
      organizationId: "acme-org",
      factoryKey: "SP",
      lineId: "line-1",
    });
  });
});

describe("onboardingDefaultBranch", () => {
  it("keeps a saved branch when the lookup returns an empty name", async () => {
    await expect(onboardingDefaultBranch({ lookup: async () => "", saved: "develop" })).resolves.toBe("develop");
  });

  it("keeps a saved branch when the lookup fails", async () => {
    await expect(
      onboardingDefaultBranch({
        lookup: async () => {
          throw new Error("github unavailable");
        },
        saved: "develop",
      }),
    ).resolves.toBe("develop");
  });

  it("rejects when the lookup and the saved branch are both empty", async () => {
    await expect(onboardingDefaultBranch({ lookup: async () => "", saved: "" })).rejects.toBeInstanceOf(
      MissingRepositoryBranchError,
    );
  });
});

describe("provisionWorkspace default branch", () => {
  it("does not save or provision when no default branch is known", async () => {
    const updateOnboarding = vi.fn().mockResolvedValue({});
    const installFactory = vi.fn();
    const updateFactory = vi.fn();

    await expect(
      provisionWorkspace(
        provisionArgs({
          factory: { name: "Payments Service", onboarding: { defaultBranch: "" } },
          resolveDefaultBranch: vi.fn().mockResolvedValue(""),
          updateOnboarding,
          installFactory,
          updateFactory,
        }),
      ),
    ).rejects.toBeInstanceOf(MissingRepositoryBranchError);

    expect(updateFactory).not.toHaveBeenCalled();
    expect(updateOnboarding).not.toHaveBeenCalled();
    expect(installFactory).not.toHaveBeenCalled();
  });

  it("provisions with a saved branch when the lookup returns an empty name", async () => {
    const installFactory = vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" });

    await provisionWorkspace(
      provisionArgs({
        factory: { name: "Payments Service", onboarding: { defaultBranch: "develop" } },
        resolveDefaultBranch: vi.fn().mockResolvedValue(""),
        installFactory,
      }),
    );

    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({ installParams: expect.objectContaining({ defaultBranch: "develop" }) }),
    );
  });

  it("provisions with the branch returned by the onboarding update", async () => {
    const updateOnboarding = vi.fn().mockImplementation(async (input: { defaultBranch?: string }) => {
      if (input.defaultBranch) return { onboarding: { defaultBranch: "release" } };
      return {};
    });
    const installFactory = vi.fn().mockResolvedValue({ canvasId: "canvas-1", canvasName: "canvas-1" });

    await provisionWorkspace(
      provisionArgs({
        resolveDefaultBranch: vi.fn().mockResolvedValue("master"),
        updateOnboarding,
        installFactory,
      }),
    );

    expect(installFactory).toHaveBeenCalledWith(
      expect.objectContaining({ installParams: expect.objectContaining({ defaultBranch: "release" }) }),
    );
  });
});

describe("afterOnboardingPath", () => {
  it("opens the board of the provisioned line, where the intake sits in Backlog", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
      }),
    ).toBe("/org-1/workspaces/sp/lines/line-1");
  });
});
