import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";

describe("FactorySettingsRepositoryPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("selects and saves the workspace repository", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    expect(await within(settings).findByRole("option", { name: /acme\/api/i })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    fireEvent.click(within(settings).getByRole("option", { name: /acme\/web/i }));
    const saveButton = within(settings).getByTestId("factory-settings-repository-save");
    await waitFor(() => expect(saveButton).toBeEnabled());
    fireEvent.click(saveButton);

    await waitFor(() => expect(saveButton).toBeDisabled());
    expect(within(settings).getByRole("option", { name: /acme\/web/i })).toHaveAttribute("aria-selected", "true");
  }, 10000);
});
