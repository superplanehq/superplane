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
  it("keeps Fleets active on the fleet page with organization runners disabled", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/fleets/own`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationFleetManagement: {
            fleets: [{ id: "own", enabled: false }],
            runners: [{ id: "runner-1", fleetId: "own", state: "busy", ephemeral: true }],
          },
        }}
      />,
    );
    expect(await screen.findByText("runner-1")).toBeInTheDocument();
    const sidebar = await screen.findByTestId("factory-settings-sidebar");
    expect(within(sidebar).getByRole("link", { name: "Fleets" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Terminate runner" })).toBeInTheDocument();
  });

  it("puts the create action in the Fleets page header", async () => {
    render(
      <FactoriesHarness
        experimentalFeatures={["organization_runners"]}
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/fleets`}
        factoriesFixture={{ ...defaultFactoriesFixture, organizationFleetManagement: { fleets: [], runners: [] } }}
      />,
    );
    const create = await screen.findByRole("button", { name: "Create fleet" });
    expect(screen.getAllByRole("heading", { name: "Fleets" })).toHaveLength(1);
    expect(
      within(screen.getByTestId("workspace-page-header-actions")).getByRole("button", { name: "Create fleet" }),
    ).toBe(create);
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
    expect(screen.getByRole("button", { name: "Change runner version" })).toBeInTheDocument();
  });
});
