import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

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
  useBindGitHubInstallation: () => ({ mutate: bindMutate, isPending: false, variables: undefined }),
}));

const navigateSpy = vi.fn();

let accountOrganizations: Array<{ id: string; name: string; slug?: string }>;

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: accountOrganizations }),
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
    openSection: "issues",
    setOpenSection: vi.fn(),
    requestConnect: vi.fn(),
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

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

describe("FirstRunSetup", () => {
  beforeEach(() => {
    factory = { id: "factory-1", onboarding: { vcsIntegrationId: "github-1" } };
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
    navigateSpy.mockClear();
    bindMutate.mockReset();
  });

  it("finishes setup from the ticket screen when hosted credentials cover the agent", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: true });

    renderSetup(model);

    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    expect(model.saveIssues).toHaveBeenCalledWith("vcs");
    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

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

  it("shows GitHub as connected when the workspace already saved this connection", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-github-connected")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
  });

  it("starts a new connect from the connected state to change the GitHub account", async () => {
    const user = userEvent.setup();
    const createVcsConnection = vi.fn();

    renderSetup(
      pageModel({
        openSection: "vcs",
        setup: { ...setupState(), vcsReady: true },
        selectedVcsConnectionId: "github-1",
        createVcsConnection,
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    await user.click(screen.getByTestId("first-run-github-use-different"));
    expect(createVcsConnection).toHaveBeenCalled();
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

  it("shows the GitHub account picker on the connect screen", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        githubConnections: {
          name: "github",
          readyInstances: [],
          allInstances: [
            {
              metadata: { id: "int-1", integrationName: "github" },
              status: {
                state: "pending",
                metadata: {
                  startedByUserID: "user-1",
                  state: "csrf",
                  githubApp: { slug: "superplane" },
                  pendingInstallations: [
                    { id: "11", accountLogin: "acme" },
                    { id: "22", accountLogin: "octo" },
                  ],
                },
              },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.getByTestId("first-run-github-account-picker")).toHaveTextContent(
      FIRST_RUN_COPY.connect.selectAccount,
    );
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.connect.useAccount("acme") })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-connect-github")).not.toBeInTheDocument();
  });

  it("does not show another member's GitHub account picker", () => {
    renderSetup(
      pageModel({
        openSection: "vcs",
        githubConnections: {
          name: "github",
          readyInstances: [],
          allInstances: [
            {
              metadata: { id: "int-1", integrationName: "github" },
              status: {
                state: "pending",
                metadata: {
                  startedByUserID: "some-other-user",
                  state: "csrf",
                  githubApp: { slug: "superplane" },
                  pendingInstallations: [
                    { id: "11", accountLogin: "acme" },
                    { id: "22", accountLogin: "octo" },
                  ],
                },
              },
            },
          ],
        },
      }),
      "/org-1/workspaces/PAY/setup?step=vcs",
    );

    expect(screen.queryByTestId("first-run-github-account-picker")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
  });

  it("opens Connect when GitHub returned an install request without a step", () => {
    renderSetup(pageModel({ openSection: "vcs" }), "/org-1/workspaces/PAY/setup?githubSetup=request");

    expect(screen.getByTestId("first-run-connect")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-github-install-requested")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-welcome")).not.toBeInTheDocument();
  });

  it("shows a waiting chip when GitHub returned an install request", () => {
    renderSetup(pageModel({ openSection: "vcs" }), "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request");

    expect(screen.getByTestId("first-run-github-install-requested")).toHaveTextContent(
      FIRST_RUN_COPY.connect.installRequested,
    );
    expect(screen.getByTestId("first-run-connect-github")).toBeInTheDocument();
  });

  it("names the GitHub organization from the return query", () => {
    renderSetup(
      pageModel({ openSection: "vcs" }),
      "/org-1/workspaces/PAY/setup?step=vcs&githubSetup=request&githubOrg=acme",
    );

    expect(screen.getByTestId("first-run-github-install-org")).toHaveTextContent("acme");
    expect(screen.queryByText(FIRST_RUN_COPY.connect.installRequestedBody("acme"))).not.toBeInTheDocument();
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

  it("opens the agent screen when the agent needs a connected provider", async () => {
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
    renderSetup(pageModel({ hostedAgentReady: true, openSection: "agent" }));

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

  it("resumes on the agent screen when the agent still needs a connected provider", () => {
    renderSetup(pageModel({ hostedAgentReady: false, openSection: "agent" }));

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
  });

  it("shows Log out and the organization switch when another workspace exists", () => {
    factories = [factory, { id: "factory-2" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
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
  });

  it("opens the current organization from the switch menu so the user can leave setup", async () => {
    factories = [factory, { id: "factory-2" }];
    const user = userEvent.setup();

    renderSetup(pageModel());

    await user.click(screen.getByTestId("first-run-organization-switch"));
    await user.click(screen.getByTestId("first-run-organization-option-org-1"));

    expect(navigateSpy).toHaveBeenCalledWith("/org-1");
  });

  it("goes back from the ticket screen to the repository screen", async () => {
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
  });

  it("keeps Log out and hides the organization switch with a single org and single workspace", () => {
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-organization-switch")).not.toBeInTheDocument();
    expect(screen.queryByTestId("first-run-cancel")).not.toBeInTheDocument();
  });
});
