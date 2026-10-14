import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { WorkspaceDefaultModelSection } from "./WorkspaceDefaultModelSection";

vi.mock("@monaco-editor/react", () => ({
  Editor: () => <div data-testid="monaco-mock" />,
}));

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;
const agentPageFeatures = [FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];

describe("workspace agent preview fields", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens the model selector from its label", async () => {
    const user = userEvent.setup();
    render(<WorkspaceDefaultModelSection canUpdate models={[{ value: "claude-sonnet", label: "Claude Sonnet" }]} />);

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
});
