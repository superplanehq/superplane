import { act, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FirstRunSetup } from "./FirstRunSetup";
import type { IntegrationId } from "./onboardingFixtures";
import { writeOnboardingGithubEnterprise } from "./onboardingGithubEnterprise";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const connectGitHubEnterprise = vi.hoisted(() => vi.fn().mockResolvedValue(true));

vi.mock("./useFirstRunGitHubEnterprise", () => ({
  useFirstRunGitHubEnterprise: () => ({ connect: connectGitHubEnterprise }),
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
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
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
    connectGitHubEnterprise.mockClear();
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

    await waitFor(() => expect(connectGitHubEnterprise).toHaveBeenCalled());
    expect(connectGitHubEnterprise).toHaveBeenCalledWith("acme.ghe.com", expect.stringContaining("step=repo"));
  });

  it("lists repositories from the GitHub Enterprise integration", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    const connected = new Set<IntegrationId>(["github"]);
    Object.defineProperty(hook.result.current, "connected", { get: () => connected });
    hook.result.current.selectVcsHost("github");

    renderSetup(hook.result.current, "/org-1/workspaces/PAY/setup?step=repo");

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    expect(screen.getByText("acme/api")).toBeInTheDocument();
  });

  it("resumes repository choice after GitHub App setup before GitHub is marked ready", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=repo"]}>
        <FirstRunSetup model={pageModel(hook.result.current, { githubIntegrationId: "" })} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
    expect(screen.getByText("acme/api")).toBeInTheDocument();
  });

  it("resumes Enterprise from the saved integration host after the tab is closed", () => {
    const hook = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
    hook.result.current.selectVcsHost("github");

    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=repo"]}>
        <FirstRunSetup
          model={pageModel(hook.result.current, { githubIntegrationId: "int-ghe", githubHost: "acme.ghe.com" })}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("first-run-github-enterprise-choose")).toBeInTheDocument();
  });
});
