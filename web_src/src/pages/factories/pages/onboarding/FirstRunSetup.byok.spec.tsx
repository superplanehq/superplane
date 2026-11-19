import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "bun:test";

import { factoriesDescribeFactory, type FactoriesFactoryCodexConnection } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  factoriesFixtureWithSetupAnswers,
  SETUP_ANSWERS,
} from "../../__fixtures__/setupStoryFixtures";
import { FEATURE_ORGANIZATION_BYOK } from "@/lib/experimentalFeatures";
import { HOSTED_DEFAULT_USAGE_REPORT, NO_GRANT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { OnboardingPage } from "./OnboardingPage";

const installFactory = vi.hoisted(() => vi.fn().mockResolvedValue({ canvasId: "agent-app", canvasName: "agent-app" }));
vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory, isInstalling: false }),
}));

let connectionResponse: FactoriesFactoryCodexConnection | undefined;

beforeEach(() => {
  connectionResponse = undefined;
  installFactory.mockClear();
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (request) => {
      if (
        request instanceof Request &&
        request.method === "GET" &&
        request.url.endsWith("/me/codex-connection") &&
        connectionResponse
      ) {
        return Response.json({ connection: connectionResponse });
      }
      return globalThis.fetch(request);
    },
  });
  Element.prototype.scrollIntoView ??= vi.fn();
  localStorage.clear();
});

it("finishes with a provider key and model without a hosted default or BYOK flag", async () => {
  const user = userEvent.setup();
  const fixture = factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.issues, {
    organizationWorkspaceUsage: NO_GRANT_USAGE_REPORT,
  });
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID ? { ...factory, lines: [] } : factory,
  );
  fixture.appsByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  fixture.intakesByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  fixture.organizationBilling = { billingEnabled: false, productFeatures: { byo_key: false } };
  fixture.byokConnectedProviders = ["anthropic"];
  localStorage.setItem(`superplane:onboarding-model-source:${PRIMARY_FACTORY_ID}`, "subscription");
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID
      ? { ...factory, onboarding: { ...factory.onboarding, codexSubscription: true } }
      : factory,
  );
  fixture.byokCandidatesByProvider = { anthropic: ["claude-sonnet-4-6", "claude-opus-5-5"] };
  render(
    <FactoriesHarness
      mode="app"
      billingEnabled={false}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=issues`}
      factoriesFixture={fixture}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      pageOverrides={{ onboarding: OnboardingPage }}
    />,
  );

  await user.click(await screen.findByTestId("first-run-analyze-tickets", {}, { timeout: 8000 }));
  expect(await screen.findByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Connect OpenAI" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Connect OpenRouter" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Connect Custom provider" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Connect Anthropic" }));
  await waitFor(() => expect(screen.getByRole("combobox", { name: "Model" })).toBeEnabled());
  expect(screen.getByRole("combobox", { name: "Model" })).not.toHaveTextContent("claude-opus-5-5");
  await user.click(screen.getByRole("combobox", { name: "Model" }));
  await user.click(screen.getByRole("option", { name: "claude-opus-5-5" }));
  await user.click(screen.getByTestId("first-run-finish-setup"));

  await waitFor(async () => {
    const saved = await factoriesDescribeFactory({ path: { id: PRIMARY_FACTORY_ID } });
    expect(saved.data?.factory?.onboarding?.completedAt).toBeTruthy();
    expect(saved.data?.factory?.onboarding?.agentIntegrationId).toBe("storybook-claude-connection");
    expect(saved.data?.factory?.onboarding?.codexSubscription).toBe(false);
  });
  expect(installFactory).toHaveBeenCalledWith(
    expect.objectContaining({
      agentRewrite: {
        component: "runnerClaudeCode",
        model: "claude-opus-5-5",
        planningModel: "claude-opus-5-5",
        credentials: { source: "integration", name: "acme-claude" },
      },
    }),
  );
}, 10000);

it("connects a subscription, restores it after navigation, and provisions Codex", async () => {
  const user = userEvent.setup();
  const fixture = factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.issues, {
    organizationWorkspaceUsage: NO_GRANT_USAGE_REPORT,
  });
  fixture.organizationBilling = { billingEnabled: false, productFeatures: { byo_key: false } };
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID ? { ...factory, lines: [] } : factory,
  );
  fixture.appsByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  fixture.intakesByFactoryId = { [PRIMARY_FACTORY_ID]: [] };
  const mount = (step: string) =>
    render(
      <FactoriesHarness
        mode="app"
        billingEnabled={false}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=${step}`}
        factoriesFixture={fixture}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        pageOverrides={{ onboarding: OnboardingPage }}
      />,
    );
  const first = mount("issues");
  await user.click(await screen.findByTestId("first-run-analyze-tickets", {}, { timeout: 8000 }));
  expect(await screen.findByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Connect OpenAI" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Connect OpenRouter" })).toBeInTheDocument();
  expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "Connect Codex" }));
  let dialog = await screen.findByRole("dialog");
  await user.click(within(dialog).getByRole("button", { name: "Connect ChatGPT" }));
  expect(await screen.findByTestId("first-run-codex-code")).toBeInTheDocument();
  await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
  await waitFor(() => expect(screen.queryByRole("dialog") === null).toBe(true));
  expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "Connect Codex" }));
  dialog = await screen.findByRole("dialog");
  await user.click(within(dialog).getByRole("button", { name: "Connect ChatGPT" }));
  await screen.findByTestId("first-run-codex-code");
  connectionResponse = { eligible: true, status: "ready", availableModels: ["subscription-only-model"] };
  await waitFor(() => expect(screen.queryByRole("dialog") === null).toBe(true), { timeout: 6000 });
  expect(screen.getByTestId("first-run-finish-setup")).toBeEnabled();
  await user.click(screen.getByTestId("first-run-back"));
  expect(await screen.findByTestId("first-run-tickets")).toBeInTheDocument();
  await user.click(screen.getByTestId("first-run-analyze-tickets"));
  expect(await screen.findByRole("button", { name: /Codex subscription/ })).toHaveAttribute("aria-pressed", "true");
  first.unmount();
  mount("agent");
  expect(await screen.findByRole("button", { name: /Codex subscription/ })).toHaveAttribute("aria-pressed", "true");
  await waitFor(() => expect(screen.getByTestId("first-run-finish-setup")).toBeEnabled());
  await user.click(screen.getByTestId("first-run-finish-setup"));
  await waitFor(async () => {
    const saved = await factoriesDescribeFactory({ path: { id: PRIMARY_FACTORY_ID } });
    expect(saved.data?.factory?.onboarding?.completedAt).toBeTruthy();
    expect(saved.data?.factory?.onboarding?.codexSubscription).toBe(true);
    expect(saved.data?.factory?.onboarding?.agentHarness).toBe("AGENT_HARNESS_CODEX");
    expect(saved.data?.factory?.onboarding?.agentIntegrationId).toBe("");
  });
  expect(installFactory).toHaveBeenCalledWith(
    expect.objectContaining({
      agentRewrite: {
        component: "runnerCodex",
        model: "",
        planningModel: "",
        credentials: { source: "subscription" },
      },
    }),
  );
}, 20000);

