import { act, render, renderHook, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, OrganizationsIntegration } from "@/api-client";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;
const factory: FactoriesFactory = {
  id: "factory-1",
  key: "PAY",
  name: "New workspace",
  onboarding: { vcsIntegrationId: "github-1" },
};

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory,
    factories: [factory],
  }),
}));
vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));
vi.mock("@/hooks/useMe", () => ({ useMe: () => ({ data: { id: "user-1" } }) }));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => true, enabledExperimentalFeatures: [], isLoading: false }),
}));
vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: [{ id: "org-1", name: "Acme" }] }),
}));
vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({ useRecheckGitHubInstallRequest: vi.fn() }));
vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({ mutateAsync: vi.fn().mockResolvedValue(undefined) }),
}));
vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));
vi.mock("./AgentStep", () => ({ AgentStep: () => <div data-testid="agent-step" /> }));

function setupState(): OnboardingSetupApi {
  return renderHook(() => useOnboardingSetupState("Payments Service", { simulateDiscovery: false })).result.current;
}

function pageModel(overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  return {
    setup: setupState(),
    hostedAgentReady: false,
    hostedModelsAvailable: false,
    hostedModelsAvailableLoading: false,
    bringYourOwnKey: false,
    bringYourOwnKeyLoading: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    openSection: "vcs",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    refreshGithubConnections: vi.fn().mockResolvedValue(undefined),
    syncGithubConnection: vi.fn().mockResolvedValue(undefined),
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
    provisionedDestination: null,
    githubOwner: undefined,
    jiraIntegrationId: "",
    jiraProjectId: "",
    setJiraProjectId: vi.fn(),
    jiraCompletion: { jiraMoveOnComplete: true, jiraCompletionColumn: "" },
    setJiraCompletion: vi.fn(),
    jiraProjects: [],
    jiraProjectsLoading: false,
    jiraProjectsError: false,
    retryJiraProjects: vi.fn(),
    ...overrides,
  };
}

function renderSetup(model: OnboardingPageModel, path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve };
}

function githubConnections(instances: OrganizationsIntegration[]) {
  return { name: "github", readyInstances: [], allInstances: instances };
}

function githubConnection(id: string, metadata: Record<string, unknown>): OrganizationsIntegration {
  return { metadata: { id, integrationName: "github" }, status: { state: "pending", metadata } };
}

describe("FirstRunSetup GitHub callback synchronization", () => {
  it("waits for a fresh GitHub account after a completed callback", async () => {
    const refreshed = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [{ id: "11", accountLogin: "acme", repositories: [{ id: "101", name: "acme/api" }] }],
    });
    const model = pageModel({
      syncGithubConnection: vi.fn(async () => {
        model.githubConnections = githubConnections([refreshed]);
        return refreshed;
      }),
    });

    renderSetup(model, "/org-1/workspaces/PAY/setup?githubSetup=complete&githubIntegrationId=int-1");

    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
    expect(await screen.findByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") })).toBeInTheDocument();
    expect(model.syncGithubConnection).toHaveBeenCalledWith("int-1");
  });

  it("does not flash cached GitHub accounts after a completed callback", async () => {
    const stale = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [
        { id: "11", accountLogin: "existing", repositories: [{ id: "101", name: "existing/api" }] },
      ],
    });
    const refreshed = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [
        { id: "11", accountLogin: "existing", repositories: [{ id: "101", name: "existing/api" }] },
        { id: "22", accountLogin: "new-org", repositories: [{ id: "202", name: "new-org/api" }] },
      ],
    });
    const sync = deferred<OrganizationsIntegration>();
    const model = pageModel({
      githubConnections: githubConnections([stale]),
      syncGithubConnection: vi.fn(() => sync.promise),
    });
    const view = renderSetup(
      model,
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=complete&githubIntegrationId=int-1",
    );

    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.useAccount("existing"))).not.toBeInTheDocument();

    model.githubConnections = githubConnections([refreshed]);
    view.rerender(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=vcs"]}>
        <FirstRunSetup model={model} />
      </MemoryRouter>,
    );
    await act(async () => sync.resolve(refreshed));

    expect(await screen.findByText(FIRST_RUN_COPY.connect.useAccount("existing"))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.connect.useAccount("new-org"))).toBeInTheDocument();
  });

  it("refreshes a request callback before showing cached accounts", async () => {
    const stale = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [
        { id: "11", accountLogin: "existing", repositories: [{ id: "101", name: "existing/api" }] },
      ],
    });
    const refreshed = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [
        { id: "11", accountLogin: "existing", repositories: [{ id: "101", name: "existing/api" }] },
      ],
      installRequests: [{ id: "1", accountLogin: "requested-org", requesterLogin: "member" }],
    });
    const model = pageModel({
      githubConnections: githubConnections([stale]),
      syncGithubConnection: vi.fn(async () => {
        model.githubConnections = githubConnections([refreshed]);
        return refreshed;
      }),
    });

    renderSetup(model, "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubIntegrationId=int-1");

    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.connect.useAccount("existing"))).not.toBeInTheDocument();
    expect(await screen.findByText(FIRST_RUN_COPY.connect.useAccount("existing"))).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent("requested-org");
  });

  it("lets the user retry a failed callback synchronization", async () => {
    const refreshed = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      pendingInstallations: [{ id: "11", accountLogin: "acme", repositories: [{ id: "101", name: "acme/api" }] }],
    });
    const model = pageModel();
    model.syncGithubConnection = vi
      .fn<OnboardingPageModel["syncGithubConnection"]>()
      .mockRejectedValueOnce(new Error("network"))
      .mockImplementationOnce(async () => {
        model.githubConnections = githubConnections([refreshed]);
        return refreshed;
      });
    const user = userEvent.setup();

    renderSetup(model, "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=complete&githubIntegrationId=int-1");

    expect(await screen.findByRole("alert")).toHaveTextContent(FIRST_RUN_COPY.connect.refreshError);
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.tryAgain }));

    expect(await screen.findByText(FIRST_RUN_COPY.connect.useAccount("acme"))).toBeInTheDocument();
    expect(model.syncGithubConnection).toHaveBeenCalledTimes(2);
  });
});
