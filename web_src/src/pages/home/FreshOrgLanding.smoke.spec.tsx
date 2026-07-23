import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./__fixtures__/HomePageHarness";
import { emptyHomePageFixture } from "./__fixtures__/homePageResponses";

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
    expect(screen.getByText(/automation starters \(not software factory setup\)/i)).toBeInTheDocument();
  });

  it("opens inline Software Factory setup with connect, params, optional starting task, and always-available run", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" />);

    await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 });
    await user.click(screen.getByRole("button", { name: /setup factory/i }));

    const panel = await screen.findByRole("region", { name: /software factory setup/i });
    expect(within(panel).getByRole("heading", { name: "Connect your GitHub and Claude" })).toBeInTheDocument();
    expect(
      within(panel).getByText(/automate coding work with agents, from trigger to pull request/i),
    ).toBeInTheDocument();
    expect(within(panel).getByText("GitHub")).toBeInTheDocument();
    expect(within(panel).getByText("Claude")).toBeInTheDocument();
    expect(within(panel).queryByText("Choose repository")).not.toBeInTheDocument();
    expect(within(panel).queryByText(/anthropic api key/i)).not.toBeInTheDocument();
    expect(within(panel).getByText("Choose starting task")).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^write test$/i })).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^fix bug$/i })).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^improve agents\.md$/i })).toBeInTheDocument();
    expect(within(panel).queryByLabelText(/^Prompt$/i)).not.toBeInTheDocument();

    const runButton = within(panel).getByRole("button", { name: /^Run$/i });
    expect(runButton).toBeEnabled();
    expect(within(panel).getByRole("button", { name: /^Cancel$/i })).toBeInTheDocument();

    await user.click(within(panel).getByRole("button", { name: /^write test$/i }));
    const promptField = within(panel).getByLabelText(/^Prompt$/i);
    expect(promptField).toHaveAttribute("readonly");
    expect(promptField).toHaveValue(
      "Scan the codebase to understand its main business logic. Then identify ONE untested function related to this business logic and write a single focused, useful unit test for it. Cover the main execution path and follow existing test patterns. Ensure the test passes.",
    );
    expect(runButton).toBeEnabled();

    const githubRow = within(panel).getByText("GitHub").closest("div");
    expect(githubRow).toBeTruthy();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    await user.click(within(githubRow!).getByRole("button", { name: /^Connect$/i }));
    // GitHub is capability-based: Connect opens the setup wizard in a new tab.
    expect(openSpy).toHaveBeenCalledWith(
      expect.stringMatching(/\/settings\/integrations\/github\/setup/),
      "_blank",
      "noopener,noreferrer",
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(panel).toBeInTheDocument();
  });

  it("opens the legacy Claude connect dialog from factory setup", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" />);

    await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 });
    await user.click(screen.getByRole("button", { name: /setup factory/i }));

    const panel = await screen.findByRole("region", { name: /software factory setup/i });
    const claudeRow = within(panel).getByText("Claude").closest("div");
    expect(claudeRow).toBeTruthy();
    await user.click(within(claudeRow!).getByRole("button", { name: /^Connect$/i }));
    const claudeDialog = await screen.findByRole("dialog");
    expect(within(claudeDialog).getByText("API Key")).toBeInTheDocument();
    await user.click(within(claudeDialog).getByRole("button", { name: /^Cancel$/i }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });
});
