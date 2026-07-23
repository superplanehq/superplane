import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./__fixtures__/HomePageHarness";
import { emptyHomePageFixture } from "./__fixtures__/homePageResponses";

describe("FreshOrgLanding", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("renders factory-first landing with blank and browse escape hatches", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" prototypeNewApp />);

    expect(await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 })).toBeInTheDocument();
    expect(screen.getByText(/set up a software factory to automate coding work/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /setup factory/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create a blank app/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /browse starter apps/i })).toBeInTheDocument();
    expect(screen.queryByText(/automation starters/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /browse starter apps/i }));
    expect(screen.getByText(/automation starters \(not software factory setup\)/i)).toBeInTheDocument();
  });

  it("opens inline Software Factory setup with connect, repo, starting task, and gated run", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/new" prototypeNewApp />);

    await screen.findByRole("heading", { name: "Create a new app" }, { timeout: 5000 });
    await user.click(screen.getByRole("button", { name: /setup factory/i }));

    const panel = await screen.findByRole("region", { name: /software factory setup/i });
    expect(within(panel).getByRole("heading", { name: "Connect your GitHub and Claude" })).toBeInTheDocument();
    expect(
      within(panel).getByText(
        /this will create software factory that automates your delivery from trigger to pull request/i,
      ),
    ).toBeInTheDocument();
    expect(within(panel).getByText("GitHub")).toBeInTheDocument();
    expect(within(panel).getByText("Claude")).toBeInTheDocument();
    expect(within(panel).queryByText("Choose repository")).not.toBeInTheDocument();
    expect(within(panel).getByText("Choose starting task")).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^write test$/i })).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^fix bug$/i })).toBeInTheDocument();
    expect(within(panel).getByRole("button", { name: /^improve agents\.md$/i })).toBeInTheDocument();
    expect(within(panel).queryByLabelText(/^Prompt$/i)).not.toBeInTheDocument();
    expect(
      within(panel).queryByRole("button", { name: /take me to the app without connecting/i }),
    ).not.toBeInTheDocument();

    const runButton = within(panel).getByRole("button", { name: /^Run$/i });
    expect(runButton).toBeDisabled();
    expect(within(panel).getByRole("button", { name: /^Cancel$/i })).toBeInTheDocument();

    await user.click(within(panel).getByRole("button", { name: /^write test$/i }));
    const promptField = within(panel).getByLabelText(/^Prompt$/i);
    expect(promptField).toHaveAttribute("readonly");
    expect(promptField).toHaveValue(
      "Scan the codebase to understand its main business logic. Then identify ONE untested function related to this business logic and write a single focused, useful unit test for it. Cover the main execution path and follow existing test patterns. Ensure the test passes.",
    );

    const githubRow = within(panel).getByText("GitHub").closest("div");
    expect(githubRow).toBeTruthy();
    await user.click(within(githubRow!).getByRole("button", { name: /^Connect$/i }));

    const githubDialog = await screen.findByRole("dialog");
    expect(within(githubDialog).getByPlaceholderText(/my-app-integration/i)).toBeInTheDocument();
    expect(within(githubDialog).getByText("Organization")).toBeInTheDocument();
    expect(within(githubDialog).queryByText("API Key")).not.toBeInTheDocument();
    const githubNameInput = within(githubDialog).getByPlaceholderText(/my-app-integration/i);
    await user.clear(githubNameInput);
    await user.type(githubNameInput, "acme-github");
    await user.click(within(githubDialog).getByRole("button", { name: /^Connect$/i }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    expect(await within(panel).findByText("Connected")).toBeInTheDocument();
    expect(await within(panel).findByText("Choose repository")).toBeInTheDocument();
    expect(runButton).toBeDisabled();

    const claudeRow = within(panel).getByText("Claude").closest("div");
    expect(claudeRow).toBeTruthy();
    await user.click(within(claudeRow!).getByRole("button", { name: /^Connect$/i }));

    const claudeDialog = await screen.findByRole("dialog");
    expect(within(claudeDialog).getByText("API Key")).toBeInTheDocument();
    expect(within(claudeDialog).getByText("Admin API Key")).toBeInTheDocument();
    expect(within(claudeDialog).queryByText("Organization")).not.toBeInTheDocument();
    await user.click(within(claudeDialog).getByRole("button", { name: /^Cancel$/i }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());

    await user.click(within(panel).getByRole("button", { name: /^Cancel$/i }));
    expect(screen.queryByRole("region", { name: /software factory setup/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /setup factory/i })).toBeInTheDocument();
  });
});
