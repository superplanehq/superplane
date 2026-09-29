import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingSetupState } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const github = vi.hoisted(() => ({
  data: {
    appConfigured: true,
    identity: undefined as { userId: string; login: string } | undefined,
    repositories: [] as Array<{
      repositoryId: string;
      installationId: string;
      fullName: string;
      defaultBranch: string;
    }>,
    pendingRequests: [] as Array<{ requestId: string; accountLogin: string }>,
    synchronizing: false,
  },
}));

const startInstallation = vi.fn().mockResolvedValue("https://github.com/apps/superplane/installations/new");
const configureInstallation = vi.fn().mockResolvedValue("https://github.com/settings/installations/101");

vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: () => ({
    data: github.data,
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: startInstallation },
    configureInstallation: { mutateAsync: configureInstallation },
  }),
}));

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
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
  useExperimentalFeature: () => ({ has: () => false, isLoading: false, organizationReady: true }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));
vi.mock("./AgentStep", () => ({ AgentStep: () => <div data-testid="agent-step" /> }));

function pageModel(overrides: Partial<OnboardingPageModel> = {}): OnboardingPageModel {
  const { result } = renderHook(() => useOnboardingSetupState("Payments", { simulateDiscovery: false }));
  return {
    setup: result.current,
    hostedAgentReady: true,
    hostedModelsAvailable: true,
    hostedModelsAvailableLoading: false,
    bringYourOwnKey: false,
    bringYourOwnKeyLoading: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    openSection: "vcs",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    selectCatalogRepository: vi.fn().mockResolvedValue(true),
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
    ...overrides,
  };
}

function renderSetup(model: OnboardingPageModel) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=vcs"]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

describe("FirstRunSetup GitHub catalog", () => {
  beforeEach(() => {
    github.data.identity = undefined;
    github.data.repositories = [];
    github.data.pendingRequests = [];
    github.data.synchronizing = false;
  });

  it("shows Connect GitHub when identity is missing", () => {
    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent("Connect GitHub");
  });

  it("returns from GitHub connection at repository selection", async () => {
    const user = userEvent.setup();
    const previousAssign = window.location.assign.bind(window.location);
    const assign = vi.fn();
    window.location.assign = assign;

    try {
      renderSetup(pageModel());
      await user.click(screen.getByTestId("first-run-connect-github"));

      expect(assign).toHaveBeenCalledWith(
        "/auth/github?intent=connect&redirect=%2Forg-1%2Fworkspaces%2FPAY%2Fsetup%3Fstep%3Drepo",
      );
    } finally {
      window.location.assign = previousAssign;
    }
  });

  it("skips installation when cached accessible repositories exist", async () => {
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.repositories = [
      { repositoryId: "77", installationId: "101", fullName: "acme/api", defaultBranch: "main" },
      { repositoryId: "88", installationId: "202", fullName: "example/web", defaultBranch: "trunk" },
    ];
    renderSetup(pageModel());

    await waitFor(() => expect(screen.getByTestId("first-run-choose")).toBeInTheDocument());
    expect(screen.getByTestId("first-run-github-signed-in-as")).toHaveTextContent(
      "You are signed in to GitHub as octocat.",
    );
    expect(screen.getByRole("option", { name: /acme\/api/ })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /example\/web/ })).toBeInTheDocument();
    expect(startInstallation).not.toHaveBeenCalled();
  });

  it("opens repository selection while repositories synchronize", async () => {
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.synchronizing = true;

    renderSetup(pageModel());

    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-repositories-synchronizing")).toHaveTextContent(
      FIRST_RUN_COPY.choose.synchronizing,
    );
  });

  it("selects the numeric catalog repository before continuing", async () => {
    const user = userEvent.setup();
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.repositories = [
      { repositoryId: "77", installationId: "101", fullName: "acme/api", defaultBranch: "main" },
    ];
    const selectCatalogRepository = vi.fn().mockResolvedValue(true);
    const model = pageModel({ selectCatalogRepository });
    model.setup = { ...model.setup, selectedRepo: "acme/api" };
    renderSetup(model);

    expect(await screen.findByRole("option", { name: /acme\/api/ })).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    await waitFor(() =>
      expect(selectCatalogRepository).toHaveBeenCalledWith(
        expect.objectContaining({ repositoryId: "77", fullName: "acme/api" }),
      ),
    );
  });
});
