import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./__fixtures__/HomePageHarness";
import { emptyHomePageFixture } from "./__fixtures__/homePageResponses";

async function openFactorySetup() {
  const user = userEvent.setup();
  render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" />);
  await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 });
  await user.click(screen.getByRole("button", { name: /setup factory/i }));
  const panel = await screen.findByRole("region", { name: /software factory setup/i });
  return { user, panel };
}

describe("FreshOrgLanding", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders factory-first landing with blank and browse escape hatches", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" />);

    expect(await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 })).toBeInTheDocument();
    expect(screen.getByText(/set up a software factory to automate coding work/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /setup factory/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create a blank app/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /browse starter apps/i })).toBeInTheDocument();
    expect(screen.queryByText(/automation starters/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /browse starter apps/i }));
    expect(screen.getByText(/automation starters/i)).toBeInTheDocument();
  });

  it("opens factory setup with GitHub, Claude, and starting tasks", async () => {
    const { panel } = await openFactorySetup();

    expect(within(panel).getByRole("heading", { name: "Connect your GitHub and Claude" })).toBeInTheDocument();
    expect(
      within(panel).getByText(/automate coding work with agents, from issue to pull request/i),
    ).toBeInTheDocument();
    expect(within(panel).getByText("GitHub")).toBeInTheDocument();
    expect(within(panel).getByText("Claude")).toBeInTheDocument();
    expect(within(panel).queryByText("Choose repository")).not.toBeInTheDocument();
    expect(within(panel).queryByText("Anthropic API key secret")).not.toBeInTheDocument();
    expect(within(panel).queryByText("GitHub token secret")).not.toBeInTheDocument();
    expect(within(panel).getByText("Choose starting task")).toBeInTheDocument();

    const taskButtons = within(panel).getAllByRole("button", {
      name: /^(improve agents\.md|improve test coverage|find and fix a bug|set up or improve ci)$/i,
    });
    expect(taskButtons.map((button) => button.textContent)).toEqual([
      "Improve AGENTS.md",
      "Improve test coverage",
      "Find and fix a bug",
      "Set up or improve CI",
    ]);
    expect(within(panel).getByRole("button", { name: /^improve agents\.md$/i })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(within(panel).getByRole("button", { name: /^Run$/i })).toBeDisabled();
    expect(within(panel).getByRole("button", { name: /^Cancel$/i })).toBeInTheDocument();
  });

  it("keeps a starting task selected when the same button is clicked again", async () => {
    const { user, panel } = await openFactorySetup();

    const agentsPrompt =
      "Improve AGENTS.md for this repo (create it if missing).\n\n- Review the repository to understand layout, build/test, and conventions\n- Cover build/test, key packages, and repo-specific guidance\n- Keep useful guidance; remove generic or outdated advice\n- One focused pass only — do not rewrite unrelated docs";
    const writeTestPrompt =
      "Add one focused unit test.\n\n- Review the repository to find core logic and existing test patterns\n- Target a single untested helper or pure function in core logic\n- Use existing test tooling and patterns — do not add a new framework\n- Keep the change as small as possible\n- If there is no test tooling: add only the minimal setup for the primary language, just enough to run that one test";

    const promptField = within(panel).getByLabelText(/^Prompt$/i);
    expect(promptField).toHaveAttribute("readonly");
    expect(promptField).toHaveValue(agentsPrompt);

    await user.click(within(panel).getByRole("button", { name: /^improve test coverage$/i }));
    expect(promptField).toHaveValue(writeTestPrompt);
    await user.click(within(panel).getByRole("button", { name: /^improve test coverage$/i }));
    expect(within(panel).getByRole("button", { name: /^improve test coverage$/i })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(within(panel).getByLabelText(/^Prompt$/i)).toHaveValue(writeTestPrompt);
  });

  it("opens GitHub capability setup in a new tab from Connect", async () => {
    const { user, panel } = await openFactorySetup();
    const githubRow = within(panel).getByText("GitHub").closest("div");
    expect(githubRow).toBeTruthy();

    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    await user.click(within(githubRow!).getByRole("button", { name: /^Connect$/i }));
    expect(openSpy).toHaveBeenCalledWith(
      expect.stringMatching(/\/settings\/integrations\/github\/setup/),
      "_blank",
      "noopener,noreferrer",
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(panel).toBeInTheDocument();
  });
});
