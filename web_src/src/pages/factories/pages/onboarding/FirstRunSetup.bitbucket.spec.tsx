import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FEATURE_FACTORY_BITBUCKET } from "@/lib/experimentalFeatures";

import { FirstRunSetup } from "./FirstRunSetup";
import { FirstRunBitbucketForgeScreen } from "./first-run/FirstRunBitbucketForgeScreen";
import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const resources = vi.hoisted(() => ({
  calls: [] as Array<{ integrationId: string; type: string; enabled?: boolean }>,
}));

const experimental = vi.hoisted(() => ({
  enabled: new Set<string>(),
  loading: false,
}));

const bitbucketOnboarding = vi.hoisted(() => ({
  loaded: true,
  refetch: (() => undefined) as () => unknown,
  providerConfigured: false,
  identity: undefined as { login?: string; providerUserId?: string } | undefined,
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
          repositories: bitbucketOnboarding.repositories,
          installedWorkspaces: bitbucketOnboarding.installedWorkspaces,
        }
      : undefined,
    isPending: bitbucketOnboarding.isPending,
    error: bitbucketOnboarding.error,
    refetch: bitbucketOnboarding.refetch,
    startInstallation: { mutateAsync: vi.fn() },
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

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (id: string) => experimental.enabled.has(id),
    isLoading: experimental.loading,
    organizationReady: true,
  }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));
vi.mock("./AgentStep", () => ({ AgentStep: () => <div data-testid="agent-step" /> }));

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

function renderSetup(connected: Set<IntegrationId>, overrides: Partial<OnboardingPageModel> = {}) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
      <StatefulSetup connected={connected} overrides={overrides} />
    </MemoryRouter>,
  );
}

async function chooseBitbucket(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("first-run-get-started"));
  await user.click(within(screen.getByTestId("first-run-host-bitbucket")).getByRole("button"));
  const workspace = screen.queryByRole("button", { name: "Use acme-team" });
  if (workspace) await user.click(workspace);
}

