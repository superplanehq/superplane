import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsLayout repository navigation", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));

  it("shows Agent instead of the retired Repository settings entry", async () => {
    render(<FactoriesHarness pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`} />);
    const sidebar = await screen.findByTestId("factory-settings-sidebar");
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-models")).toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-repository")).not.toBeInTheDocument();
  });
});
