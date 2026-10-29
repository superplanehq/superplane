import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "bun:test";

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
    const dialog = await screen.findByTestId("factory-settings-update-tasks-dialog");
    expect(dialog).toHaveTextContent(/acme\/api/);
    expect(dialog).toHaveTextContent(/acme\/web/);
    expect(dialog).toHaveTextContent("Save without updating tasks keeps current tasks on their repository and branch.");
    fireEvent.click(await screen.findByTestId("factory-settings-update-tasks-confirm"));

    await waitFor(() => expect(saveButton).toBeDisabled());
    expect(within(settings).getByRole("option", { name: /acme\/web/i })).toHaveAttribute("aria-selected", "true");
  }, 10000);

  it("shows only repositories accessible to the linked account for a hosted binding", async () => {
    const fixture = factoriesFixtureWithSetupAnswers({
      ...SETUP_ANSWERS.agent,
      appRepositoryId: "201",
      backlogRepositoryId: "201",
    });
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={fixture}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    expect(await within(settings).findByRole("option", { name: /acme\/api/i })).toBeInTheDocument();
    expect(within(settings).queryByRole("option", { name: /acme\/web/i })).not.toBeInTheDocument();
  }, 10000);

  it("lists integration repositories when catalog IDs are unset zero strings", async () => {
    // The API serializes unset int64 IDs as "0". A Bitbucket workspace without
    // catalog bindings must still list its integration repositories.
    const fixture = factoriesFixtureWithSetupAnswers({
      ...SETUP_ANSWERS.agent,
      vcsProvider: "bitbucket",
      appRepositoryId: "0",
      backlogRepositoryId: "0",
    });
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={fixture}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    expect(await within(settings).findByRole("option", { name: /acme\/api/i })).toBeInTheDocument();
    expect(within(settings).getByRole("option", { name: /acme\/web/i })).toBeInTheDocument();
  }, 10000);

  it("enables save when the branch changes and sends that branch with the current repository", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    const saves: { repository?: string; defaultBranch?: string; retargetTasks?: boolean }[] = [];
    const currentFetch = globalThis.fetch;
    globalThis.fetch = async (input, init) => {
      const target = requestTarget(input);
      if (target.includes("/repository") && requestMethodFrom(input, init) !== "GET") {
        saves.push(await readSaveBody(input, init));
      }
      return currentFetch(input, init);
    };
    try {
      const branch = await within(settings).findByTestId("factory-settings-default-branch");
      await user.click(branch);
      await user.type(await screen.findByPlaceholderText("Find a branch"), "dev");
      expect(screen.queryByRole("option", { name: "main" })).not.toBeInTheDocument();
      await user.click(await screen.findByRole("option", { name: "develop" }));
      const saveButton = within(settings).getByTestId("factory-settings-repository-save");
      await waitFor(() => expect(saveButton).toBeEnabled());
      await user.click(saveButton);
      await user.click(await screen.findByTestId("factory-settings-update-tasks-confirm"));

      await waitFor(() =>
        expect(saves).toEqual([{ repository: "acme/api", defaultBranch: "develop", retargetTasks: true }]),
      );
    } finally {
      globalThis.fetch = currentFetch;
    }
  }, 10000);

  it("saves a branch without moving unfinished tasks", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/repository`}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.agent)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
      />,
    );

    const settings = await screen.findByTestId("factory-settings-repository", {}, { timeout: 8000 });
    const saves: { repository?: string; defaultBranch?: string; retargetTasks?: boolean }[] = [];
    const currentFetch = globalThis.fetch;
    globalThis.fetch = async (input, init) => {
      const target = requestTarget(input);
      if (target.includes("/repository") && requestMethodFrom(input, init) !== "GET") {
        saves.push(await readSaveBody(input, init));
      }
      return currentFetch(input, init);
    };
    try {
      const branch = await within(settings).findByTestId("factory-settings-default-branch");
      await user.click(branch);
      await user.click(await screen.findByRole("option", { name: "develop" }));
      const saveButton = within(settings).getByTestId("factory-settings-repository-save");
      await waitFor(() => expect(saveButton).toBeEnabled());
      await user.click(saveButton);
      await user.click(await screen.findByTestId("factory-settings-update-tasks-keep"));

      await waitFor(() =>
        expect(saves).toEqual([{ repository: "acme/api", defaultBranch: "develop", retargetTasks: false }]),
      );
    } finally {
      globalThis.fetch = currentFetch;
    }
  }, 10000);
});

function requestTarget(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  if (input instanceof URL) return input.href;
  return input.url;
}

function requestMethodFrom(input: RequestInfo | URL, init?: RequestInit): string {
  return (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
}

async function readSaveBody(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<{ repository?: string; defaultBranch?: string; retargetTasks?: boolean }> {
  if (typeof init?.body === "string" && init.body) {
    return JSON.parse(init.body) as { repository?: string; defaultBranch?: string; retargetTasks?: boolean };
  }
  if (input instanceof Request) {
    return (await input.clone().json()) as {
      repository?: string;
      defaultBranch?: string;
      retargetTasks?: boolean;
    };
  }
  return {};
}
