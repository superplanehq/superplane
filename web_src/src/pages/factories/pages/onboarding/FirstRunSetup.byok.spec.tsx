import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, expect, it, vi } from "bun:test";

import { factoriesDescribeFactory } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_ID, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  factoriesFixtureWithSetupAnswers,
  SETUP_ANSWERS,
} from "../../__fixtures__/setupStoryFixtures";
import { NO_GRANT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { OnboardingPage } from "./OnboardingPage";

const installFactory = vi.hoisted(() => vi.fn().mockResolvedValue({ canvasId: "agent-app", canvasName: "agent-app" }));
vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory, isInstalling: false }),
}));

beforeAll(() => {
  client.setConfig({ baseUrl: "http://localhost" });
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
