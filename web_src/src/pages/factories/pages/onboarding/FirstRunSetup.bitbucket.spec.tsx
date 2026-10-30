import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { FirstRunSetup } from "./FirstRunSetup";
import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

const resources = vi.hoisted(() => ({
  calls: [] as Array<{ integrationId: string; type: string; enabled?: boolean }>,
}));

const bitbucketOnboarding = vi.hoisted(() => ({
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

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, isLoading: false, organizationReady: true }),
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
    agentModelPicker: undefined,
    openSection: "vcs",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
    selectCatalogRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketRepository: vi.fn().mockResolvedValue(true),
    selectBitbucketForgeRepository: vi.fn().mockResolvedValue(true),
    bitbucketIntegrationId: "",
    branchIntegrationId: "",
    defaultBranchRequired: false,
    chosenDefaultBranch: "",
    setChosenDefaultBranch: vi.fn(),
    requireDefaultBranch: vi.fn(),
    clearDefaultBranchRequirement: vi.fn(),
    saveChosenDefaultBranch: vi.fn().mockResolvedValue(true),
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
  const setup = useOnboardingSetupState("Payments", {
    connected,
    simulateDiscovery: false,
  });
  return <FirstRunSetup model={pageModel(setup, { jiraProjectId: "PAY", ...overrides })} />;
}