describe("FirstRunSetup Bitbucket", () => {
  beforeEach(() => {
    resources.calls = [];
    bitbucketOnboarding.loaded = true;
    bitbucketOnboarding.refetch = vi.fn();
    bitbucketOnboarding.providerConfigured = false;
    bitbucketOnboarding.identity = undefined;
    bitbucketOnboarding.repositories = [];
    bitbucketOnboarding.installedWorkspaces = [];
    bitbucketOnboarding.isPending = false;
    bitbucketOnboarding.error = null;
    experimental.enabled = new Set([FEATURE_FACTORY_BITBUCKET]);
    experimental.loading = false;
    localStorage.clear();
  });

  it("chooses a workspace before a repository and clears the selection when switching workspaces", async () => {
    const user = userEvent.setup();
    function WorkspaceChoice() {
      const [repository, setRepository] = useState<string | null>(null);
      return (
        <FirstRunBitbucketForgeScreen
          phase="choose"
          connectHref=""
          repositories={["acme-team/api", "other-team/web"]}
          installedWorkspaces={["empty-team"]}
          selectedRepository={repository}
          chrome={{ stepIndex: 2, onBack: vi.fn() }}
          onGrantAccess={vi.fn()}
          onSelectRepository={setRepository}
          onClearRepository={() => setRepository(null)}
          onContinue={vi.fn()}
        />
      );
    }
    render(<WorkspaceChoice />);

    expect(screen.getByRole("heading", { name: "Choose a Bitbucket workspace" })).toBeInTheDocument();
    expect(screen.queryByRole("option")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use empty-team" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Use acme-team" }));
    expect(screen.queryByRole("option", { name: /other-team\/web/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /acme-team\/api/ }));
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeEnabled();

    await user.click(screen.getByTestId("first-run-back"));
    await user.click(screen.getByRole("button", { name: "Use other-team" }));
    expect(screen.queryByRole("option", { name: /acme-team\/api/ })).not.toBeInTheDocument();
    expect(screen.getByRole("option", { name: /other-team\/web/ })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
  });

  it("offers installation on another workspace when repositories are already available", async () => {
    const user = userEvent.setup();
    const onGrantAccess = vi.fn();
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        selectedRepository="acme-team/api"
        onGrantAccess={onGrantAccess}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Install on another workspace" }));

    expect(onGrantAccess).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
    expect(screen.getByText("You need write access to a repository to see it here.")).toBeInTheDocument();
    expect(screen.getByText("Do not see your repository?")).toBeInTheDocument();
  });

  it("shows the workspace hint with a subtle install link before a workspace is chosen", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        installedWorkspaces={["acme-team"]}
        selectedRepository={null}
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByText("You need write access to at least one repository in a workspace.")).toBeInTheDocument();
    expect(screen.getByText("Do not see your Bitbucket workspace?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Install on another workspace" })).toHaveClass("underline");
  });

  it("keeps Bitbucket unavailable until the organization enables it", async () => {
    experimental.enabled = new Set();
    const user = userEvent.setup();
    renderSetup(new Set());

    await user.click(screen.getByTestId("first-run-get-started"));

    const bitbucket = screen.getByTestId("first-run-host-bitbucket");
    expect(within(bitbucket).getByRole("button")).toBeDisabled();
    expect(bitbucket).toHaveTextContent("Bitbucket is not available yet.");
    expect(screen.queryByTestId("first-run-bitbucket-connect")).not.toBeInTheDocument();
  });

  it("offers GitHub and Bitbucket on the host screen and keeps GitLab disabled", async () => {
    const user = userEvent.setup();
    renderSetup(new Set());

    await user.click(screen.getByTestId("first-run-get-started"));

    expect(within(screen.getByTestId("first-run-host-github")).getByRole("button")).toBeEnabled();
    expect(within(screen.getByTestId("first-run-host-bitbucket")).getByRole("button")).toBeEnabled();
    expect(within(screen.getByTestId("first-run-host-gitlab")).getByRole("button")).toBeDisabled();
  });

  it("asks for a Bitbucket connection without a GitHub identity", async () => {
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);
    await user.click(screen.getByTestId("first-run-connect-bitbucket"));

    expect(requestConnect).toHaveBeenCalledWith("bitbucket");
    expect(screen.queryByTestId("first-run-connect")).not.toBeInTheDocument();
  });

  it("connects a Bitbucket account with OAuth when Forge is configured", async () => {
    bitbucketOnboarding.providerConfigured = true;
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);
    const link = screen.getByTestId("first-run-bitbucket-oauth");

    expect(link).toHaveAttribute("href", expect.stringContaining("/auth/bitbucket?intent=connect"));
    expect(requestConnect).not.toHaveBeenCalled();
    expect(
      screen.getByText("Connect your Bitbucket account. SuperPlane uses it to find repositories you can open."),
    ).toBeInTheDocument();
  });

  it("shows an existing installation without asking to install again when no repositories are available", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    bitbucketOnboarding.installedWorkspaces = [{ slug: "acme-team" }];
    const user = userEvent.setup();
    renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByText("SuperPlane is installed on your Bitbucket workspace.")).toBeInTheDocument();
    expect(screen.getByText("Workspace: acme-team")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-install")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-continue-to-tickets")).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Check repositories" }));
    expect(bitbucketOnboarding.refetch).toHaveBeenCalled();
  });

  it("keeps checking when an installation is not yet detected", async () => {
    bitbucketOnboarding.providerConfigured = true;
    bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
    const user = userEvent.setup();
    const { rerender } = renderSetup(new Set());

    await chooseBitbucket(user);

    expect(screen.getByRole("status")).toHaveTextContent("SuperPlane checks automatically for installed workspaces.");
    expect(screen.getByRole("heading", { name: "Choose a Bitbucket workspace" })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-stepper")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-waiting")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Check again" })).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-bitbucket-install")).toBeEnabled();

    bitbucketOnboarding.installedWorkspaces = [{ slug: "acme-team" }];
    bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
    rerender(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
        <StatefulSetup connected={new Set()} overrides={{}} />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Use acme-team" }));

    expect(await screen.findByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-grant")).not.toBeInTheDocument();
  });

  it("offers Check again after the installation attempt times out", async () => {
    const user = userEvent.setup();
    const onRetryLookup = vi.fn();
    render(
      <FirstRunBitbucketForgeScreen
        phase="grant"
        connectHref=""
        repositories={[]}
        installedWorkspaces={[]}
        selectedRepository={null}
        attemptTimedOut
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onGrantAccess={vi.fn()}
        onRetryLookup={onRetryLookup}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByText(/SuperPlane did not detect a new workspace/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Check again" }));
    expect(onRetryLookup).toHaveBeenCalledTimes(1);
  });

  it("makes the Bitbucket wait obvious once the installation starts", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="grant"
        connectHref=""
        repositories={[]}
        installedWorkspaces={[]}
        selectedRepository={null}
        attemptActive
        chrome={{ stepIndex: 2, onBack: vi.fn() }}
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent("Checking for the new workspace…");
    expect(screen.getByTestId("first-run-bitbucket-waiting")).toHaveTextContent(
      "This page updates automatically when Bitbucket confirms the installation.",
    );
  });

  it("shows the checking status while a second installation is in flight", async () => {
    render(
      <FirstRunBitbucketForgeScreen
        phase="choose"
        connectHref=""
        repositories={["acme-team/api"]}
        selectedRepository="acme-team/api"
        attemptActive
        onGrantAccess={vi.fn()}
        onSelectRepository={vi.fn()}
        onContinue={vi.fn()}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent("Checking for the new workspace…");
    expect(screen.getByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
  });

  it.each(["loading", "lookup failed", "connect", "grant", "choose"])(
    "keeps the %s controls separate from the Bitbucket animation",
    async (state) => {
      bitbucketOnboarding.providerConfigured = true;
      if (state === "loading") {
        bitbucketOnboarding.loaded = false;
        bitbucketOnboarding.isPending = true;
      }
      if (state === "lookup failed") {
        bitbucketOnboarding.loaded = false;
        bitbucketOnboarding.error = new Error("network error");
      }
      if (state === "grant" || state === "choose") {
        bitbucketOnboarding.identity = { login: "ada", providerUserId: "bitbucket-user-1" };
      }
      if (state === "choose") {
        bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
      }
      const user = userEvent.setup();
      renderSetup(new Set());

      await chooseBitbucket(user);

      const artStage = screen.getByTestId("first-run-art-stage");
      expect(artStage).toContainElement(screen.getByTestId("first-run-art-pane"));
      expect(artStage).not.toContainElement(screen.getByTestId("first-run-content"));
      expect(screen.getByTestId("first-run-back")).toBeEnabled();
    },
  );

  it("offers a retry instead of the access token setup when the Bitbucket lookup fails", async () => {
    bitbucketOnboarding.loaded = false;
    bitbucketOnboarding.error = new Error("network error");
    const user = userEvent.setup();
    const requestConnect = vi.fn();
    renderSetup(new Set(), { requestConnect });

    await chooseBitbucket(user);

    expect(screen.queryByTestId("first-run-connect-bitbucket")).not.toBeInTheDocument();
    expect(screen.getByText("SuperPlane could not check the Bitbucket setup. Try again.")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-bitbucket-retry"));
    expect(bitbucketOnboarding.refetch).toHaveBeenCalled();
    expect(requestConnect).not.toHaveBeenCalled();
  });

  it("saves the chosen Bitbucket repository and opens the ticket screen", async () => {
    const user = userEvent.setup();
    const selectBitbucketRepository = vi.fn().mockResolvedValue(true);
    renderSetup(new Set<IntegrationId>(["bitbucket"]), {
      bitbucketIntegrationId: "bitbucket-1",
      selectBitbucketRepository,
    });

    await chooseBitbucket(user);
    await user.click(screen.getByRole("option", { name: /acme-team\/api/ }));
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    await waitFor(() => expect(selectBitbucketRepository).toHaveBeenCalledWith("acme-team/api"));
    expect(resources.calls).toContainEqual({ integrationId: "bitbucket-1", type: "repository", enabled: true });
    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
  });

  it("returns from the Bitbucket repository screen to the host screen", async () => {
    const user = userEvent.setup();
    renderSetup(new Set<IntegrationId>(["bitbucket"]), { bitbucketIntegrationId: "bitbucket-1" });

    await chooseBitbucket(user);
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-host")).toBeInTheDocument();
  });
});
