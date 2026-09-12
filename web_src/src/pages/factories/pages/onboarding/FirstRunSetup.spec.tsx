import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
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

// The install-request recheck and the in-place bind need a query client and
// the network; the flow tests cover the screens only.
vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  useRecheckGitHubInstallRequest: vi.fn(),
}));

const bindMutate = vi.fn();

vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({
    mutateAsync: (variables: unknown) =>
      new Promise<void>((resolve, reject) => {
        bindMutate(variables, { onSuccess: resolve, onError: reject });
      }),
  }),
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

// The agent step reports organization spend, which this flow test does not use.
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
    provisionedDestination: null,
    githubOwner: undefined,
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

describe("FirstRunSetup", () => {
  beforeEach(() => {
    factory = { id: "factory-1", key: "PAY", name: "New workspace", onboarding: { vcsIntegrationId: "github-1" } };
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
    navigateSpy.mockClear();
    bindMutate.mockReset();
  });

  it.each([
    {
      scenario: "a new user, organization, and workspace",
      currentFactory: { onboarding: { initial: true, vcsIntegrationId: "github-1" } },
      otherFactories: [],
      otherOrganizations: [],
    },
    {
      scenario: "an existing user and organization with a new workspace",
      currentFactory: { onboarding: { vcsIntegrationId: "github-1" } },
      otherFactories: [{ id: "factory-2", key: "CORE", name: "Core" }],
      otherOrganizations: [],
    },
    {
      scenario: "an existing user with a new organization and workspace",
      currentFactory: { onboarding: { vcsIntegrationId: "github-1" } },
      otherFactories: [],
      otherOrganizations: [{ id: "org-2", name: "Existing organization" }],
    },
  ])(
    "finishes setup without the agent screen for $scenario",
    async ({ currentFactory, otherFactories, otherOrganizations }) => {
      factory = { id: "factory-1", key: "PAY", name: "New workspace", ...currentFactory };
      factories = [factory, ...otherFactories];
      accountOrganizations = [{ id: "org-1", name: "Acme" }, ...otherOrganizations];
      const user = userEvent.setup();
      const model = pageModel({ hostedAgentReady: true });

      renderSetup(model);

      await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

      expect(model.saveIssues).toHaveBeenCalledWith("vcs");
      await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
      expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
    },
  );

  // Regression: the click that sets the issues choice and the call that
  // provisions the workspace happen in the same handler. `finish` used to
  // read the issues choice back off setup state captured before the click,
  // which was still empty, so it saved an empty issues source over the one
  // `saveIssues` had just stored and provisioning failed on the first click.
  // A repository with no issues took the same "vcs" (GitHub Issues) answer as
  // any other repository, so this reproduced on every repository, not only
  // ones without issues.
  it("passes the just-selected issues choice to finish instead of stale setup state", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: true });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(model.finish).toHaveBeenCalledWith("vcs");
  });

  it("asks to connect GitHub when the workspace has not saved a connection", () => {
    factory = { id: "factory-1", onboarding: {} };

    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
        githubConnections: {
          name: "github",
          readyInstances: [
            {
              metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
              status: { state: "ready", metadata: { owner: "acme" } },
            },
          ],
          allInstances: [
            {
              metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
              status: { state: "ready", metadata: { owner: "acme" } },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // A connection bound before picker data was kept has nothing to pick from,
  // so the screen offers a new connect instead of a dead connected state.
  it("asks to connect GitHub again when the saved connection kept no picker data", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // A resumed pending organization or workspace opens without a step in the
  // URL. Setup then always starts on the welcome screen, even when earlier
  // answers exist.
  it("starts on the welcome screen when the URL carries no step", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
      }),
      "/org-1/workspaces/PAY/setup",
    );

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect")).not.toBeInTheDocument();
  });

  /** The workspace's bound connection, with the picker data a bind keeps. */
  function boundConnectionModel(selectVcsConnection: OnboardingPageModel["selectVcsConnection"]) {
    const boundInstance = {
      metadata: { id: "github-1", name: "github-acme", integrationName: "github" },
      status: {
        state: "ready",
        metadata: {
          owner: "acme",
          startedByUserID: "user-1",
          startedByGitHubLogin: "forestileao",
          state: "csrf",
          githubApp: { slug: "superplane" },
          pendingInstallations: [
            { id: "11", accountLogin: "acme" },
            { id: "22", accountLogin: "octo" },
          ],
        },
      },
    };
    return pageModel({
      openSection: "vcs",
      setup: { ...setupState(), vcsReady: true },
      selectedVcsConnectionId: "github-1",
      selectVcsConnection,
      githubConnections: {
        name: "github",
        readyInstances: [boundInstance],
        allInstances: [boundInstance],
      },
    });
  }

  it("reopens the account picker for the workspace's bound connection", () => {
    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup?step=vcs");

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") })).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-signed-in-as")).toHaveTextContent(
      FIRST_RUN_COPY.connect.signedInAs("forestileao"),
    );
    expect(screen.queryByTestId("first-run-github-connected")).not.toBeInTheDocument();
  });

  // Get started always opens the Connect GitHub page, even when picker data
  // exists. The user picks the GitHub account on every forward pass, because
  // many people stay signed in to two GitHub accounts.
  it("opens the Connect GitHub page from Get started even when picker data exists", async () => {
    const user = userEvent.setup();

    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup");

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-get-started"));

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
  });

  // Back walks the exact pages in reverse order: account picker, Connect
  // GitHub page, welcome.
  it("walks back from the picker to the Connect GitHub page and then to welcome", async () => {
    const user = userEvent.setup();

    renderSetup(boundConnectionModel(vi.fn().mockResolvedValue(true)), "/org-1/workspaces/PAY/setup?step=vcs");

    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
  });

  it("reopens the account picker when the user goes back from the repository screen", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(boundConnectionModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") }));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-github-account-picker")).toBeInTheDocument();
  });

  it("moves the bound connection to another account through the picker", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(boundConnectionModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("octo") }));

    expect(bindMutate).toHaveBeenCalledWith(
      { state: "csrf", installationId: "22" },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("github-1"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  function bindablePageModel(selectVcsConnection: OnboardingPageModel["selectVcsConnection"]) {
    const pendingInstance = {
      metadata: { id: "int-new", integrationName: "github" },
      status: {
        state: "pending",
        metadata: {
          startedByUserID: "user-1",
          state: "csrf",
          githubApp: { slug: "superplane" },
          pendingInstallations: [{ id: "11", accountLogin: "acme" }],
        },
      },
    };
    // The static test model shows the post-bind refetch already applied: the
    // bound connection reports ready.
    const readyInstance = {
      metadata: { id: "int-new", name: "github-acme", integrationName: "github" },
      status: { state: "ready", metadata: { owner: "acme" } },
    };
    return pageModel({
      openSection: "vcs",
      selectVcsConnection,
      githubConnections: {
        name: "github",
        readyInstances: [readyInstance],
        allInstances: [pendingInstance],
      },
    });
  }

  it("saves the bound GitHub connection for a workspace of an existing organization", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    factory = { id: "factory-1", onboarding: {} };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  it("saves the bound GitHub connection for the initial organization", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(true);
    factory = { id: "factory-1", onboarding: { initial: true } };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(await screen.findByTestId("first-run-choose")).toBeInTheDocument();
  });

  // Regression: the repository screen opened before the bound connection was
  // saved, so a fast repository pick stored the repository on the prior
  // connection. The screen must stay on connect when the save fails.
  it("keeps the connect screen when the bound connection does not save", async () => {
    const user = userEvent.setup();
    const selectVcsConnection = vi.fn().mockResolvedValue(false);
    factory = { id: "factory-1", onboarding: {} };
    bindMutate.mockImplementation((_vars: unknown, options: { onSuccess?: () => void }) => {
      options.onSuccess?.();
    });

    renderSetup(bindablePageModel(selectVcsConnection), "/org-1/workspaces/PAY/setup?step=vcs");

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") }));

    await waitFor(() => expect(selectVcsConnection).toHaveBeenCalledWith("int-new"));
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-choose")).not.toBeInTheDocument();
  });

  it("counts the ticket screen as the last step when the agent screen is skipped", () => {
    renderSetup(pageModel({ hostedAgentReady: true }));

    expect(screen.getByRole("navigation", { name: FIRST_RUN_COPY.chrome.stepLabel(4, 4) })).toBeInTheDocument();
  });

  it("shows setup progress on the ticket screen while it provisions the workspace", () => {
    renderSetup(pageModel({ hostedAgentReady: true, saving: true }));

    const finish = screen.getByTestId("first-run-analyze-tickets");
    expect(finish).toHaveTextContent(FIRST_RUN_COPY.finish.saving);
    expect(finish).toBeDisabled();
  });

  it("opens the agent screen when local setup has no hosted agent", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: false });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.continue }));

    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    expect(await screen.findByTestId("first-run-agent")).toBeInTheDocument();
    expect(model.finish).not.toHaveBeenCalled();
  });

  // Setup saved the ticket answer, then provisioning did not finish. The user
  // returns to the screen that carries the action, not to a screen with no
  // question left to answer.
  it("resumes on the ticket screen when hosted credentials cover the agent", () => {
    renderSetup(pageModel({ hostedAgentReady: true, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

  it("resumes on the agent screen when the agent still needs a connected provider", () => {
    renderSetup(pageModel({ hostedAgentReady: false, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
  });

  it("goes back through every screen to the welcome screen", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      setup: (() => {
        const setup = setupState();
        setup.selectRepo("acme/payments-service");
        return setup;
      })(),
    });

    renderSetup(model);

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-choose")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    // The welcome screen is the first screen, so it offers no Back.
    expect(screen.queryByTestId("first-run-back")).not.toBeInTheDocument();
  });
});
