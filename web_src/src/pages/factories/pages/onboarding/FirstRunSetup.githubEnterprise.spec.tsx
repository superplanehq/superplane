import { act, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FirstRunSetup } from "./FirstRunSetup";
import type { IntegrationId } from "./onboardingFixtures";
import {
  githubOnboardingIntegrationId,
  resolvedOnboardingGithubEnterprise,
  writeOnboardingGithubEnterprise,
} from "./onboardingGithubEnterprise";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const gheApp = vi.hoisted(() => ({
  ready: false,
  currentStepName: "",
  connect: vi.fn().mockResolvedValue(true),
  submitOwner: vi.fn().mockResolvedValue(true),
  resume: vi.fn().mockReturnValue(true),
}));

vi.mock("./useFirstRunGitHubEnterprise", () => ({
  useFirstRunGitHubEnterprise: () => ({
    connect: gheApp.connect,
    submitOwner: gheApp.submitOwner,
    resume: gheApp.resume,
    ready: gheApp.ready,
    currentStepName: gheApp.currentStepName,
  }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [{ name: "acme/api" }, { name: "acme/web" }],
    isLoading: false,
    isError: false,
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
    data: { providerConfigured: false, accountConnectionRequired: true, identity: undefined, repositories: [] },
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
    configureInstallation: { mutateAsync: vi.fn() },
    selectIdentity: { mutateAsync: vi.fn() },
  }),
  useGitHubInstallationChecks: vi.fn(),
}));

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    routeSegment: "pay",
    factory: { id: "factory-1", key: "PAY", name: "Payments" },
    factories: [{ id: "factory-1", key: "PAY", name: "Payments" }],
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "account-1",
      name: "Ada Lovelace",
      email: "ada@example.com",
      license: { edition: "enterprise", features: ["github_enterprise"] },
    },
  }),
}));

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Acme" }] }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => true, isLoading: false, organizationReady: true }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({
    data: { billingEnabled: false, productFeatures: { byo_key: true } },
    isPending: false,
    isSuccess: true,
    refetch: async () => ({ data: { billingEnabled: false, productFeatures: { byo_key: true } } }),
  }),
}));

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" },
    isSuccess: true,
    isPending: false,
    isError: false,
    refetch: async () => ({ data: { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" } }),
  }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));
vi.mock("./AgentStep", () => ({ AgentStep: () => <div data-testid="agent-step" /> }));