function renderSetup(connected: Set<IntegrationId>, overrides: Partial<OnboardingPageModel> = {}) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=welcome"]}>
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
    bitbucketOnboarding.identities = [];
    bitbucketOnboarding.selectIdentity = vi.fn().mockResolvedValue(undefined);
    bitbucketOnboarding.repositories = [];
    bitbucketOnboarding.installedWorkspaces = [];
    bitbucketOnboarding.isPending = false;
    bitbucketOnboarding.error = null;
    localStorage.clear();
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
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=welcome"]}>
        <StatefulSetup connected={new Set()} overrides={{}} />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Use acme-team" }));

    expect(await screen.findByRole("option", { name: /acme-team\/api/ })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-bitbucket-grant")).not.toBeInTheDocument();
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

  it("offers only Jira and Linear for a Bitbucket backlog", async () => {
    const user = userEvent.setup();
    const selectBitbucketRepository = vi.fn().mockResolvedValue(true);
    const requestConnect = vi.fn().mockResolvedValue(true);
    renderSetup(new Set<IntegrationId>(["bitbucket", "jira"]), {
      bitbucketIntegrationId: "bitbucket-1",
      selectBitbucketRepository,
      requestConnect,
    });

    await chooseBitbucket(user);
    await user.click(screen.getByRole("option", { name: /acme-team\/api/ }));
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    await waitFor(() => expect(selectBitbucketRepository).toHaveBeenCalledWith("acme-team/api"));
    expect(resources.calls).toContainEqual({ integrationId: "bitbucket-1", type: "repository", enabled: true });
    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /GitHub issues/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Linear issues/ })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Jira issues/ }));
    // Jira is connected, so the card opens on the project step.
    expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "done");
    expect(screen.getByTestId("first-run-jira-projects")).toBeInTheDocument();
    expect(requestConnect).not.toHaveBeenCalled();
  });

  it("returns from the Bitbucket repository screen to the host screen", async () => {
    const user = userEvent.setup();
    renderSetup(new Set<IntegrationId>(["bitbucket"]), { bitbucketIntegrationId: "bitbucket-1" });

    await chooseBitbucket(user);
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-host")).toBeInTheDocument();
  });

  describe("account switcher", () => {
    const ada = { login: "ada-bb", providerUserId: "11111111-1111-1111-1111-111111111111" };
    const grace = { login: "grace-bb", providerUserId: "22222222-2222-2222-2222-222222222222" };

    beforeEach(() => {
      bitbucketOnboarding.providerConfigured = true;
      bitbucketOnboarding.identity = ada;
      bitbucketOnboarding.repositories = [{ fullName: "acme-team/api" }];
    });

    it("lists linked accounts like GitHub and blocks the screen while a switch runs", async () => {
      bitbucketOnboarding.identities = [ada, grace];
      let finishSwitch: (() => void) | undefined;
      bitbucketOnboarding.selectIdentity = vi.fn(
        () =>
          new Promise<void>((resolve) => {
            finishSwitch = resolve;
          }),
      );
      const user = userEvent.setup();
      renderSetup(new Set());

      await chooseBitbucket(user);

      expect(screen.getByTestId("first-run-bitbucket-signed-in-as")).toHaveTextContent(
        "You are signed in to Bitbucket as ada-bb.",
      );
      const switchButton = screen.getByTestId("first-run-switch-bitbucket-account");
      expect(switchButton).toHaveTextContent("Switch account");

      await user.click(switchButton);
      const menuEntries = Array.from(
        screen.getByRole("menu").querySelectorAll('[role="menuitemradio"], [role="separator"], [role="menuitem"]'),
      ).map((entry) => `${entry.getAttribute("role")}:${entry.textContent}`);
      expect(menuEntries).toEqual([
        "menuitemradio:ada-bb",
        "menuitemradio:grace-bb",
        "separator:",
        "menuitem:Connect another Bitbucket account",
      ]);
      expect(screen.getByRole("menuitemradio", { name: "ada-bb" })).toHaveAttribute("aria-checked", "true");

      await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));

      expect(bitbucketOnboarding.selectIdentity).toHaveBeenCalledWith(grace.providerUserId);
      expect(screen.getByTestId("first-run-switch-bitbucket-account")).toBeDisabled();
      await act(async () => finishSwitch?.());
      expect(screen.getByTestId("first-run-switch-bitbucket-account")).toBeEnabled();
    });

    it("explains a failed account switch and clears the message on the next attempt", async () => {
      bitbucketOnboarding.identities = [ada, grace];
      let rejectSwitch: ((error: Error) => void) | undefined;
      let finishSwitch: (() => void) | undefined;
      bitbucketOnboarding.selectIdentity = vi
        .fn()
        .mockImplementationOnce(
          () =>
            new Promise<void>((_, reject) => {
              rejectSwitch = reject;
            }),
        )
        .mockImplementationOnce(
          () =>
            new Promise<void>((resolve) => {
              finishSwitch = resolve;
            }),
        );
      const user = userEvent.setup();
      renderSetup(new Set());
      await chooseBitbucket(user);

      await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
      await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));
      await act(async () => rejectSwitch?.(new Error("denied")));

      expect(await screen.findByTestId("first-run-bitbucket-switch-error")).toHaveTextContent(
        "SuperPlane could not switch the Bitbucket account. Try again.",
      );
      expect(screen.getByTestId("first-run-bitbucket-signed-in-as")).toHaveTextContent(
        "You are signed in to Bitbucket as ada-bb.",
      );

      await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
      await user.click(screen.getByRole("menuitemradio", { name: "grace-bb" }));

      expect(screen.queryByTestId("first-run-bitbucket-switch-error")).not.toBeInTheDocument();
      await act(async () => finishSwitch?.());
      expect(screen.queryByTestId("first-run-bitbucket-switch-error")).not.toBeInTheDocument();
    });

    it("offers to connect another account when only one account is linked", async () => {
      bitbucketOnboarding.identities = [ada];
      const previousAssign = window.location.assign.bind(window.location);
      const assign = vi.fn();
      window.location.assign = assign;
      const user = userEvent.setup();

      try {
        renderSetup(new Set());
        await chooseBitbucket(user);
        await user.click(screen.getByTestId("first-run-switch-bitbucket-account"));
        await user.click(screen.getByRole("menuitem", { name: "Connect another Bitbucket account" }));

        expect(assign).toHaveBeenCalledWith(expect.stringContaining("/auth/bitbucket?intent=connect"));
        expect(bitbucketOnboarding.selectIdentity).not.toHaveBeenCalled();
      } finally {
        window.location.assign = previousAssign;
      }
    });
  });
});

vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({
    data: { billingEnabled: false },
    refetch: async () => ({ data: { billingEnabled: false } }),
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
