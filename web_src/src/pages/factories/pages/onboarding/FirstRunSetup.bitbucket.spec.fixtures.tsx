import { render, screen, within } from "@testing-library/react";
import type userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import { FirstRunSetup } from "./FirstRunSetup";
import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

export const resources = vi.hoisted(() => ({
  calls: [] as Array<{ integrationId: string; type: string; enabled?: boolean }>,
}));

export const bitbucketOnboarding = vi.hoisted(() => ({
  loaded: true,
  refetch: (() => undefined) as () => unknown,
  providerConfigured: false,
  identity: undefined as { login?: string; providerUserId?: string } | undefined,
  identities: [] as Array<{ login?: string; providerUserId?: string }>,
  selectIdentity: (async () => undefined) as (providerUserId: string) => Promise<void>,
  repositories: [] as Array<{ fullName?: string }>,
  installedWorkspaces: [] as Array<{ slug?: string }>,
  isPending: false,
  error: null as unknown,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: (
    _organizationId: string,
    integrationId: string,
    type: string,
    _parameters?: Record<string, string>,
    options?: { enabled?: boolean },
  ) => {
    resources.calls.push({ integrationId, type, enabled: options?.enabled });
    return {
      data: options?.enabled ? [{ name: "acme-team/api" }, { name: "acme-team/web" }] : undefined,
      isLoading: false,
      isError: false,
    };
  },
}));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: () => ({
    data: bitbucketOnboarding.loaded
      ? {
          providerConfigured: bitbucketOnboarding.providerConfigured,
          identity: bitbucketOnboarding.identity,
          identities: bitbucketOnboarding.identities,
          repositories: bitbucketOnboarding.repositories,
          installedWorkspaces: bitbucketOnboarding.installedWorkspaces,
        }
      : undefined,
    isPending: bitbucketOnboarding.isPending,
    error: bitbucketOnboarding.error,
    refetch: bitbucketOnboarding.refetch,
    startInstallation: { mutateAsync: vi.fn() },
    selectIdentity: { mutateAsync: bitbucketOnboarding.selectIdentity },
  }),
}));

vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: () => ({
    data: {
      providerConfigured: true,
      accountConnectionRequired: true,
      identity: undefined,
      repositories: [],
      pendingRequests: [],
      synchronizing: false,
    },
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

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));
vi.mock("./AgentStep", () => ({ AgentStep: () => <div data-testid="agent-step" /> }));

export function resetBitbucketFirstRunHarness() {
  resources.calls = [];
  bitbucketOnboarding.loaded = true;
  bitbucketOnboarding.refetch = vi.fn();
  bitbucketOnboarding.providerConfigured = false;
  bitbucketOnboarding.identity = undefined;
  bitbucketOnboarding.identities = [];
  bitbucketOnboarding.selectIdentity = vi.fn().mockResolvedValue(undefined);
  bitbucketOnboarding.repositories = [];
  bitbucketOnboarding.installedWorkspaces = [];
  bitbucketOnboarding.isPending = false;
  bitbucketOnboarding.error = null;
  localStorage.clear();
  sessionStorage.clear();
}

function pageModel(setup: OnboardingSetupApi, overrides: Partial<OnboardingPageModel>): OnboardingPageModel {
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

function StatefulSetup({
  connected,
  overrides,
}: {
  connected: Set<IntegrationId>;
  overrides: Partial<OnboardingPageModel>;
}) {
  const setup = useOnboardingSetupState("Payments", { connected, simulateDiscovery: false });
  return <FirstRunSetup model={pageModel(setup, overrides)} />;
}

export function bitbucketSetupElement(connected: Set<IntegrationId>, overrides: Partial<OnboardingPageModel> = {}) {
  return (
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
      <StatefulSetup connected={connected} overrides={overrides} />
    </MemoryRouter>
  );
}

export function renderSetup(connected: Set<IntegrationId>, overrides: Partial<OnboardingPageModel> = {}) {
  return render(bitbucketSetupElement(connected, overrides));
}

export async function chooseBitbucket(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("first-run-get-started"));
  await user.click(within(screen.getByTestId("first-run-host-bitbucket")).getByRole("button"));
  const workspace = screen.queryByRole("button", { name: "Use acme-team" });
  if (workspace) await user.click(workspace);
}
