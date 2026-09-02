import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsAutomationsPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("opens the automations list under workspace settings", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/automations`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-automations")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(await screen.findByTestId("settings-redesign-automations", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Automations" })).toBeInTheDocument();
  }, 10000);
});
