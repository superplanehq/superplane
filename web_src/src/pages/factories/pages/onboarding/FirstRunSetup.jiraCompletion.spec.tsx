import { render, renderHook, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";

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

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
    isPending: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/hooks/useRecheckGitHubInstallRequest", () => ({
  useRecheckGitHubInstallRequest: vi.fn(),
}));

vi.mock("@/hooks/useBindGitHubInstallation", () => ({
  useBindGitHubInstallation: () => ({ mutateAsync: vi.fn() }),
}));

let accountOrganizations: Array<{ id: string; name: string; slug?: string }>;

vi.mock("@/hooks/useAccountOrganizations", () => ({
  useAccountOrganizations: () => ({ data: accountOrganizations, refetch: vi.fn() }),
}));

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

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  render(
    <MemoryRouter initialEntries={[path]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

function jiraSetupState() {
  return renderHook(() =>
    useOnboardingSetupState("Payments Service", {
      simulateDiscovery: false,
      connected: new Set(["jira"]),
      initial: { issuesChoice: "jira" },
    }),
  );
}

describe("FirstRunSetup Jira completion", () => {
  beforeEach(() => {
    factory = { id: "factory-1", key: "PAY", name: "New workspace", onboarding: { vcsIntegrationId: "github-1" } };
    factories = [factory];
    accountOrganizations = [{ id: "org-1", name: "Acme" }];
  });

  it("keeps the Jira completion column off the ticket screen when Connect agent is next", () => {
    const { result } = jiraSetupState();

    renderSetup(
      pageModel({
        hostedAgentReady: false,
        setup: result.current,
        jiraIntegrationId: "jira-1",
        jiraProjectId: "PAY",
        jiraProjects: [{ id: "PAY", name: "Payments" }],
      }),
    );

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.queryByTestId("jira-completion-column")).not.toBeInTheDocument();
  });

  it("shows the Jira completion column on the agent screen", () => {
    const { result } = jiraSetupState();

    renderSetup(
      pageModel({
        hostedAgentReady: false,
        openSection: "agent",
        setup: result.current,
        jiraIntegrationId: "jira-1",
        jiraProjectId: "PAY",
        jiraProjects: [{ id: "PAY", name: "Payments" }],
      }),
      "/org-1/workspaces/PAY/setup?step=agent",
    );

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
    expect(screen.getByTestId("jira-completion-column")).toBeInTheDocument();
  });

  it("shows the Jira completion column on the ticket screen when the agent step is skipped", () => {
    const { result } = jiraSetupState();

    renderSetup(
      pageModel({
        hostedAgentReady: true,
        setup: result.current,
        jiraIntegrationId: "jira-1",
        jiraProjectId: "PAY",
        jiraProjects: [{ id: "PAY", name: "Payments" }],
      }),
    );

    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(screen.getByTestId("jira-completion-column")).toBeInTheDocument();
  });
});
