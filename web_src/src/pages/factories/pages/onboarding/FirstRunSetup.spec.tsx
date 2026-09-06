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

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Acme" } } }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

// The install-request recheck and the in-place bind need a query client and
// the network; the flow tests cover the screens only.
vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  useRecheckGitHubInstallRequest: vi.fn(),
}));

vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({ mutate: vi.fn(), isPending: false, variables: undefined }),
}));

const deleteFactoryMutateAsync = vi.fn().mockResolvedValue(undefined);
const navigateSpy = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useDeleteFactory: () => ({ mutateAsync: deleteFactoryMutateAsync, isPending: false }),
}));

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
    selectVcsConnection: vi.fn(),
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

async function openAccountMenu(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("factories-sidebar-user-menu-trigger"));
}

describe("FirstRunSetup", () => {
  beforeEach(() => {
    factory = { id: "factory-1", onboarding: { vcsIntegrationId: "github-1" } };
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
    deleteFactoryMutateAsync.mockClear();
    navigateSpy.mockClear();
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

  it("offers Quit onboarding from the account menu when another workspace exists", async () => {
    factories = [factory, { id: "factory-2" }];
    const user = userEvent.setup();

    renderSetup(pageModel());
    await openAccountMenu(user);

    expect(screen.getByRole("menuitem", { name: "Quit onboarding" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Sign out" })).toBeInTheDocument();
  });

  it("deletes the placeholder workspace and returns to the workspace index when the user quits onboarding", async () => {
    factories = [factory, { id: "factory-2" }];
    const user = userEvent.setup();

    renderSetup(pageModel());
    await openAccountMenu(user);
    await user.click(screen.getByRole("menuitem", { name: "Quit onboarding" }));

    expect(deleteFactoryMutateAsync).toHaveBeenCalledWith("factory-1");
    await waitFor(() => expect(navigateSpy).toHaveBeenCalledWith("/org-1/workspaces"));
  });

  it("offers Quit onboarding when another organization exists, even with no other workspace here", async () => {
    factories = [factory];
    accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co" },
    ];
    const user = userEvent.setup();

    renderSetup(pageModel());
    await openAccountMenu(user);

    expect(screen.getByRole("menuitem", { name: "Quit onboarding" })).toBeInTheDocument();
  });

  it("navigates to another organization when the user quits onboarding, not back into onboarding, when this org has no other workspace", async () => {
    factories = [factory];
    accountOrganizations = [
      { id: "org-1", name: "Acme" },
      { id: "org-2", name: "Other Co", slug: "other-co" },
    ];
    const user = userEvent.setup();

    renderSetup(pageModel());
    await openAccountMenu(user);
    await user.click(screen.getByRole("menuitem", { name: "Quit onboarding" }));

    expect(deleteFactoryMutateAsync).toHaveBeenCalledWith("factory-1");
    await waitFor(() => expect(navigateSpy).toHaveBeenCalledWith("/other-co"));
    expect(navigateSpy).not.toHaveBeenCalledWith("/org-1/workspaces");
  });

  it("hides Quit onboarding and keeps Sign out with a single org and single (placeholder) workspace", async () => {
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
    const user = userEvent.setup();

    renderSetup(pageModel());
    await openAccountMenu(user);

    expect(screen.queryByRole("menuitem", { name: "Quit onboarding" })).not.toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Sign out" })).toBeInTheDocument();
  });
});
