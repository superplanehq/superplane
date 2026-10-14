import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  CONNECTED_SETUP_INTEGRATIONS,
  SETUP_ANSWERS,
  factoriesFixtureWithSetupAnswers,
} from "../../__fixtures__/setupStoryFixtures";
import { RepositoryPicker } from "../onboarding/onboardingSteps";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/agent`;
const agentPageFeatures = [FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];

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
