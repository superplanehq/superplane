import { render, renderHook, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

let factory: FactoriesFactory;
let factories: FactoriesFactory[];

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory,
    factories,
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "user-1" } }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  useRecheckGitHubInstallRequest: vi.fn(),
}));

vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({ mutateAsync: vi.fn() }),
}));

const navigateSpy = vi.fn();

let accountOrganizations: Array<{ id: string; name: string; slug?: string }>;

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: accountOrganizations, refetch: vi.fn() }),
}));

vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof ReactRouterDom>("react-router");
  return {
    ...actual,
    useNavigate: () => navigateSpy,
  };
});

vi.mock("./AgentStep", () => ({
  AgentStep: () => <div data-testid="agent-step" />,
}));

function setupState(): OnboardingSetupApi {
  const { result } = renderHook(() => useOnboardingSetupState("Payments Service", { simulateDiscovery: false }));
  return result.current;
}

function pageModel(overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  return {
    setup: setupState(),
    hostedAgentReady: false,
    agentLoading: false,
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

function renderSetup(model: OnboardingPageModel) {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=issues"]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

describe("FirstRunSetup chrome", () => {
  beforeEach(() => {
    factory = { id: "factory-1", key: "PAY", name: "New workspace", onboarding: { vcsIntegrationId: "github-1" } };
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
    navigateSpy.mockClear();
  });

  it("shows the workspace switch when another workspace exists", () => {
    factories = [factory, { id: "factory-2", key: "CORE", name: "Core" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-workspace-switch")).toHaveTextContent("NW");
    expect(screen.getByTestId("first-run-workspace-switch")).toHaveAccessibleName(/Switch workspace, New workspace/);
    expect(screen.queryByTestId("first-run-organization-switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
  });

  it("opens the other workspace from the switch menu", async () => {
    factories = [factory, { id: "factory-2", key: "CORE", name: "Core", lines: [{ id: "line-core" }] }];
    const user = userEvent.setup();

    renderSetup(pageModel());

    await user.click(screen.getByTestId("first-run-workspace-switch"));
    await user.click(screen.getByTestId("first-run-workspace-option-factory-2"));

    expect(navigateSpy).toHaveBeenCalledWith("/org-1/workspaces/core/lines/line-core");
  });

  it("shows Log out and the organization switch when another organization exists", () => {
    factories = [factory];
    accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co" },
    ];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-workspace-switch")).not.toBeInTheDocument();
  });

  it("shows both switches when another workspace and another organization exist", () => {
    factories = [factory, { id: "factory-2", key: "CORE", name: "Core" }];
    accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co" },
    ];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-workspace-switch")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
  });

  it("opens the current organization from the switch menu so the user can leave setup", async () => {
    accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co" },
    ];
    const user = userEvent.setup();

    renderSetup(pageModel());

    await user.click(screen.getByTestId("first-run-organization-switch"));
    await user.click(screen.getByTestId("first-run-organization-option-org-1"));

    expect(navigateSpy).toHaveBeenCalledWith("/org-1");
  });

  it("keeps Log out and hides both switches with a single org and single workspace", () => {
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-organization-switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-workspace-switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
  });
});