it.each([
  { status: "pending", eligible: true },
  { status: "ready", eligible: true, reservationHeld: true, availableModels: ["subscription-model"] },
  { status: "disconnected", eligible: true },
  { status: "reconnect_required", eligible: true },
  { status: "ready", eligible: false, availableModels: ["subscription-model"] },
  { status: "ready", eligible: true, availableModels: [] },
])("blocks subscription completion for connection %j even with an API key and credit", async (connection) => {
  const fixture = factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.issues, {
    organizationWorkspaceUsage: { ...NO_GRANT_USAGE_REPORT, remainingCreditCents: "5000" },
  });
  fixture.codexConnection = connection;
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID
      ? { ...factory, onboarding: { ...factory.onboarding, codexSubscription: true } }
      : factory,
  );
  fixture.organizationBilling = { billingEnabled: false, productFeatures: { byo_key: false } };
  render(
    <FactoriesHarness
      mode="app"
      billingEnabled={false}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=agent`}
      factoriesFixture={fixture}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      pageOverrides={{ onboarding: OnboardingPage }}
    />,
  );
  expect(await screen.findByRole("button", { name: /Codex subscription/ }, { timeout: 8000 })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  expect(screen.getByTestId("first-run-finish-setup")).toBeDisabled();
  expect(installFactory).not.toHaveBeenCalled();
});

it("ignores stale subscriptions on hosted onboarding even when BYOK is allowed", async () => {
  const fixture = factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.issues, {
    organizationWorkspaceUsage: HOSTED_DEFAULT_USAGE_REPORT,
  });
  fixture.organizationBilling = { billingEnabled: false, productFeatures: { byo_key: true } };
  fixture.factories = fixture.factories.map((factory) =>
    factory.id === PRIMARY_FACTORY_ID
      ? { ...factory, onboarding: { ...factory.onboarding, codexSubscription: true } }
      : factory,
  );
  localStorage.setItem(`superplane:onboarding-model-source:${PRIMARY_FACTORY_ID}`, "subscription");
  render(
    <FactoriesHarness
      mode="app"
      billingEnabled={false}
      experimentalFeatures={[FEATURE_ORGANIZATION_BYOK]}
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/setup?step=agent`}
      factoriesFixture={fixture}
      orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      pageOverrides={{ onboarding: OnboardingPage }}
    />,
  );
  expect(await screen.findByTestId("first-run-model-source", {}, { timeout: 8000 })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /Codex subscription/ })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Connect Codex" })).not.toBeInTheDocument();
  await userEvent.setup().click(screen.getByTestId("first-run-finish-setup"));
  await waitFor(async () => {
    const saved = await factoriesDescribeFactory({ path: { id: PRIMARY_FACTORY_ID } });
    expect(saved.data?.factory?.onboarding?.completedAt).toBeTruthy();
    expect(saved.data?.factory?.onboarding?.codexSubscription).toBe(false);
    expect(saved.data?.factory?.onboarding?.agentHarness).toBe("AGENT_HARNESS_SUPERPLANE");
  });
});
