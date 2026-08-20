import { render, screen, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";

describe("FactorySettingsLayout sidebar", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("shows workspace settings without the account menu", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-back")).toHaveTextContent("Back to workspace");
    expect(within(sidebar).queryByText("Semaphore")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("Workspace")).not.toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-workspace-nav")).toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-profile-title")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-you-section")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-profile")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-back-to-apps")).not.toBeInTheDocument();
    expect(within(sidebar).queryByLabelText("Appearance")).not.toBeInTheDocument();
  }, 10000);

  it("shows only profile settings when Profile opens", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/profile`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    const backLink = within(sidebar).getByTestId("factory-settings-back");
    const title = within(sidebar).getByTestId("factory-settings-profile-title");
    const selected = within(sidebar).getByTestId("factory-settings-nav-profile");

    expect(backLink).toHaveTextContent("Back to workspace");
    expect(title).toHaveTextContent("Profile settings");
    expect(within(sidebar).queryByTestId("factory-settings-you-section")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("Leonardo DiCaprio")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-back-to-apps")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-sign-out")).not.toBeInTheDocument();
    expect(within(sidebar).queryByLabelText("Appearance")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-workspace-nav")).not.toBeInTheDocument();
    expect(within(sidebar).queryByText("Semaphore")).not.toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-repositories")).not.toBeInTheDocument();
    expect(selected).toHaveAttribute("aria-current", "page");
    expect(backLink.compareDocumentPosition(title) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(title.compareDocumentPosition(selected) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    const profile = await screen.findByTestId("factory-settings-profile-form");
    expect(within(profile).getAllByText("Leonardo DiCaprio").length).toBeGreaterThan(0);
    expect(within(profile).getByText("Name")).toBeInTheDocument();
    expect(within(profile).getByText("Email address")).toBeInTheDocument();
    expect(within(profile).getByText("john.doe@superplane.dev")).toBeInTheDocument();
    expect(within(profile).getByRole("img", { name: "Leonardo DiCaprio" })).toHaveAttribute(
      "src",
      "/storybook/leonardo-dicaprio.jpg",
    );
  }, 10000);
});
