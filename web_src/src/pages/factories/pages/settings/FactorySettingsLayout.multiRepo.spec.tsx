import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_FACTORY_MULTI_REPO_TASKS } from "@/lib/experimentalFeatures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsLayout multi-repo navigation", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it.each([false, true])("keeps Repository discoverable only when multi-repo is off (enabled: %s)", async (enabled) => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        experimentalFeatures={enabled ? [FEATURE_FACTORY_MULTI_REPO_TASKS] : []}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar");
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-models")).toBeInTheDocument();
    const repository = within(sidebar).queryByTestId("factory-settings-nav-workspace-repository");
    if (enabled) {
      expect(repository).not.toBeInTheDocument();
    } else {
      expect(repository).toBeInTheDocument();
    }

    await user.type(within(sidebar).getByTestId("factory-settings-find"), "repository");
    if (enabled) {
      expect(within(sidebar).getByTestId("factory-settings-find-empty")).toBeInTheDocument();
    } else {
      expect(within(sidebar).getByTestId("factory-settings-search-results")).toBeInTheDocument();
    }
  });
});
