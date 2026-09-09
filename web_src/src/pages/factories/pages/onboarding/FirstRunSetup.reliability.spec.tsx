import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

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
    initialOnboarding: false,
    provisionedDestination: null,
    ...overrides,
  };
}

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
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

describe("FirstRunSetup reliability", () => {
  it("shows local placeholders while GitHub accounts and repositories load", () => {
    const accounts = renderSetup(
      pageModel({ openSection: "vcs", githubConnectionsLoading: true }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );
    expect(screen.getByTestId("first-run-connect-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();

    accounts.unmount();
    renderSetup(
      pageModel({ openSection: "repo", repositories: ["octo/stale-repo"], repositoriesLoading: true }),
      "/org-1/workspaces/PAY/setup?step=repo",
    );
    expect(screen.getByTestId("first-run-repositories-loading")).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /octo\/stale-repo/ })).not.toBeInTheDocument();
  });

  it("locks the connect screen while GitHub opens and ignores a second click", async () => {
    const user = userEvent.setup();
    const navigation = deferred<boolean>();
    const requestConnect = vi.fn(() => navigation.promise);
    renderSetup(pageModel({ openSection: "vcs", requestConnect }), "/org-1/workspaces/PAY/setup?step=vcs");

    const connect = screen.getByTestId("first-run-connect-github");
    await user.click(connect);
    expect(connect).toHaveTextContent(FIRST_RUN_COPY.connect.openingGitHub);
    expect(connect).toBeDisabled();
    expect(screen.getByTestId("first-run-back")).toBeDisabled();
    await user.click(connect);
    expect(requestConnect).toHaveBeenCalledTimes(1);

    navigation.resolve(true);
    await waitFor(() => expect(connect).toHaveTextContent(FIRST_RUN_COPY.connect.connectGitHub));
  });

  it("keeps repository and ticket screens locked until their saves finish", async () => {
    const user = userEvent.setup();
    const repositorySaved = deferred<boolean>();
    const setup = { ...setupState(), selectedRepo: "acme/payments-service" };
    const repository = renderSetup(
      pageModel({ openSection: "repo", saveRepository: vi.fn(() => repositorySaved.promise), setup }),
      "/org-1/workspaces/PAY/setup?step=repo",
    );
    const repositoryContinue = screen.getByTestId("first-run-continue-to-tickets");
    await user.click(repositoryContinue);
    expect(repositoryContinue).toHaveTextContent(FIRST_RUN_COPY.choose.saving);
    expect(screen.getByTestId("first-run-back")).toBeDisabled();
    repositorySaved.resolve(true);
    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();

    repository.unmount();
    const issuesSaved = deferred<boolean>();
    const saveIssues = vi.fn(() => issuesSaved.promise);
    renderSetup(pageModel({ openSection: "issues", saveIssues }));
    const issuesContinue = screen.getByTestId("first-run-analyze-tickets");
    await user.click(issuesContinue);
    expect(issuesContinue).toHaveTextContent(FIRST_RUN_COPY.tickets.saving);
    await user.click(issuesContinue);
    expect(saveIssues).toHaveBeenCalledTimes(1);
    issuesSaved.resolve(true);
    expect(await screen.findByTestId("first-run-agent")).toBeInTheDocument();
  });

  it("shows only the current user's GitHub account picker", () => {
    const picker = (userId: string) =>
      githubConnection("int-1", {
        startedByUserID: userId,
        state: "csrf",
        githubApp: { slug: "superplane" },
        pendingInstallations: [{ id: "11", accountLogin: "acme" }],
      });
    const mine = renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([picker("user-1")]) }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") })).toBeInTheDocument();

    mine.unmount();
    renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([picker("another-user")]) }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
  });

  it("opens the waiting screen from a GitHub request return", () => {
    const request = githubConnection("int-1", { startedByUserID: "user-1", installRequested: true });
    renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([request]) }),
      "/org-1/workspaces/PAY/setup?githubSetup=request&githubIntegrationId=int-1",
    );
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent(
      FIRST_RUN_COPY.connect.installRequested,
    );
  });

  it("uses server metadata for every requested organization", () => {
    const request = githubConnection("int-1", {
      startedByUserID: "user-1",
      installRequests: [
        { id: "1", accountLogin: "acme", requesterLogin: "member" },
        { id: "2", accountLogin: "octo", requesterLogin: "member" },
      ],
    });
    renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([request]) }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubOrg=wrong&githubIntegrationId=int-1",
    );
    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("octo");
    expect(screen.queryByText("wrong")).not.toBeInTheDocument();
  });

  it("replaces the waiting state with the approved account picker", () => {
    const waiting = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      installRequests: [{ id: "1", accountLogin: "acme", requesterLogin: "member" }],
    });
    const approved = githubConnection("int-1", {
      startedByUserID: "user-1",
      state: "csrf",
      githubApp: { slug: "superplane" },
      installRequests: [],
      pendingInstallations: [{ id: "11", accountLogin: "acme" }],
    });
    const view = renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([waiting]) }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubIntegrationId=int-1",
    );
    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();

    view.rerender(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=vcs"]}>
        <FirstRunSetup model={pageModel({ openSection: "vcs", githubConnections: githubConnections([approved]) })} />
      </MemoryRouter>,
    );
    expect(screen.queryByTestId("first-run-github-install-requested")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") })).toBeInTheDocument();
  });

  it("does not borrow an organization or request from another member", () => {
    const mine = githubConnection("mine", { startedByUserID: "user-1", installRequested: true });
    const theirs = githubConnection("theirs", {
      startedByUserID: "another-user",
      installRequested: true,
      installRequestedAccount: "wrong-org",
    });
    renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([mine, theirs]) }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );
    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
    expect(screen.queryByText("wrong-org")).not.toBeInTheDocument();
  });

  it("hides another member's request when the current user has none", () => {
    const theirs = githubConnection("theirs", {
      startedByUserID: "another-user",
      installRequested: true,
      installRequestedAccount: "their-org",
    });
    renderSetup(
      pageModel({ openSection: "vcs", githubConnections: githubConnections([theirs]) }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );
    expect(screen.queryByTestId("first-run-github-install-requested")).not.toBeInTheDocument();
  });

  it("ignores a stale request marker after server metadata loads", () => {
    renderSetup(
      pageModel({ openSection: "vcs", githubConnectionsLoading: false }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubOrg=stale-org",
    );
    expect(screen.queryByTestId("first-run-github-install-requested")).not.toBeInTheDocument();
    expect(screen.queryByText("stale-org")).not.toBeInTheDocument();
  });
});
