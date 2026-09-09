import { describe, expect, it, vi } from "vitest";

import {
  afterOnboardingPath,
  afterWorkspaceProvisioned,
  finishOnboardingError,
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
        githubReady: true,
        remainingCreditCents: 5000,
        hostedModelsLoading: false,
        plan: readyPlan,
      }),
    ).toBeNull();
  });
});

// Regression: provisioning used to read the issues answer off `setup`, which
// a caller in the middle of the same click (the ticket screen's Analyze
// action) can still hold at its pre-click value. An explicit `issuesChoice`
// argument removes that dependency, so a repository with zero issues (or
// any repository, since the ticket screen always answers "vcs") saves the
// answer it was just given instead of overwriting it with
// ISSUES_SOURCE_UNSPECIFIED.
describe("provisionWorkspace", () => {
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
      listPRFeedbackHandlers: vi.fn().mockResolvedValue([]),
      createPRFeedbackHandler: vi.fn().mockResolvedValue({ id: "handler-1" }),
      listApps: vi.fn().mockResolvedValue([]),
      workspaceName: "Payments Service",
      takenNames: [],
      appRepository: "acme/payments-service",
      backlogRepository: "acme/payments-service",
      issuesChoice: "vcs" as const,
      resolveDefaultBranch: vi.fn().mockResolvedValue("main"),
      github: { id: "github-1" },
      agentPlan: readyPlan,
      agentRewrite: {
        component: "runnerSuperPlane",
        model: "",
        planningModel: "",
      },
      ...overrides,
    };
  }

  it("saves the issues choice it was given, not a value read off setup state", async () => {
    const updateOnboarding = vi.fn().mockResolvedValue({});

    await provisionWorkspace(provisionArgs({ issuesChoice: "vcs", updateOnboarding }));

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
});

describe("afterWorkspaceProvisioned", () => {
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
    expect(navigate).toHaveBeenCalledWith("/acme-org/workspaces/SP/lines/line-1", { replace: true });
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

describe("afterOnboardingPath", () => {
  it("opens the board of the provisioned line, where the intake sits in Backlog", () => {
    expect(
      afterOnboardingPath({
        organizationId: "org-1",
        factoryKey: "SP",
        lineId: "line-1",
      }),
    ).toBe("/org-1/workspaces/SP/lines/line-1");
  });
});
