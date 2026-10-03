import { act, render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import { readOnboardingGitHubConnect, writeOnboardingGitHubConnect } from "./onboardingGitHubConnect";
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
  error: null as unknown,
  calls: [] as Array<{ organizationId: string; options?: { poll?: boolean } }>,
}));

const showErrorToast = vi.hoisted(() => vi.fn());
const startInstallation = vi.fn().mockResolvedValue("https://github.com/apps/superplane/installations/new");
const configureInstallation = vi.fn().mockResolvedValue("https://github.com/settings/installations/101");

vi.mock("@/lib/toast", () => ({
  showErrorToast,
}));

vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: (organizationId: string, options?: { poll?: boolean }) => {
    github.calls.push({ organizationId, options });
    return {
      data: github.data,
      isPending: false,
      error: github.error,
      startInstallation: { mutateAsync: startInstallation },
      configureInstallation: { mutateAsync: configureInstallation },
      selectIdentity: { mutateAsync: vi.fn() },
    };
  },
}));

vi.mock("./first-run/useFirstRunAnalysis", () => ({
  useFirstRunAnalysis: () => ({
    progress: { total: 0, scored: 0, ready: 0, stageIndex: 0, empty: true },
    sourceName: "GitHub issues",
    failed: false,
  }),
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
    customProvider: false,
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

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=vcs") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

const runningDestination = {
  organizationId: "acme",
  factoryKey: "PAY",
  lineId: "line-1",
};

describe("FirstRunSetup GitHub catalog", () => {
  beforeEach(() => {
    github.data.identity = undefined;
    github.data.repositories = [];
    github.data.pendingRequests = [];
    github.data.synchronizing = false;
    github.error = null;
    github.calls = [];
    showErrorToast.mockReset();
    localStorage.clear();
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
        "/auth/github?intent=connect&redirect=%2Forg-1%2Fworkspaces%2FPAY%2Fsetup%3Fstep%3Drepo%26githubConnected%3D1",
      );
    } finally {
      window.location.assign = previousAssign;
    }
  });

  it("enables Connect GitHub again when the browser restores the page from its cache", async () => {
    const user = userEvent.setup();
    const previousAssign = window.location.assign.bind(window.location);
    window.location.assign = vi.fn();

    try {
      renderSetup(pageModel());
      await user.click(screen.getByTestId("first-run-connect-github"));
      expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent(FIRST_RUN_COPY.connect.openingGitHub);

      const restored = new Event("pageshow");
      Object.defineProperty(restored, "persisted", { value: true });
      act(() => {
        window.dispatchEvent(restored);
      });

      expect(screen.getByTestId("first-run-connect-github")).toHaveTextContent("Connect GitHub");
      expect(screen.getByTestId("first-run-connect-github")).toBeEnabled();
    } finally {
      window.location.assign = previousAssign;
    }
  });

  it("asks the user to connect GitHub when the identity only comes from sign-in", async () => {
    const user = userEvent.setup();
    const previousAssign = window.location.assign.bind(window.location);
    const assign = vi.fn();
    window.location.assign = assign;
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.repositories = [
      { repositoryId: "77", installationId: "101", fullName: "acme/api", defaultBranch: "main" },
    ];

    try {
      renderSetup(pageModel(), "/org-1/workspaces/PAY/setup?step=repo");

      expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
      expect(screen.queryByTestId("first-run-choose")).not.toBeInTheDocument();

      await user.click(screen.getByTestId("first-run-connect-github"));
      expect(assign).toHaveBeenCalled();
      expect(readOnboardingGitHubConnect("account-1", "factory-1")).toBe(false);
    } finally {
      window.location.assign = previousAssign;
    }
  });

  it("saves the GitHub connection when GitHub returns after a successful connection", async () => {
    github.data.identity = { userId: "9", login: "octocat" };

    renderSetup(pageModel(), "/org-1/workspaces/PAY/setup?step=repo&githubConnected=1");

    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
    await waitFor(() => expect(readOnboardingGitHubConnect("account-1", "factory-1")).toBe(true));
  });

  it("does not use a GitHub connection that another person saved in this browser", () => {
    writeOnboardingGitHubConnect("account-2", "factory-1");
    github.data.identity = { userId: "9", login: "octocat" };

    renderSetup(pageModel(), "/org-1/workspaces/PAY/setup?step=repo");

    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
  });

  it("stays on repository selection when the user clears a saved repository", async () => {
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.repositories = [
      { repositoryId: "77", installationId: "101", fullName: "acme/api", defaultBranch: "main" },
      { repositoryId: "88", installationId: "202", fullName: "octo/web", defaultBranch: "main" },
    ];
    const path = "/org-1/workspaces/PAY/setup?step=repo";
    const model = pageModel();
    const view = renderSetup({ ...model, setup: { ...model.setup, selectedRepo: "acme/api" } }, path);
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();

    view.rerender(
      <MemoryRouter initialEntries={[path]}>
        <FirstRunSetup model={{ ...model, setup: { ...model.setup, selectedRepo: null } }} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("first-run-choose")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect")).not.toBeInTheDocument();
  });

  it("skips installation when cached accessible repositories exist", async () => {
    const user = userEvent.setup();
    writeOnboardingGitHubConnect("account-1", "factory-1");
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
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.choose.useOrganization("example") })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.choose.useOrganization("acme") }));
    expect(screen.getByRole("option", { name: /acme\/api/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /example\/web/ })).not.toBeInTheDocument();
    expect(startInstallation).not.toHaveBeenCalled();
  });

  it("opens repository selection while repositories synchronize", async () => {
    writeOnboardingGitHubConnect("account-1", "factory-1");
    github.data.identity = { userId: "9", login: "octocat" };
    github.data.synchronizing = true;

    renderSetup(pageModel());

    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-repositories-synchronizing")).toHaveTextContent(
      FIRST_RUN_COPY.choose.synchronizingOrganizations,
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

  describe("without a requested step", () => {
    const setupPath = "/org-1/workspaces/PAY/setup";

    it("opens the welcome screen before GitHub is connected", () => {
      renderSetup(pageModel(), setupPath);

      expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    });

    it("opens the welcome screen when the GitHub identity only comes from sign-in", () => {
      github.data.identity = { userId: "9", login: "octocat" };

      renderSetup(pageModel(), setupPath);

      expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    });

    it("resumes at repository selection after a GitHub setup callback", async () => {
      writeOnboardingGitHubConnect("account-1", "factory-1");
      github.data.identity = { userId: "9", login: "octocat" };

      renderSetup(pageModel(), setupPath);

      expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
    });

    it("goes back to the connect screen from repository selection and stays there", async () => {
      const user = userEvent.setup();
      writeOnboardingGitHubConnect("account-1", "factory-1");
      github.data.identity = { userId: "9", login: "octocat" };

      renderSetup(pageModel(), setupPath);
      await user.click(await screen.findByTestId("first-run-back"));

      await waitFor(() => expect(screen.getByTestId("first-run-connect")).toBeInTheDocument());
      expect(screen.queryByTestId("first-run-choose")).not.toBeInTheDocument();

      await user.click(screen.getByTestId("first-run-back"));
      expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();

      await user.click(screen.getByTestId("first-run-get-started"));
      expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
    });
  });

  it("stops GitHub onboarding polling once the running workspace step is shown", () => {
    renderSetup(pageModel({ provisionedDestination: runningDestination }));

    expect(screen.getByText(FIRST_RUN_COPY.analysis.headline)).toBeInTheDocument();
    expect(github.calls.length).toBeGreaterThan(0);
    expect(github.calls.every((call) => call.options?.poll === false)).toBe(true);
  });

  it("does not toast Not Found after setup finishes, and keeps the empty import", async () => {
    const model = pageModel({ provisionedDestination: runningDestination });
    const view = renderSetup(model);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.emptyImport("GitHub issues"))).toBeInTheDocument();
    expect(showErrorToast).not.toHaveBeenCalled();

    github.error = { message: "Not Found" };
    view.rerender(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=vcs"]}>
        <FirstRunSetup model={model} />
      </MemoryRouter>,
    );

    await waitFor(() => expect(github.calls.length).toBeGreaterThan(1));
    expect(showErrorToast).not.toHaveBeenCalled();
    expect(screen.queryByText("Not Found")).not.toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.emptyImport("GitHub issues"))).toBeInTheDocument();
  });

  it("does not show the raw Not Found status before setup finishes", async () => {
    github.error = { message: "Not Found" };
    renderSetup(pageModel());

    expect(await screen.findByText("SuperPlane could not load GitHub access")).toBeInTheDocument();
    expect(screen.queryByText("Not Found")).not.toBeInTheDocument();
    expect(showErrorToast).toHaveBeenCalledWith("Failed to load repositories");
    expect(showErrorToast).not.toHaveBeenCalledWith("Not Found");
  });

  it("still shows a real GitHub access error before setup finishes", async () => {
    github.error = { message: "GitHub token expired" };
    renderSetup(pageModel());

    expect(await screen.findByText("GitHub token expired")).toBeInTheDocument();
    expect(showErrorToast).toHaveBeenCalledWith("GitHub token expired");
  });
});
