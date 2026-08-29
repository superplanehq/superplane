import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_LINE_PLAN_ID,
} from "../../__fixtures__/factoryPageResponses";
import { factoryHomePath, factorySettingsPath } from "../../lib/factoryPagePaths";

describe("OrganizationSettingsWorkspacesPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("lists workspace names as links and hides the key column", async () => {
    render(<FactoriesHarness pathSuffix="organization/workspaces" factoriesFixture={defaultFactoriesFixture} />);

    const table = await screen.findByTestId("organization-settings-workspaces-table", {}, { timeout: 8000 });
    const semaphore = within(table).getByTestId(`organization-settings-workspace-name-${PRIMARY_FACTORY_ID}`);

    expect(semaphore).toHaveTextContent("Semaphore");
    expect(semaphore).toHaveAttribute(
      "href",
      factoryHomePath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY, REFUND_LINE_PLAN_ID),
    );
    expect(table).not.toHaveTextContent("RF");
    expect(table).not.toHaveTextContent("PF");
    expect(within(table).queryByText("Key")).not.toBeInTheDocument();
    expect(within(table).queryByText("Created by")).not.toBeInTheDocument();
  }, 10000);

  it("opens workspace settings from the cog", async () => {
    render(<FactoriesHarness pathSuffix="organization/workspaces" factoriesFixture={defaultFactoriesFixture} />);

    const settings = await screen.findByTestId(
      `organization-settings-workspace-settings-${PRIMARY_FACTORY_ID}`,
      {},
      { timeout: 8000 },
    );

    expect(settings).toHaveAttribute("href", factorySettingsPath(FACTORIES_ORGANIZATION_ID, PRIMARY_FACTORY_KEY));
  }, 10000);

  it("asks to confirm before deleting a workspace", async () => {
    const user = userEvent.setup();
    render(<FactoriesHarness pathSuffix="organization/workspaces" factoriesFixture={defaultFactoriesFixture} />);

    const deleteButton = await screen.findByTestId(
      `organization-settings-workspace-delete-${PRIMARY_FACTORY_ID}`,
      {},
      { timeout: 8000 },
    );
    await user.click(deleteButton);

    expect(await screen.findByText("Do you really want to delete the workspace?")).toBeInTheDocument();

    await user.click(screen.getByTestId("factory-delete-confirm-button"));

    const table = await screen.findByTestId("organization-settings-workspaces-table");
    expect(within(table).queryByText("Semaphore")).not.toBeInTheDocument();
    expect(within(table).getByText("SuperPlane")).toBeInTheDocument();
  }, 10000);
});
