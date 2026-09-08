import { render, renderHook } from "@testing-library/react";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

export type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

export const setupFixtures: {
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
  accountOrganizations: Array<{ id: string; name: string; slug?: string }>;
} = {
  factory: { id: "factory-1" },
  factories: [],
  accountOrganizations: [],
};

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory: setupFixtures.factory,
    factories: setupFixtures.factories,
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "user-1" } }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

// The install-request recheck and the in-place bind need a query client and
// the network; the flow tests cover the screens only.
vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  useRecheckGitHubInstallRequest: vi.fn(),
}));

export const bindMutate = vi.fn();

vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({ mutate: bindMutate, isPending: false, variables: undefined }),
}));

export const navigateSpy = vi.fn();

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: setupFixtures.accountOrganizations, refetch: vi.fn() }),
}));

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof ReactRouterDom>("react-router");
  return {
    ...actual,
    useNavigate: () => navigateSpy,
  };
});

// The agent step reports organization spend, which this flow test does not use.
vi.mock("./AgentStep", () => ({
  AgentStep: () => <div data-testid="agent-step" />,
}));

export function setupState(): OnboardingSetupApi {
  const { result } = renderHook(() => useOnboardingSetupState("Payments Service", { simulateDiscovery: false }));
  return result.current;
}

export function pageModel(overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  return {
    setup: setupState(),
    hostedAgentReady: false,
    openSection: "issues",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    refreshGithubConnections: vi.fn().mockResolvedValue(undefined),
    githubConnectionsLoading: false,
    requestPrivateGitHubConnect: vi.fn(),
    offersPrivateGitHubAppSetup: false,
    createVcsConnection: vi.fn(),
    selectVcsConnection: vi.fn().mockResolvedValue(true),
    githubConnections: { name: "github", allInstances: [], readyInstances: [] },
    selectedVcsConnectionId: "github-1",
    requestConfigure: vi.fn(),
    integrationDialogs: <></>,
    repositories: ["acme/payments-service"],
    repositoriesLoading: false,
    repositoriesError: null,
    canConfigureWorkspace: true,
    saving: false,
    saveName: vi.fn().mockResolvedValue(true),
    saveRepository: vi.fn().mockResolvedValue(true),
    saveIssues: vi.fn().mockResolvedValue(true),
    finish: vi.fn(),
    ...overrides,
  };
}

export function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

export function resetSetupFixtures() {
  setupFixtures.factory = { id: "factory-1", onboarding: { vcsIntegrationId: "github-1" } };
  setupFixtures.factories = [setupFixtures.factory];
  setupFixtures.accountOrganizations = [{ id: "org-1", name: "Acme" }];
  navigateSpy.mockClear();
  bindMutate.mockReset();
}
