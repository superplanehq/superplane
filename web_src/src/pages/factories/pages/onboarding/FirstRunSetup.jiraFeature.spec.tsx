import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { intakeCatalogAvailability, seededIntakeCatalog } from "@/test/intakeCatalog";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import { shouldClearSavedJiraChoice, savedJiraChoiceBlock } from "./useFirstRunSetupFlow";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const feature = vi.hoisted(() => ({
  jiraIntake: true,
  linearIntake: false,
  linearStatus: "planned",
  organizationReady: true,
  isLoading: false,
}));

vi.mock("@/hooks/useIntakeCatalogAvailability", () => ({
  useIntakeCatalogAvailability: () => {
    if (feature.isLoading) {
      return intakeCatalogAvailability(null);
    }
    if (!feature.organizationReady) {
      return { ...intakeCatalogAvailability(null), loading: false, error: true };
    }
    const granted = [
      ...(feature.jiraIntake ? ["jira-issues"] : []),
      ...(feature.linearIntake ? ["linear-issues"] : []),
    ];
    return intakeCatalogAvailability(
      seededIntakeCatalog(granted, feature.linearIntake ? { "linear-issues": { status: feature.linearStatus } } : {}),
    );
  },
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => false,
    isLoading: false,
    organizationReady: true,
    enabledExperimentalFeatures: [],
  }),
}));

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    routeSegment: "pay",
    factory: { id: "factory-1", key: "PAY", name: "New workspace", onboarding: { vcsIntegrationId: "github-1" } },
    factories: [{ id: "factory-1", key: "PAY", name: "New workspace", onboarding: { vcsIntegrationId: "github-1" } }],
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "user-1" } }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: () => ({
    data: { providerConfigured: false, identity: undefined, repositories: [], installUrl: "" },
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
  }),
}));

vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: () => ({
    data: {
      appConfigured: true,
      identity: { userId: "42", login: "octocat" },
      repositories: [{ repositoryId: "201", installationId: "101", fullName: "acme/api", defaultBranch: "main" }],
      pendingRequests: [],
      synchronizing: false,
    },
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
    configureInstallation: { mutateAsync: vi.fn() },
  }),
  useGitHubInstallationChecks: vi.fn(),
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Acme" }], refetch: vi.fn() }),
}));

vi.mock("./AgentStep", () => ({
  AgentStep: () => <div data-testid="agent-step" />,
}));

function setupState(): OnboardingSetupApi {
  const { result } = renderHook(() => useOnboardingSetupState("Payments Service", { simulateDiscovery: false }));
  return result.current;
}

type SetupOptions = NonNullable<Parameters<typeof useOnboardingSetupState>[1]>;

function LiveJiraSetup({
  model,
  setupOptions,
  setupRef,
}: {
  model: OnboardingPageModel;
  setupOptions: SetupOptions;
  setupRef: { current: OnboardingSetupApi | null };
}) {
  const setup = useOnboardingSetupState("Payments Service", setupOptions);
  setupRef.current = setup;
  return <FirstRunSetup model={{ ...model, setup }} />;
}

function pageModel(overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  return {
    setup: setupState(),
    hostedAgentReady: false,
    hostedModelsAvailable: false,
    hostedModelsAvailableLoading: false,
    bringYourOwnKey: false,
    bringYourOwnKeyLoading: false,
    customProvider: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    openSection: "issues",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    selectCatalogRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketForgeRepository: vi.fn().mockResolvedValue(true),
    bitbucketIntegrationId: "",
    integrationDialogs: <></>,
    canConfigureWorkspace: true,
    saving: false,
    saveName: vi.fn().mockResolvedValue(true),
    saveIssues: vi.fn().mockResolvedValue(true),
    finish: vi.fn(),
    provisionedDestination: null,
    githubOwner: undefined,
    jiraIntegrationId: "",
    jiraProjectId: "",
    setJiraProjectId: vi.fn(),
    jiraCompletion: { jiraMoveOnComplete: true, jiraCompletionColumn: "" },
    jiraProjects: [],
    jiraProjectsLoading: false,
    jiraProjectsError: false,
    retryJiraProjects: vi.fn(),
    linearIntegrationId: "",
    linearProjectIds: [],
    toggleLinearProject: vi.fn(),
    linearProjects: [],
    linearProjectsLoading: false,
    linearProjectsError: false,
    retryLinearProjects: vi.fn(),
    ...overrides,
  };
}

function renderLiveSetup(model: OnboardingPageModel, setupOptions: SetupOptions) {
  const setupRef = { current: null as OnboardingSetupApi | null };
  const view = render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=issues"]}>
      <LiveJiraSetup model={model} setupOptions={setupOptions} setupRef={setupRef} />
    </MemoryRouter>,
  );
  return { ...view, setupRef };
}

