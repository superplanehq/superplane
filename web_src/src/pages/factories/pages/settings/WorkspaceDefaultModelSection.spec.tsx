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
          factoriesFixture={defaultFactoriesFixture}
          experimentalFeatures={agentPageFeatures}
        />
      </>,
    );

    const select = await screen.findByTestId("agent-default-model-select", {}, { timeout: 8000 });
    expect(select).toHaveTextContent("No default model");
    await user.click(select);
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
            ...defaultFactoriesFixture,
            failFactoryUpdate: true,
            factories: defaultFactoriesFixture.factories.map((factory) =>
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
});
