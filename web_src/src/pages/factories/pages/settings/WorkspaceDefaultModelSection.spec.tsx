import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "bun:test";
import { Toaster } from "sonner";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;
const agentPageFeatures = [FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS];

describe("workspace default model", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("saves a choice, clears it, and restores the previous value when save fails", async () => {
    const user = userEvent.setup();
    const { unmount } = render(
      <>
        <Toaster />
        <FactoriesHarness
          pathSuffix={agentPath}
          factoriesFixture={hostedDefaultFixture()}
          experimentalFeatures={agentPageFeatures}
        />
      </>,
    );

    const select = await screen.findByTestId("agent-default-model-select", {}, { timeout: 8000 });
    expect(select).toHaveTextContent("No default model");
    await user.click(select);
    expect(screen.queryByRole("option", { name: "Your keys: anthropic/claude-sonnet-4-6" })).not.toBeInTheDocument();
    await user.click(await screen.findByRole("option", { name: "SuperPlane: openai/gpt-5" }));
    expect(await screen.findByTestId("agent-default-model-select")).toHaveTextContent("SuperPlane: openai/gpt-5");

    await user.click(screen.getByTestId("agent-default-model-select"));
    await user.click(await screen.findByRole("option", { name: "No default model" }));
    expect(await screen.findByTestId("agent-default-model-select")).toHaveTextContent("No default model");

    unmount();
    render(
      <>
        <Toaster />
        <FactoriesHarness
          pathSuffix={agentPath}
          factoriesFixture={{
            ...hostedDefaultFixture(),
            failFactoryUpdate: true,
            factories: hostedDefaultFixture().factories.map((factory) =>
              factory.id === PRIMARY_FACTORY_ID
                ? { ...factory, defaultAgentModel: "gpt-5", defaultAgentModelProvider: "openai" }
                : factory,
            ),
          }}
          experimentalFeatures={agentPageFeatures}
        />
      </>,
    );

    const failedSelect = await screen.findByTestId("agent-default-model-select", {}, { timeout: 8000 });
    expect(failedSelect).toHaveTextContent("SuperPlane: openai/gpt-5");
    await user.click(failedSelect);
    await user.click(await screen.findByRole("option", { name: "No default model" }));
    expect(await screen.findByText("SuperPlane could not save the default model. Try again.")).toBeInTheDocument();
    expect(screen.getByTestId("agent-default-model-select")).toHaveTextContent("SuperPlane: openai/gpt-5");
  }, 15000);

  it("offers only models the active source can use", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={codexDefaultFixture()}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-default-model-select", {}, { timeout: 8000 }));
    expect(await screen.findByRole("option", { name: "Your keys: openai/gpt-5" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "SuperPlane: openai/gpt-5" })).not.toBeInTheDocument();
  }, 15000);

  it("offers models for the saved connection when the connection list is unavailable", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        orgIntegrations={anthropicOnlyConnections}
        factoriesFixture={secondOpenRouterFixture()}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-default-model-select", {}, { timeout: 8000 }));
    expect(await screen.findByRole("option", { name: "Your keys: moonshotai/kimi-k2.6" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Your keys: anthropic/claude-sonnet-4-6" })).not.toBeInTheDocument();
  }, 15000);
});

function hostedDefaultFixture() {
  return withAgentHarness(defaultFactoriesFixture, "AGENT_HARNESS_SUPERPLANE");
}

function codexDefaultFixture() {
  return withAgentHarness(defaultFactoriesFixture, "AGENT_HARNESS_CODEX");
}

function withAgentHarness(
  fixture: typeof defaultFactoriesFixture,
  agentHarness: "AGENT_HARNESS_SUPERPLANE" | "AGENT_HARNESS_CODEX" | "AGENT_HARNESS_CLAUDE_CODE",
  agentIntegrationId?: string,
) {
  return {
    ...fixture,
    factories: fixture.factories.map((factory) =>
      factory.id === PRIMARY_FACTORY_ID
        ? { ...factory, onboarding: { ...factory.onboarding, agentHarness, agentIntegrationId } }
        : factory,
    ),
  };
}

const anthropicOnlyConnections = [
  {
    metadata: { id: "claude-1", name: "anthropic-key", integrationName: "claude" },
    status: { state: "ready" as const },
  },
];

function secondOpenRouterFixture() {
  const fixture = withAgentHarness(defaultFactoriesFixture, "AGENT_HARNESS_CLAUDE_CODE", "openrouter-2");
  return {
    ...fixture,
    factories: fixture.factories.map((factory) =>
      factory.id === PRIMARY_FACTORY_ID ? { ...factory, agentModelSource: "openrouter" } : factory,
    ),
    byokConnectedProviders: ["anthropic", "openrouter"],
    byokCandidatesByProvider: {
      anthropic: ["claude-sonnet-4-6"],
      openrouter: ["moonshotai/kimi-k2.6"],
    },
  };
}
