import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

beforeAll(() => {
  client.setConfig({ baseUrl: "http://localhost" });
  Element.prototype.scrollIntoView ??= vi.fn();
});

describe("FactorySettingsLayout fleets", () => {
  it("keeps Fleets active on runner detail routes with organization runners disabled", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/fleets/own/runners/runner-1`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationFleetManagement: {
            fleets: [{ id: "own", enabled: false }],
            runners: [{ id: "runner-1", fleetId: "own", state: "idle", ephemeral: true }],
          },
        }}
      />,
    );
    expect(await screen.findByRole("heading", { name: "Runner runner-1" })).toBeInTheDocument();
    const sidebar = await screen.findByTestId("factory-settings-sidebar");
    expect(within(sidebar).getByRole("link", { name: "Fleets" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Terminate runner" })).toBeInTheDocument();
  });

  it("opens enabled fleet management with administrator actions in the settings harness", async () => {
    render(
      <FactoriesHarness
        experimentalFeatures={["organization_runners"]}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/fleets/own`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationFleetManagement: {
            fleets: [{ id: "own", enabled: true }],
            runners: [],
          },
        }}
      />,
    );
    expect(await screen.findByRole("button", { name: "Connect runner" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Edit fleet" })).toBeInTheDocument();
  });
});
