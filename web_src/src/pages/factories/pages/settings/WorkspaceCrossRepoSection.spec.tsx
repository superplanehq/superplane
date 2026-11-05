import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_FACTORY_MULTI_REPO_TASKS, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";
import { RepositoryPicker } from "../onboarding/onboardingSteps";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;
const agentPageFeatures = [FEATURE_WORKSPACE_SKILLS, FEATURE_FACTORY_MULTI_REPO_TASKS];

describe("cross-repo context picker", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("explains how to expand the repository list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.vcs)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText(AGENT_RESOURCES_COPY.crossRepoMissingHint)).toBeInTheDocument();
    expect(screen.getByText(AGENT_RESOURCES_COPY.crossRepoMissingQuestion)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: AGENT_RESOURCES_COPY.crossRepoGrantGitHub })).toBeInTheDocument();
  }, 10000);

  it("says no match only when a search filters everything out", async () => {
    const user = userEvent.setup();
    render(<RepositoryPicker host="github" repos={["acme/api", "acme/web"]} selectedRepo={null} onSelect={() => {}} />);

    await user.type(screen.getByRole("textbox", { name: "Search repositories" }), "zzz-no-such-repo");
    expect(await screen.findByText("No matching repositories.", {}, { timeout: 8000 })).toBeInTheDocument();
  }, 15000);

  it("says unavailable when the catalog itself is empty", async () => {
    render(<RepositoryPicker host="github" repos={[]} selectedRepo={null} onSelect={() => {}} />);

    expect(await screen.findByText("No repositories available.", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.queryByText("No matching repositories.")).not.toBeInTheDocument();
  }, 15000);

  it("opens GitHub access without an error when the list is empty", async () => {
    const user = userEvent.setup();
    const opened: string[] = [];
    vi.spyOn(window, "open").mockImplementation(((url?: string | URL) => {
      opened.push(String(url));
      return null;
    }) as typeof window.open);
    try {
      render(
        <FactoriesHarness
          pathSuffix={agentPath}
          factoriesFixture={{
            ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.vcs),
            agentResourcesByFactoryId: {},
          }}
          orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
          experimentalFeatures={agentPageFeatures}
        />,
      );

      await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
      await user.click(
        await screen.findByRole("button", { name: AGENT_RESOURCES_COPY.crossRepoGrantGitHub }, { timeout: 8000 }),
      );

      await waitFor(() => expect(opened.length).toBeGreaterThan(0), { timeout: 8000 });
      expect(screen.queryByText(/could not open/)).not.toBeInTheDocument();
    } finally {
      vi.restoreAllMocks();
    }
  }, 15000);

  it("hides the workspace repository from the add list", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository)}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    expect(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 })).toBeInTheDocument();
    expect(within(picker).queryByRole("option", { name: /^acme\/api$/ })).not.toBeInTheDocument();
  }, 15000);

  it("lists the primary repository once, first, without a remove action", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {
            [PRIMARY_FACTORY_ID]: [
              { id: "ctx-primary", kind: "KIND_CONTEXT_REPO", enabled: true, repository: "ACME/api" },
              { id: "ctx-web", kind: "KIND_CONTEXT_REPO", enabled: true, repository: "acme/web" },
            ],
          },
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    const rows = await within(list).findAllByRole("listitem", {}, { timeout: 8000 });
    expect(rows.map((row) => row.textContent)).toEqual([
      expect.stringContaining("acme/api"),
      expect.stringContaining("acme/web"),
    ]);
    const primary = rows[0]!;
    expect(within(primary).queryByRole("button", { name: AGENT_RESOURCES_COPY.removeRepository })).toBeNull();
  }, 15000);

  it("asks for the branch and access before it adds a repository", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {},
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    await user.click(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 }));
    const settings = await within(picker).findByTestId("agent-cross-repo-settings", {}, { timeout: 8000 });
    expect(within(settings).getByTestId("agent-cross-repo-new-access")).toHaveTextContent(
      AGENT_RESOURCES_COPY.accessWrite,
    );
    await user.click(within(settings).getByTestId("agent-cross-repo-confirm"));

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    expect(
      await within(list).findByText(`main · ${AGENT_RESOURCES_COPY.accessWrite}`, {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 15000);

  it("keeps the access chosen before the repository", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {},
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    await user.click(await within(picker).findByTestId("agent-cross-repo-new-list-access", {}, { timeout: 8000 }));
    await user.click(await screen.findByRole("option", { name: AGENT_RESOURCES_COPY.accessRead }));
    await user.click(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 }));
    expect(await within(picker).findByTestId("agent-cross-repo-new-access", {}, { timeout: 8000 })).toHaveTextContent(
      AGENT_RESOURCES_COPY.accessRead,
    );
    await user.click(within(picker).getByTestId("agent-cross-repo-confirm"));

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    expect(
      await within(list).findByText(`main · ${AGENT_RESOURCES_COPY.accessRead}`, {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 15000);

  it("adds a repository on a branch that the branch list does not show", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {},
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    await user.click(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 }));
    await user.click(await within(picker).findByTestId("agent-cross-repo-new-branch", {}, { timeout: 8000 }));
    await user.type(await screen.findByPlaceholderText("Find a branch"), "release/v2");
    await user.click(await screen.findByRole("option", { name: 'Use "release/v2"' }));
    await user.click(within(picker).getByTestId("agent-cross-repo-confirm"));

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    expect(
      await within(list).findByText(`release/v2 · ${AGENT_RESOURCES_COPY.accessWrite}`, {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 15000);

  it("uses the default branch of a repository chosen after Back", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {},
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    await user.click(await within(picker).findByTestId("agent-cross-repo-new-list-access", {}, { timeout: 8000 }));
    await user.click(await screen.findByRole("option", { name: AGENT_RESOURCES_COPY.accessRead }));
    await user.click(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 }));
    await user.click(await within(picker).findByTestId("agent-cross-repo-new-branch", {}, { timeout: 8000 }));
    await user.type(await screen.findByPlaceholderText("Find a branch"), "release/v2");
    await user.click(await screen.findByRole("option", { name: 'Use "release/v2"' }));
    await user.click(within(picker).getByRole("button", { name: AGENT_RESOURCES_COPY.back }));
    await user.click(await within(picker).findByRole("option", { name: /^acme\/mobile$/ }, { timeout: 8000 }));
    await user.click(await within(picker).findByTestId("agent-cross-repo-confirm", {}, { timeout: 8000 }));

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    expect(
      await within(list).findByText(`main · ${AGENT_RESOURCES_COPY.accessRead}`, {}, { timeout: 8000 }),
    ).toBeInTheDocument();
  }, 15000);

  it("treats a repository without access as read only and edits it in place", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {
            [PRIMARY_FACTORY_ID]: [
              { id: "ctx-web", kind: "KIND_CONTEXT_REPO", enabled: true, repository: "acme/web", ref: "release" },
            ],
          },
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    const toggle = await screen.findByTestId("agent-cross-repo-toggle-ctx-web", {}, { timeout: 8000 });
    expect(toggle).toHaveTextContent(`release · ${AGENT_RESOURCES_COPY.accessRead}`);
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(await screen.findByTestId("agent-cross-repo-ctx-web-access", {}, { timeout: 8000 })).toHaveTextContent(
      AGENT_RESOURCES_COPY.accessRead,
    );
  }, 15000);

  it("adds read-only repositories without multi-repo tasks", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...factoriesFixtureWithSetupAnswers(SETUP_ANSWERS.repository),
          agentResourcesByFactoryId: {},
        }}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    await user.click(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 }));
    const settings = await within(picker).findByTestId("agent-cross-repo-settings", {}, { timeout: 8000 });
    expect(within(settings).queryByTestId("agent-cross-repo-new-access")).toBeNull();
    await user.click(within(settings).getByTestId("agent-cross-repo-confirm"));

    const list = await screen.findByTestId("agent-cross-repo-list", {}, { timeout: 8000 });
    expect(await within(list).findByText("main", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(within(list).queryByText(new RegExp(AGENT_RESOURCES_COPY.accessWrite))).toBeNull();
    expect(within(list).queryByTestId("agent-cross-repo-primary")).toBeNull();
  }, 15000);

  it("hides the workspace repository when only the letter case differs", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={factoriesFixtureWithSetupAnswers({
          ...SETUP_ANSWERS.repository,
          appRepository: "ACME/API",
        })}
        orgIntegrations={CONNECTED_SETUP_INTEGRATIONS}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    const picker = await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 });
    expect(await within(picker).findByRole("option", { name: /^acme\/web$/ }, { timeout: 8000 })).toBeInTheDocument();
    expect(within(picker).queryByRole("option", { name: /^acme\/api$/ })).not.toBeInTheDocument();
  }, 15000);

  it("asks to connect version control when nothing is connected", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={agentPageFeatures}
      />,
    );

    await user.click(await screen.findByTestId("agent-cross-repo-add", {}, { timeout: 8000 }));
    expect(await screen.findByTestId("agent-cross-repo-picker", {}, { timeout: 8000 })).toHaveTextContent(
      AGENT_RESOURCES_COPY.crossRepoConnectGitHub,
    );
  }, 10000);
});
