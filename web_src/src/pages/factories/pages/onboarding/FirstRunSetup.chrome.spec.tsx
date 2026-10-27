import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type * as ReactRouterDom from "react-router";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";
import { unmockedPackage } from "@/test/unmockedModule";

import { billingState, defaultBillingState, refetchBilling } from "./FirstRunSetup.chrome.spec.billing";
import { hostedVcsModel, pageModel, setupState } from "./FirstRunSetup.chrome.spec.fixtures";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import type { OnboardingAgentCredentialChoice } from "./onboardingAgentReadiness";
import { useOnboardingSetupState } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

let factory: FactoriesFactory;
let factories: FactoriesFactory[];
let canManageBilling = true;
let permissionsLoading = false;

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => resource !== "org" || action !== "update" || canManageBilling,
    isLoading: permissionsLoading,
  }),
}));

const layoutIds = { organizationId: "org-1", factoryId: "factory-1", factoryKey: "PAY", routeSegment: "pay" };
vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({ ...layoutIds, factory, factories }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));

vi.mock("@/hooks/useMe", () => ({ useMe: () => ({ data: { id: "user-1" } }) }));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (feature: string) => feature === "organization_byok" && billingState.byok,
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
      repositories: ["acme/api", "acme/billing", "other/tools"].map((fullName, index) => ({
        repositoryId: `${201 + index}`,
        installationId: "101",
        fullName,
        defaultBranch: "main",
      })),
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

const workspaceUsage = { defaultHostedProvider: "openrouter", defaultHostedModel: "openai/gpt-5" };
vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({
    data: workspaceUsage,
    isSuccess: true,
    isPending: false,
    isError: false,
    refetch: async () => ({ data: workspaceUsage }),
  }),
}));
const openCardSetup = vi.fn(async () => {});
const activateOnboardingPlan = vi.fn(async () => {
  billingState.planActive = true;
  return false;
});
const resumeOnboardingPlan = vi.fn(async () => false);
vi.mock("./onboardingCardPlan", () => ({
  activateOnboardingPlan,
  createOnboardingCardSession: openCardSetup,
  resumeOnboardingPlan,
}));

const navigateSpy = vi.fn();

