import { render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesFactory } from "@/api-client";

import { FIRST_RUN_COPY } from "./first-run/firstRunCopy";
import { FirstRunSetup } from "./FirstRunSetup";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import type { useOnboardingPageModel } from "./useOnboardingPageModel";

type OnboardingPageModel = ReturnType<typeof useOnboardingPageModel>;

let factory: FactoriesFactory;

vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "PAY",
    factory,
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "user-1", name: "Ada Lovelace", email: "ada@example.com" } }),
}));

vi.mock("@/posthog", () => ({ posthog: { reset: vi.fn() } }));

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
    integrationDialogs: null,
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

function renderSetup(model: OnboardingPageModel) {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/PAY/setup?step=issues"]}>
      <FirstRunSetup model={model} />
    </MemoryRouter>,
  );
}

describe("FirstRunSetup", () => {
  beforeEach(() => {
    factory = { id: "factory-1", onboarding: { vcsIntegrationId: "github-1" } };
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
});
