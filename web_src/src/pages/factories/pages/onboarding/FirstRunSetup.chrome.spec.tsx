import { render, renderHook, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";
import { unmockedPackage } from "@/test/unmockedModule";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import type { OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";
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

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => true, enabledExperimentalFeatures: [], isLoading: false }),
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

vi.mock("react-router", () => {
  const actual = unmockedPackage<typeof ReactRouterDom>("react-router/dist/development/index.js");
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
    hostedModelsAvailable: false,
    hostedModelsAvailableLoading: false,
    bringYourOwnKey: false,
    bringYourOwnKeyLoading: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    openSection: "issues",
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

function withRepository(): OnboardingSetupApi {
  return { ...setupState(), selectedRepo: "acme/payments-service" };
}

function StatefulSetup({ model }: { model: OnboardingPageModel }) {
  const [choice, setChoice] = useState<OnboardingAgentCredentialChoice | null>(null);
  return <FirstRunSetup model={{ ...model, agentCredentialChoice: choice, setAgentCredentialChoice: setChoice }} />;
}

function renderStatefulSetup(overrides: Partial<OnboardingPageModel>) {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=agent"]}>
      <StatefulSetup model={pageModel({ openSection: "agent", ...overrides })} />
    </MemoryRouter>,
  );
}

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  render(
    <MemoryRouter initialEntries={[path]}>
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

  // A new organization and a new workspace in an existing organization use
  // the same redesigned screens, so the sphere pane shows on both.
  it("shows the factory sphere on welcome", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup"]}>
        <FirstRunSetup model={pageModel({ openSection: "vcs" })} />
      </MemoryRouter>,
    );

    expect(screen.getByText(FIRST_RUN_COPY.sphere.captionSetup)).toBeInTheDocument();
  });

  it("shows the model source before the backlog for a bring-your-own-key organization", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      hostedModelsAvailable: true,
      bringYourOwnKey: true,
      openSection: "repo",
      setup: withRepository(),
    });

    renderSetup(model, "/org-1/workspaces/PAY/setup?step=repo");
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.choose.continueReady }));

    expect(await screen.findByTestId("first-run-model-source")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-tickets")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-step")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  });

  it("lists SuperPlane-hosted models before Your key in the model source choice", () => {
    renderSetup(pageModel({ hostedModelsAvailable: true, bringYourOwnKey: true }));

    const source = screen.getByTestId("first-run-model-source");
    const buttons = within(source).getAllByRole("button");
    expect(buttons[0]).toHaveAccessibleName(new RegExp(FIRST_RUN_COPY.agent.hostedModels));
    expect(buttons[1]).toHaveAccessibleName(new RegExp(FIRST_RUN_COPY.agent.ownKey));
  });

  it("opens the backlog when the organization chooses SuperPlane-hosted models", async () => {
    const user = userEvent.setup();
    renderStatefulSetup({ hostedAgentReady: true, hostedModelsAvailable: true, bringYourOwnKey: true });

    await user.click(screen.getByRole("button", { name: new RegExp(FIRST_RUN_COPY.agent.hostedModels) }));
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.continue }));

    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze })).toBeInTheDocument();
  });

  it("shows provider keys when the organization chooses its own key", async () => {
    const user = userEvent.setup();
    renderStatefulSetup({ hostedAgentReady: true, hostedModelsAvailable: true, bringYourOwnKey: true });

    await user.click(screen.getByRole("button", { name: new RegExp(FIRST_RUN_COPY.agent.ownKey) }));

    expect(screen.getByText(FIRST_RUN_COPY.agent.ownKeyBody)).toBeInTheDocument();
    expect(screen.getByTestId("agent-step")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  });

  it("sends a bring-your-own-key organization back to the model source when none is chosen", () => {
    renderSetup(pageModel({ hostedModelsAvailable: true, bringYourOwnKey: true }));

    expect(screen.getByTestId("first-run-model-source")).toBeInTheDocument();
  });

  it("scans the backlog after a saved model source", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      bringYourOwnKey: true,
      agentCredentialChoice: "hosted",
    });

    renderSetup(model);
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));

    await waitFor(() => expect(model.finish).toHaveBeenCalledWith("vcs"));
  });

  it("hides the model source from an organization without the bring-your-own-key flag", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      openSection: "repo",
      setup: withRepository(),
    });

    renderSetup(model, "/org-1/workspaces/PAY/setup?step=repo");
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.choose.continueReady }));

    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-model-source")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze }));
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
    expect(screen.queryByTestId("first-run-model-source")).not.toBeInTheDocument();
  });

  it("hides the model source when only a provider key can run the agent", () => {
    renderSetup(pageModel({ bringYourOwnKey: true, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-model-source")).not.toBeInTheDocument();
  });

  it("waits to choose the agent screen while the bring-your-own-key flag is loading", () => {
    renderSetup(pageModel({ hostedAgentReady: true, hostedModelsAvailable: true, bringYourOwnKeyLoading: true }));

    const continueButton = screen.getByTestId("first-run-analyze-tickets");
    expect(continueButton).toBeDisabled();
    expect(continueButton).toHaveTextContent(FIRST_RUN_COPY.agent.loading);
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });
});