let accountOrganizations: Array<{ id: string; name: string; slug?: string }>;
const twoOrganizations = [
  { id: "org-1", name: "Acme" },
  { id: "org-2", name: "Other Co" },
];

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
    Object.assign(billingState, defaultBillingState);
    billingState.availablePlans = [{ key: "free", name: "Free" }];
    refetchBilling.mockClear();
    openCardSetup.mockReset();
    activateOnboardingPlan.mockClear();
    resumeOnboardingPlan.mockClear();
    localStorage.clear();
    githubPending = false;
    githubLinked = true;
    canManageBilling = true;
    permissionsLoading = false;
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
    accountOrganizations = twoOrganizations;

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-log-out")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-workspace-switch")).not.toBeInTheDocument();
  });

  it("shows both switches when another workspace and another organization exist", () => {
    factories = [factory, { id: "factory-2", key: "CORE", name: "Core" }];
    accountOrganizations = twoOrganizations;

    renderSetup(pageModel());

    expect(screen.getByTestId("first-run-workspace-switch")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-organization-switch")).toBeInTheDocument();
  });

  it("opens the current organization from the switch menu so the user can leave setup", async () => {
    accountOrganizations = twoOrganizations;
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

  it.each([
    { enabled: false, planActive: false },
    { enabled: true, planActive: false },
    { enabled: true, planActive: true },
  ])("keeps repository and intake before the final model choice (%j)", async ({ enabled, planActive }) => {
    Object.assign(billingState, { enabled, planActive, byok: true, availablePlans: [] });
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
    // Nothing is highlighted until the user picks a source.
    const githubIssues = screen.getByRole("button", { name: /GitHub issues/ });
    expect(githubIssues).toHaveAttribute("aria-pressed", "false");
    await user.click(githubIssues);
    // The backlog repository list keeps only the organization of the app repository.
    await user.click(screen.getByTestId("first-run-change-repository"));
    expect(screen.queryByRole("option", { name: /other\/tools/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: /acme\/billing/ }));
    await user.click(screen.getByTestId("first-run-confirm-projects"));
    expect(screen.getByText(FIRST_RUN_COPY.tickets.stepRepositoryDone("acme/billing"))).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await waitFor(() => expect(model.saveIssues).toHaveBeenCalledWith("vcs"));
    expect(await screen.findByTestId("first-run-model-source")).toBeInTheDocument();
    const steps = 5;
    expect(screen.getByTestId("first-run-progress")).toHaveAttribute("aria-label", `Step ${steps} of ${steps}`);
    expect(model.finish).not.toHaveBeenCalled();
    expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();

    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    // Back on the GitHub card returns to the source list, then to the repository.
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByRole("button", { name: /GitHub issues/ })).toHaveAttribute("aria-pressed", "true");
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

  it("starts Pay-as-you-go after the backlog step when a card is saved", async () => {
    Object.assign(billingState, { enabled: true, hasPaymentMethod: true });
    billingState.availablePlans = [{ key: "pay_as_you_go", name: "Pay-as-you-go" }];
    const user = userEvent.setup();
    const model = hostedVcsModel();
    renderSetup(model);
    // The card step follows, so the tickets button does not promise a scan.
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByTestId("first-run-card")).toBeInTheDocument();
    expect(screen.getByTestId("first-run-progress")).toHaveAttribute("aria-label", "Step 5 of 5");
    expect(model.finish).not.toHaveBeenCalled();
    await user.click(screen.getByTestId("first-run-back"));
    expect(screen.getByTestId("first-run-tickets")).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-analyze-tickets"));
    await user.click(await screen.findByTestId("first-run-card-continue"));
    await waitFor(() =>
      expect(activateOnboardingPlan).toHaveBeenCalledWith("org-1", "pay_as_you_go", expect.any(Object)),
    );
    await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
  });

  it.each(["issues", "card"])("lets a maintainer finish unpaid workspace setup from %s", async (step) => {
    Object.assign(billingState, { enabled: true });
    billingState.availablePlans = [{ key: "pay_as_you_go", name: "Pay-as-you-go" }];
    canManageBilling = false;
    const user = userEvent.setup();
    const model = hostedVcsModel();
    const path = `/org-1/workspaces/PAY/setup?step=${step}${step === "card" ? "&plan=starter&card_setup=1&subscribed=1" : ""}`;
    const previousUrl = window.location.href;
    window.history.replaceState({}, "", path);
    try {
      renderSetup(model, path);

      if (step === "issues") await user.click(screen.getByRole("button", { name: "Continue" }));

      expect(await screen.findByTestId("first-run-complete")).toHaveTextContent(
        "Ask an organization admin to select a billing plan. You can complete workspace setup now.",
      );
      expect(screen.queryByTestId("first-run-card")).not.toBeInTheDocument();
      expect(model.finish).not.toHaveBeenCalled();
      await user.click(screen.getByRole("button", { name: "Continue" }));
      await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
      expect(openCardSetup).not.toHaveBeenCalled();
      expect(activateOnboardingPlan).not.toHaveBeenCalled();
      expect(resumeOnboardingPlan).not.toHaveBeenCalled();
    } finally {
      window.history.replaceState({}, "", previousUrl);
    }
  });

  it("disables a restored card step until billing permission loads", async () => {
    Object.assign(billingState, { enabled: true });
    billingState.availablePlans = [{ key: "pay_as_you_go", name: "Pay-as-you-go" }];
    permissionsLoading = true;
    canManageBilling = false;
    const model = hostedVcsModel();
    const path = "/org-1/workspaces/PAY/setup?step=card";
    const { rerender } = renderSetup(model, path);

    expect(screen.getByRole("button", { name: "Continue with Pay-as-you-go" })).toBeDisabled();
    expect(model.finish).not.toHaveBeenCalled();

    permissionsLoading = false;
    canManageBilling = true;
    rerender(
      <MemoryRouter initialEntries={[path]}>
        <FirstRunSetup model={model} />
      </MemoryRouter>,
    );
    expect(await screen.findByTestId("first-run-card")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Continue with Pay-as-you-go" })).toBeEnabled();
    expect(model.finish).not.toHaveBeenCalled();
  });

  it("finishes a Starter checkout return after the saved GitHub connection restores", async () => {
    Object.assign(billingState, { enabled: true, planActive: true });
    billingState.availablePlans = [{ key: "starter", name: "Starter", subscriptionRequired: true }];
    const path = "/org-1/workspaces/PAY/setup?step=card&plan=starter&card_setup=1&subscribed=1";
    const previousUrl = window.location.href;
    window.history.replaceState({}, "", path);
    try {
      const model = hostedVcsModel({ setup: { ...setupState(), connected: new Set() } });
      const { rerender } = renderSetup(model, path);
      expect(model.finish).not.toHaveBeenCalled();

      const restored = { ...model, setup: { ...model.setup, connected: new Set(["github"] as const) } };
      rerender(
        <MemoryRouter initialEntries={[path]}>
          <FirstRunSetup model={restored} />
        </MemoryRouter>,
      );
      await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
      expect(activateOnboardingPlan).not.toHaveBeenCalled();
    } finally {
      window.history.replaceState({}, "", previousUrl);
    }
  });

  it.each(
    ["issues", "billing"].flatMap((step) => [
      { step, enabled: false, availablePlans: [{ key: "pay_as_you_go", name: "Pay-as-you-go" }] },
      { step, enabled: true, availablePlans: [{ key: "free", name: "Free" }] },
      { step, enabled: true, availablePlans: [] },
      { step, enabled: true, availablePlans: undefined },
    ]),
  )("finishes without plan selection (%j)", async ({ step, enabled, availablePlans }) => {
    Object.assign(billingState, { enabled, availablePlans });
    const user = userEvent.setup();
    const model = hostedVcsModel();
    renderSetup(model, `/org-1/workspaces/PAY/setup?step=${step}`);
    await user.click(
      step === "issues"
        ? screen.getByTestId("first-run-analyze-tickets")
        : screen.getByRole("button", { name: "Continue" }),
    );
    await waitFor(() => expect(model.finish).toHaveBeenCalled());
  });

  it.each([false, true])("routes a restored billing URL through model access (own key: %s)", async (byok) => {
    billingState.enabled = true;
    const model = hostedVcsModel({ bringYourOwnKey: byok });
    renderSetup(model, "/org-1/workspaces/PAY/setup?step=billing&plan=team&subscribed=1&checkout_id=old");
    expect(screen.queryByTestId("first-run-card")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-complete")).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: "Continue" }));
    if (byok) {
      expect(await screen.findByTestId("first-run-model-source")).toBeInTheDocument();
      expect(model.finish).not.toHaveBeenCalled();
    } else {
      await waitFor(() => expect(model.finish).toHaveBeenCalledTimes(1));
    }
    expect(billingState.planActive).toBe(false);
  });
});