function pageModel(setup: OnboardingSetupApi, overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  return {
    setup,
    hostedAgentReady: true,
    hostedModelsAvailable: true,
    hostedModelsAvailableLoading: false,
    bringYourOwnKey: false,
    bringYourOwnKeyLoading: false,
    codexSubscriptionOffered: false,
    codexSubscriptionLoading: false,
    codexConnection: undefined,
    codexSubscriptionReady: false,
    startCodexConnection: vi.fn(),
    cancelCodexConnection: vi.fn(),
    codexStarting: false,
    codexCanceling: false,
    customProvider: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    agentModelPicker: undefined,
    openSection: "vcs",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    selectCatalogRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketForgeRepository: vi.fn().mockResolvedValue(true),
    selectGitHubEnterpriseRepository: vi.fn().mockResolvedValue(true),
    bindGitHubEnterpriseIntegration: vi.fn(),
    githubIntegrationId: "int-ghe",
    githubHost: "",
    bitbucketIntegrationId: "",
    integrationDialogs: <></>,
    canConfigureWorkspace: true,
    saving: false,
    saveName: vi.fn().mockResolvedValue(true),
    saveAgent: vi.fn().mockResolvedValue(true),
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

function renderSetup(setup: OnboardingSetupApi, path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={pageModel(setup)} />
    </MemoryRouter>,
  );
}

describe("FirstRunSetup GitHub Enterprise Cloud", () => {
  beforeEach(() => {
    sessionStorage.clear();
    gheApp.ready = false;
    gheApp.currentStepName = "";
    gheApp.connect.mockClear();
    gheApp.submitOwner.mockClear();
    gheApp.resume.mockClear();
  });

  it("clears a github.com repository when switching to Enterprise", async () => {
    const user = userEvent.setup();
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    act(() => {
      hook.result.current.selectVcsHost("github");
      hook.result.current.selectRepo("acme/dotcom");
    });
    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup");

    await user.click(screen.getByTestId("first-run-get-started"));
    await user.click(within(screen.getByTestId("first-run-host-github-enterprise")).getByRole("button"));
    expect(hook.result.current.selectedRepo).toBeNull();
  });

  it("does not continue with a repository missing from the Enterprise list", () => {
    gheApp.ready = true;
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    const connected = new Set<IntegrationId>(["github"]);
    Object.defineProperty(hook.result.current, "connected", { get: () => connected });
    act(() => {
      hook.result.current.selectVcsHost("github");
      hook.result.current.selectRepo("acme/dotcom");
    });

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=repo");

    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
  });

  it("opens the hostname screen from GitHub Enterprise Cloud", async () => {
    const user = userEvent.setup();
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup");

    await user.click(screen.getByTestId("first-run-get-started"));
    await user.click(within(screen.getByTestId("first-run-host-github-enterprise")).getByRole("button"));
    expect(screen.getByTestId("first-run-github-enterprise-host")).toBeInTheDocument();
  });

  it("creates a private GitHub App for the entered host", async () => {
    const user = userEvent.setup();
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup");

    await user.click(screen.getByTestId("first-run-get-started"));
    await user.click(within(screen.getByTestId("first-run-host-github-enterprise")).getByRole("button"));
    await user.type(screen.getByTestId("first-run-github-enterprise-hostname"), "acme.ghe.com");
    await user.click(screen.getByTestId("first-run-github-enterprise-continue"));

    await waitFor(() => expect(gheApp.connect).toHaveBeenCalled());
    expect(gheApp.connect).toHaveBeenCalledWith("acme.ghe.com", expect.stringContaining("step=repo"));
    expect(screen.getByTestId("first-run-github-enterprise-host")).toBeInTheDocument();
    expect(screen.queryByText(/settings\/organization\/integrations/)).not.toBeInTheDocument();
  });

  it("asks for the GitHub organization after the hostname is accepted", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=ghe-host");

    expect(screen.getByTestId("first-run-github-enterprise-organization")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-enterprise-hostname")).not.toBeInTheDocument();
  });

  it("creates the GitHub App from the organization step", async () => {
    const user = userEvent.setup();
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=ghe-host");

    await user.type(screen.getByTestId("first-run-github-enterprise-organization"), "acme");
    await user.click(screen.getByTestId("first-run-github-enterprise-create-app"));

    expect(gheApp.submitOwner).toHaveBeenCalledWith("acme", expect.stringContaining("step=repo"));
  });

  it("returns to the hostname from the organization step", async () => {
    const user = userEvent.setup();
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=ghe-host");

    expect(screen.getByTestId("first-run-github-enterprise-organization")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(await screen.findByTestId("first-run-github-enterprise-hostname")).toBeInTheDocument();
  });

  it("returns to the hostname screen from repository selection", async () => {
    const user = userEvent.setup();
    gheApp.ready = true;
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    const connected = new Set<IntegrationId>(["github"]);
    Object.defineProperty(hook.result.current, "connected", { get: () => connected });
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=repo");

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(await screen.findByTestId("first-run-github-enterprise-host")).toBeInTheDocument();
  });

  it("returns to repository selection when Continue reuses a ready connection", async () => {
    const user = userEvent.setup();
    gheApp.ready = true;
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    const connected = new Set<IntegrationId>(["github"]);
    Object.defineProperty(hook.result.current, "connected", { get: () => connected });
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=repo");

    await user.click(screen.getByTestId("first-run-back"));
    expect(await screen.findByTestId("first-run-github-enterprise-hostname")).toHaveValue("acme.ghe.com");
    await user.click(screen.getByTestId("first-run-github-enterprise-continue"));

    expect(await screen.findByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    expect(gheApp.connect).not.toHaveBeenCalled();
  });

  it("lists repositories from the GitHub Enterprise integration", () => {
    gheApp.ready = true;
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    const connected = new Set<IntegrationId>(["github"]);
    Object.defineProperty(hook.result.current, "connected", { get: () => connected });
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=repo");

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    expect(screen.getByText("acme/api")).toBeInTheDocument();
  });

  it("keeps a pending GitHub App on the finish-setup screen", () => {
    gheApp.currentStepName = "createApp";
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=ghe-host"]}>
        <FirstRunSetup model={pageModel(hook.result.current, { githubIntegrationId: "" })} />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("first-run-github-enterprise-choose")).not.toBeInTheDocument();
    expect(screen.queryByText("SuperPlane cannot see repositories for this GitHub App.")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-enterprise-finish")).toBeInTheDocument();
  });

  it("resumes Enterprise from the saved integration host after the tab is closed", () => {
    gheApp.ready = true;
    const githubIntegrationId = githubOnboardingIntegrationId({
      selectionId: "int-github-com",
      vcsIntegrationId: "int-ghe",
    });
    const restored = resolvedOnboardingGithubEnterprise({
      factoryId: "factory-1",
      vcsIntegrationId: githubIntegrationId,
      hostFromIntegration: githubIntegrationId === "int-ghe" ? "acme.ghe.com" : "",
    });
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=repo"]}>
        <FirstRunSetup
          model={pageModel(hook.result.current, {
            githubIntegrationId: restored.integrationId,
            githubHost: restored.host,
          })}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-organization-picker")).not.toBeInTheDocument();
  });
});
