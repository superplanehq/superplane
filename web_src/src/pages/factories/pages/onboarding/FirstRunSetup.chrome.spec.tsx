import { act, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory, OrganizationsBillingPlanOption } from "@/api-client";
import { unmockedPackage } from "@/test/unmockedModule";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import type { FirstRunChrome } from "./first-run/firstRunTypes";
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
    routeSegment: "pay",
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
  useExperimentalFeature: () => ({
    has: (feature: string) =>
      feature === "organization_byok" ? billingState.byok : feature !== "new_billing" || billingState.newBilling,
    enabledExperimentalFeatures: [],
    organizationReady: true,
    isLoading: false,
  }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

vi.mock("./useBitbucketOnboarding", () => ({
  useBitbucketOnboarding: () => ({
    data: { providerConfigured: false, identity: undefined, repositories: [], installUrl: "" },
    isPending: false,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
  }),
}));

let githubPending = false;
let githubLinked = true;
vi.mock("./useGitHubOnboarding", () => ({
  useGitHubOnboarding: () => ({
    data: {
      providerConfigured: true,
      accountConnectionRequired: true,
      identity: githubLinked ? { userId: "42", login: "octocat" } : undefined,
      repositories: [{ repositoryId: "201", installationId: "101", fullName: "acme/api", defaultBranch: "main" }],
      pendingRequests: [],
      synchronizing: false,
    },
    isPending: githubPending,
    error: null,
    startInstallation: { mutateAsync: vi.fn() },
    configureInstallation: { mutateAsync: vi.fn() },
  }),
  useGitHubInstallationChecks: vi.fn(),
}));

const billingState = {
  enabled: false,
  newBilling: false,
  byok: false,
  availablePlans: undefined as OrganizationsBillingPlanOption[] | undefined,
};
const refetchBilling = vi.fn(async () => ({
  data: {
    billingEnabled: billingState.enabled,
    availablePlans: billingState.availablePlans,
    productFeatures: { byo_key: true },
    planCategory: "paid",
    creditPurchaseAllowed: true,
  },
}));
vi.mock("@/hooks/useOrganizationBilling", () => ({
  organizationBillingQueryKey: (id: string) => ["organizations", id, "billing"],
  useOrganizationBilling: () => ({
    data: {
      billingEnabled: billingState.enabled,
      availablePlans: billingState.availablePlans,
      productFeatures: { byo_key: true },
      planCategory: "paid",
      creditPurchaseAllowed: true,
    },
    isPending: false,
    isSuccess: true,
    refetch: refetchBilling,
  }),
}));
vi.mock("./first-run/FirstRunBillingScreen", () => ({
  FirstRunBillingScreen: ({ onContinue, chrome }: { onContinue: () => void; chrome?: FirstRunChrome }) => (
    <section data-testid="first-run-billing">
      <button data-testid="first-run-back" onClick={chrome?.onBack}>
        Back
      </button>
      <button onClick={onContinue}>Finish with selected plan</button>
    </section>
  ),
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
  AgentStep: ({ showCustomProvider }: { showCustomProvider?: boolean }) => (
    <div data-testid="agent-step" data-custom-provider={showCustomProvider ? "yes" : "no"} />
  ),
}));

function setupState(): OnboardingSetupApi {
  const { result } = renderHook(() =>
    useOnboardingSetupState("Payments Service", {
      simulateDiscovery: false,
      connected: new Set(["github"]),
      initial: { vcsHost: "github", issuesChoice: "vcs", selectedRepo: "acme/api" },
    }),
  );
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
    customProvider: false,
    agentCredentialChoice: null,
    setAgentCredentialChoice: vi.fn(),
    agentLoading: false,
    agentModelPicker: undefined,
    openSection: "issues",
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

function StatefulSetup({ model }: { model: OnboardingPageModel }) {
  const [choice, setChoice] = useState<OnboardingAgentCredentialChoice | null>(null);
  return <FirstRunSetup model={{ ...model, agentCredentialChoice: choice, setAgentCredentialChoice: setChoice }} />;
}

function NavigableSetup({ model }: { model: OnboardingPageModel }) {
  const setup = useOnboardingSetupState("Payments Service", {
    simulateDiscovery: false,
    connected: new Set(["github"]),
  });
  return <StatefulSetup model={{ ...model, setup }} />;
}

function renderStatefulSetup(overrides: Partial<OnboardingPageModel>) {
  billingState.byok = overrides.bringYourOwnKey ?? false;
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=agent"]}>
      <StatefulSetup model={pageModel({ openSection: "agent", ...overrides })} />
    </MemoryRouter>,
  );
}

function renderSetup(model: OnboardingPageModel, path = "/org-1/workspaces/PAY/setup?step=issues") {
  billingState.byok = model.bringYourOwnKey;
  return render(
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
    billingState.byok = false;
    billingState.enabled = false;
    billingState.newBilling = false;
    billingState.availablePlans = [{ key: "free", name: "Free" }];
    refetchBilling.mockClear();
    localStorage.clear();
    githubPending = false;
    githubLinked = true;
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
  it("keeps welcome and provider choice after the initial GitHub lookup", async () => {
    githubPending = true;
    githubLinked = false;
    const model = pageModel({ openSection: "vcs", setup: { ...setupState(), selectedRepo: null } });
    const path = "/org-1/workspaces/PAY/setup";
    const { rerender } = renderSetup(model, path);
    githubPending = false;
    rerender(
      <MemoryRouter initialEntries={[path]}>
        <FirstRunSetup model={model} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("first-run-welcome")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-art-stage")).toContainElement(screen.getByTestId("first-run-signed-in"));
    await userEvent.setup().click(screen.getByTestId("first-run-get-started"));
    expect(screen.getByTestId("first-run-host-bitbucket")).toBeInTheDocument();
  });

  it.each([false, true])("keeps repository and intake before the final model choice (billing: %s)", async (billing) => {
    billingState.enabled = billing;
    billingState.newBilling = billing;
    billingState.byok = true;
    const user = userEvent.setup();
    const model = pageModel({ hostedModelsAvailable: true, bringYourOwnKey: true, openSection: "vcs" });
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=welcome"]}>
        <NavigableSetup model={model} />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("first-run-get-started"));
    expect(screen.getByTestId("first-run-host")).toBeInTheDocument();
    await user.click(within(screen.getByTestId("first-run-host-github")).getByRole("button"));
    await user.click(await screen.findByRole("button", { name: "Use acme" }));
    await user.click(screen.getByRole("option", { name: /acme\/api/ }));
    await user.click(screen.getByTestId("first-run-continue-to-tickets"));

    expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
    expect(model.saveIssues).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await waitFor(() => expect(model.saveIssues).toHaveBeenCalledWith("vcs"));
    if (billing) {
      expect(await screen.findByTestId("first-run-billing")).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Finish with selected plan" }));
    }
    expect(await screen.findByTestId("first-run-model-source")).toBeInTheDocument();
    expect(model.finish).not.toHaveBeenCalled();
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();

    await user.click(screen.getByTestId("first-run-back"));
    if (billing) {
      expect(screen.getByTestId("first-run-billing")).toBeInTheDocument();
      await user.click(screen.getByTestId("first-run-back"));
    }
    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-choose")).toBeInTheDocument();
  });

  it("keeps SuperPlane-hosted models as the choice and own key as a link", () => {
    renderSetup(
      pageModel({ hostedModelsAvailable: true, bringYourOwnKey: true }),
      "/org-1/workspaces/PAY/setup?step=agent",
    );

    const source = screen.getByTestId("first-run-model-source");
    const buttons = within(source).getAllByRole("button");
    expect(buttons[0]).toHaveAccessibleName(new RegExp(FIRST_RUN_COPY.agent.hostedModels));
    expect(buttons[0]).toHaveAttribute("aria-pressed", "true");
    expect(buttons[1]).toHaveAccessibleName(FIRST_RUN_COPY.agent.ownKey);
    expect(buttons[1]).toHaveAttribute("aria-pressed", "false");
  });

  it("finishes setup with hosted models after model selection", async () => {
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      bringYourOwnKey: true,
      openSection: "agent",
    });
    billingState.byok = true;
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=agent"]}>
        <StatefulSetup model={model} />
      </MemoryRouter>,
    );
    await user.click(screen.getByTestId("first-run-finish-setup"));
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
  });

  it("shows provider keys when the organization chooses its own key", async () => {
    const user = userEvent.setup();
    renderStatefulSetup({ hostedAgentReady: true, hostedModelsAvailable: true, bringYourOwnKey: true });

    await user.click(screen.getByRole("button", { name: new RegExp(FIRST_RUN_COPY.agent.ownKey) }));

    expect(screen.getByText(FIRST_RUN_COPY.agent.ownKeyBody)).toBeInTheDocument();
    expect(screen.getByTestId("agent-step")).toHaveAttribute("data-custom-provider", "no");
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  });

  it("blocks the model screen while provisioning is in progress", async () => {
    const user = userEvent.setup();
    let complete!: () => void;
    const finish = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    const model = pageModel({
      hostedModelsAvailable: true,
      bringYourOwnKey: true,
      agentCredentialChoice: "own-key",
      setup: { ...setupState(), connected: new Set(["github", "claude"]) },
      finish,
    });
    renderSetup(model, "/org-1/workspaces/PAY/setup?step=agent");
    await user.click(screen.getByTestId("first-run-finish-setup"));
    expect(finish).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
    await act(async () => complete());
  });

  it("offers a custom provider on the agent card when that flag is on", async () => {
    const user = userEvent.setup();
    renderStatefulSetup({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      bringYourOwnKey: true,
      customProvider: true,
    });

    await user.click(screen.getByRole("button", { name: new RegExp(FIRST_RUN_COPY.agent.ownKey) }));

    expect(screen.getByText(FIRST_RUN_COPY.agent.ownKeyBodyWithCustom)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.agent.ownKey })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("agent-step")).toHaveAttribute("data-custom-provider", "yes");
  });

  it("opens the factory without model setup when BYOK is unavailable", async () => {
    const user = userEvent.setup();
    const model = pageModel({ hostedAgentReady: true, hostedModelsAvailable: true, setup: setupState() });
    renderSetup(model);
    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
    expect(screen.queryByTestId("first-run-model-source")).not.toBeInTheDocument();
  });

  it("hides the model source when only a provider key can run the agent", () => {
    renderSetup(pageModel({ bringYourOwnKey: true, openSection: "agent" }), "/org-1/workspaces/PAY/setup?step=agent");

    expect(screen.getByTestId("first-run-agent")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-model-source")).not.toBeInTheDocument();
  });

  it.each(["jira", "linear"] as const)("requires a project before honoring a %s model setup return", (source) => {
    const model = pageModel({
      bringYourOwnKey: true,
      setup: { ...setupState(), issuesChoice: source, connected: new Set(["github", source]) },
    });
    renderSetup(model, "/org-1/workspaces/PAY/setup?step=agent");
    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    expect(model.finish).not.toHaveBeenCalled();
  });

  it("keeps intake available while model access loads", () => {
    renderSetup(pageModel({ hostedAgentReady: true, hostedModelsAvailable: true, bringYourOwnKeyLoading: true }));
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeEnabled();
    expect(screen.queryByTestId("first-run-agent")).not.toBeInTheDocument();
  });

  it.each([false, true])(
    "opens plan selection before provisioning (skip initial import: %s)",
    async (skipInitialImport) => {
      billingState.enabled = true;
      billingState.newBilling = true;
      const user = userEvent.setup();
      const model = pageModel({
        hostedAgentReady: true,
        hostedModelsAvailable: true,
        setup: { ...setupState(), issuesChoice: "vcs", skipInitialImport },
      });
      renderSetup(model);
      await user.click(screen.getByTestId("first-run-analyze-tickets"));
      expect(await screen.findByRole("button", { name: "Finish with selected plan" })).toBeInTheDocument();
      expect(model.finish).not.toHaveBeenCalled();
      await user.click(screen.getByRole("button", { name: "Finish with selected plan" }));
      await waitFor(() => expect(model.finish).toHaveBeenCalled());
    },
  );

  it.each(["issues", "billing"])("finishes with the legacy trial when new billing is off at step %s", async (step) => {
    billingState.enabled = true;
    billingState.newBilling = false;
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      setup: { ...setupState(), issuesChoice: "vcs" },
    });
    renderSetup(model, `/org-1/workspaces/PAY/setup?step=${step}`);
    expect(screen.queryByRole("button", { name: "Finish with selected plan" })).not.toBeInTheDocument();
    await user.click(
      step === "issues"
        ? screen.getByTestId("first-run-analyze-tickets")
        : screen.getByRole("button", { name: "Continue" }),
    );
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
  });

  it.each([
    { step: "issues", availablePlans: [] },
    { step: "billing", availablePlans: [] },
    { step: "issues", availablePlans: undefined },
    { step: "billing", availablePlans: undefined },
  ])("skips plan selection when no plans are available (%j)", async ({ step, availablePlans }) => {
    billingState.enabled = true;
    billingState.newBilling = true;
    billingState.availablePlans = availablePlans;
    const user = userEvent.setup();
    const model = pageModel({
      hostedAgentReady: true,
      hostedModelsAvailable: true,
      setup: { ...setupState(), issuesChoice: "vcs" },
    });
    renderSetup(model, `/org-1/workspaces/PAY/setup?step=${step}`);
    expect(screen.queryByRole("button", { name: "Finish with selected plan" })).not.toBeInTheDocument();
    await user.click(
      step === "issues"
        ? screen.getByTestId("first-run-analyze-tickets")
        : screen.getByRole("button", { name: "Continue" }),
    );
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
    expect(screen.queryByRole("button", { name: "Finish with selected plan" })).not.toBeInTheDocument();
  });
});