describe("FirstRunSetup Jira intake feature", () => {
  beforeEach(() => {
    feature.jiraIntake = true;
    feature.linearIntake = false;
    feature.linearStatus = "planned";
    feature.organizationReady = true;
    feature.isLoading = false;
  });

  it("hides Jira and does not provision a Jira intake when the feature is off", async () => {
    feature.jiraIntake = false;
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      jiraIntegrationId: "jira-1",
      jiraProjectId: "PAY",
      jiraProjects: [{ id: "PAY", name: "Payments" }],
    });

    const { setupRef } = renderLiveSetup(model, {
      simulateDiscovery: false,
      connected: new Set(["jira"]),
      initial: { issuesChoice: "jira" },
    });

    expect(screen.queryByText(FIRST_RUN_COPY.tickets.jira)).not.toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.linear)).not.toBeInTheDocument();
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Jira" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-jira-choice-notice")).not.toBeInTheDocument();
    await waitFor(() => expect(setupRef.current?.issuesChoice).toBeNull());

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(model.saveIssues).toHaveBeenCalledTimes(1);
    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    expect(model.finish).toHaveBeenCalledWith("vcs");
  });

  it("keeps a saved Jira choice when the organization lookup fails", async () => {
    feature.jiraIntake = false;
    feature.organizationReady = false;
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      jiraIntegrationId: "jira-1",
      jiraProjectId: "PAY",
      jiraProjects: [{ id: "PAY", name: "Payments" }],
    });

    const { setupRef } = renderLiveSetup(model, {
      simulateDiscovery: false,
      connected: new Set(["jira"]),
      initial: { issuesChoice: "jira" },
    });

    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Jira" })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-jira-choice-notice")).toHaveTextContent(
      FIRST_RUN_COPY.tickets.jiraLookupFailed,
    );
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    expect(setupRef.current?.issuesChoice).toBe("jira");

    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(setupRef.current?.issuesChoice).toBe("vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    expect(model.finish).toHaveBeenCalledWith("vcs");
  });

  it("does not mark Jira as coming soon while the feature lookup is loading", () => {
    feature.isLoading = true;
    feature.jiraIntake = true;

    renderLiveSetup(pageModel(), { simulateDiscovery: false });

    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jiraLookupLoading)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linearLookupLoading)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.jiraSoonHelper)).not.toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.linearSoonHelper)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Jira" })).not.toBeInTheDocument();
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
  });

  it("shows Linear when the organization can use it", async () => {
    feature.linearIntake = true;
    feature.linearStatus = "ga";
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      linearIntegrationId: "linear-1",
      linearProjectIds: ["project-1"],
      linearProjects: [{ id: "project-1", name: "Platform" }],
    });

    renderLiveSetup(model, {
      simulateDiscovery: false,
      connected: new Set(["linear"]),
      initial: { issuesChoice: "linear" },
    });

    expect(screen.getByText(FIRST_RUN_COPY.tickets.linearHelper)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.linearSoonHelper)).not.toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linear).closest('[data-soon="true"]')).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-linear-projects")).toBeInTheDocument();

    await user.click(screen.getByTestId("linear-project-project-1"));
    expect(model.toggleLinearProject).toHaveBeenCalledWith("project-1");
  });
});

describe("shouldClearSavedJiraChoice", () => {
  it("clears a saved Jira choice only after the organization lookup confirms the feature is off", () => {
    expect(
      shouldClearSavedJiraChoice({
        issuesChoice: "jira",
        featureLoading: false,
        jiraAvailable: false,
        organizationReady: true,
      }),
    ).toBe(true);
  });

  it("does not clear a saved Jira choice when the organization lookup has not succeeded", () => {
    expect(
      shouldClearSavedJiraChoice({
        issuesChoice: "jira",
        featureLoading: false,
        jiraAvailable: false,
        organizationReady: false,
      }),
    ).toBe(false);
  });
});

describe("savedJiraChoiceBlock", () => {
  it("blocks a saved Jira choice when the organization lookup fails", () => {
    expect(
      savedJiraChoiceBlock({
        issuesChoice: "jira",
        featureLoading: false,
        jiraAvailable: false,
        organizationReady: false,
      }),
    ).toBe("lookup-failed");
  });

  it("blocks a saved Jira choice while the feature lookup is loading", () => {
    expect(
      savedJiraChoiceBlock({
        issuesChoice: "jira",
        featureLoading: true,
        jiraAvailable: false,
        organizationReady: false,
      }),
    ).toBe("loading");
  });

  it("does not block Jira after the lookup confirms the feature is off", () => {
    expect(
      savedJiraChoiceBlock({
        issuesChoice: "jira",
        featureLoading: false,
        jiraAvailable: false,
        organizationReady: true,
      }),
    ).toBeNull();
  });
});
