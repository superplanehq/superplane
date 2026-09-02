import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";
import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_KEY,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
} from "../../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../../__fixtures__/setupStoryFixtures";

describe("Settings redesign chrome", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => undefined;
    Element.prototype.releasePointerCapture ??= () => undefined;
  });

  it("finds settings without sidebar captions", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const sidebar = await screen.findByTestId("factory-settings-sidebar", {}, { timeout: 8000 });
    expect(within(sidebar).getByTestId("factory-settings-find")).toBeInTheDocument();
    expect(within(sidebar).getByTestId("factory-settings-workspace-nav")).toHaveTextContent("Workspace");
    expect(within(sidebar).getByTestId("factory-settings-workspace-nav")).not.toHaveTextContent("Semaphore");

    await user.type(within(sidebar).getByTestId("factory-settings-find"), "spend");
    expect(within(sidebar).getByTestId("factory-settings-nav-workspace-spending")).toBeInTheDocument();
    expect(within(sidebar).queryByTestId("factory-settings-nav-workspace-general")).not.toBeInTheDocument();
  }, 10000);

  it("shows a workspace identity hero and URL key", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("settings-redesign-identity-hero", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("factory-settings-key")).toHaveValue("RF");
    expect(screen.getByText("/superplane/workspaces/")).toBeInTheDocument();
  }, 10000);

  it("lets you edit the organization name and lists workspaces", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const name = await screen.findByTestId("organization-settings-overview-name", {}, { timeout: 8000 });
    expect(name).toHaveValue("SuperPlane");
    expect(screen.getByTestId("settings-redesign-org-workspaces")).toHaveTextContent("Semaphore");
    expect(screen.getByTestId("settings-redesign-org-danger")).toBeInTheDocument();
    expect(screen.queryByTestId("settings-redesign-org-delete-confirm")).not.toBeInTheDocument();
  }, 10000);

  it("asks you to type the organization name in a delete dialog", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    await user.click(await screen.findByTestId("settings-redesign-org-delete", {}, { timeout: 8000 }));
    expect(screen.getByText("Type SuperPlane to confirm")).toBeInTheDocument();
    const confirm = screen.getByTestId("settings-redesign-org-delete-confirm");
    const submit = screen.getByTestId("settings-redesign-org-delete-submit");
    expect(submit).toBeDisabled();
    await user.type(confirm, "SuperPlane");
    expect(submit).toBeEnabled();
  }, 10000);

  it("keeps the workspace identity form without proposed actions", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/general`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByTestId("factory-settings-general-form", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("settings-redesign-identity-hero")).toBeInTheDocument();
    expect(screen.getByTestId("factory-settings-danger-zone")).toBeInTheDocument();
    expect(screen.queryByTestId("settings-proposed")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-redesign-archive")).not.toBeInTheDocument();
  }, 10000);

  it("lists mocked repositories", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    expect(await within(settings).findByRole("option", { name: /acme\/api/ }, { timeout: 8000 })).toBeInTheDocument();
    expect(within(settings).getByRole("option", { name: /acme\/web/ })).toBeInTheDocument();
    expect(within(settings).queryByText("Connect GitHub during workspace setup")).not.toBeInTheDocument();
  }, 10000);

  it("shows organization API keys from factory settings", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/api-keys`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    expect(await screen.findByText("CI pipeline", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("Local CLI")).toBeInTheDocument();
  }, 10000);

  it("changes a member role", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/members`}
        factoriesFixture={defaultFactoriesFixture}
      />,
    );

    const roleSelect = await screen.findByTestId(
      `settings-redesign-member-role-${REVIEWER_USER.id}`,
      {},
      { timeout: 8000 },
    );
    expect(roleSelect).toHaveTextContent("Admin");
    await user.click(roleSelect);
    await user.click(screen.getByRole("option", { name: "Member" }));
    expect(roleSelect).toHaveTextContent("Member");
    expect(screen.getByTestId(`settings-redesign-member-role-${STORYBOOK_ME_USER_ID}`)).toBeDisabled();
  }, 10000);
});
