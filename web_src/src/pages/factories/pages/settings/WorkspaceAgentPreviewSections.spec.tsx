import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { WorkspaceDefaultModelSection } from "./WorkspaceDefaultModelSection";

vi.mock("@/hooks/useSelectableLLMModels", () => ({
  useSelectableLLMModels: () => ({
    data: [
      {
        source: { id: "hosted", name: "SuperPlane" },
        provider: { id: "anthropic", name: "Anthropic" },
        model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
        key: "hosted::anthropic::claude-sonnet-4-6",
        label: "Claude Sonnet",
      },
    ],
    isLoading: false,
  }),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: () => <div data-testid="monaco-mock" />,
}));

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;
const agentPageFeatures = [FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_SKILLS];

describe("workspace agent preview fields", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens the model selector from its label", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <WorkspaceDefaultModelSection organizationId="org-1" factoryId="factory-1" canUpdate />
      </QueryClientProvider>,
    );

    expect(screen.getByRole("combobox", { name: "Default model" })).toBeInTheDocument();
    await user.click(screen.getByText("Default model"));

    expect(screen.getByRole("listbox")).toBeInTheDocument();
  });

  it("opens the setup steps editor from the Agent page", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
        }}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-setup-steps-edit", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-setup-steps-editor", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 10000);

  it("hides setup and context cards without the workspace agent experiment", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
        }}
        experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
      />,
    );

    expect(await screen.findByTestId("factory-settings-agent", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByTestId("agent-setup-steps")).not.toBeInTheDocument();
    expect(screen.queryByTestId("agent-cross-repo")).not.toBeInTheDocument();
  }, 10000);

  it("shows load errors when the resource request fails", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          failAgentResourcesListForFactoryIds: [PRIMARY_FACTORY_ID],
        }}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    expect(
      await screen.findByText("SuperPlane could not load setup steps. Try again.", {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 10000);

  it("shows a load error in the setup editor when the request fails", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${agentPath}/setup-steps`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          failAgentResourcesListForFactoryIds: [PRIMARY_FACTORY_ID],
        }}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    expect(
      await screen.findByText("SuperPlane could not load setup steps. Try again.", {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 10000);
});
