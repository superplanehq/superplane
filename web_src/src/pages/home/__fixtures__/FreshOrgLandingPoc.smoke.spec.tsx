import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./HomePageHarness";
import { emptyHomePageFixture } from "./homePageResponses";

async function startFactorySetup(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByRole("heading", { name: "Ship PRs to a mergeable state" }, { timeout: 5000 });
  await user.click(screen.getByRole("button", { name: /start setup/i }));
}

async function connectRequiredIntegration(
  user: ReturnType<typeof userEvent.setup>,
  integrationLabel: string,
  connectionName: string,
) {
  const panel = screen.getByRole("complementary");
  const row = within(panel)
    .getAllByRole("listitem")
    .find((item) => within(item).queryByText(integrationLabel, { exact: true }));
  expect(row).toBeTruthy();
  await user.click(within(row!).getByRole("button", { name: /^Connect$/i }));
  const dialog = await screen.findByRole("dialog");
  const nameInput = within(dialog).getByPlaceholderText(/my-app-integration/i);
  await user.clear(nameInput);
  await user.type(nameInput, connectionName);
  await user.click(within(dialog).getByRole("button", { name: /^Connect$/i }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
}

async function selectDefaultRepository(user: ReturnType<typeof userEvent.setup>, optionLabel: string) {
  const panel = screen.getByRole("complementary");
  await user.click(within(panel).getByPlaceholderText("Select repository"));
  await user.click(await screen.findByRole("option", { name: optionLabel }));
}

async function completeVersionControlStep(user: ReturnType<typeof userEvent.setup>) {
  expect(screen.getByRole("heading", { name: "Version control" })).toBeInTheDocument();
  const continueButton = screen.getByRole("button", { name: /^Continue$/i });
  expect(continueButton).toBeEnabled();
  await user.click(continueButton);
}

async function completeCodingAgentStep(user: ReturnType<typeof userEvent.setup>) {
  expect(screen.getByRole("heading", { name: "Coding agent" })).toBeInTheDocument();
  const continueButton = screen.getByRole("button", { name: /^Continue$/i });
  expect(continueButton).toBeDisabled();

  await user.click(screen.getByRole("button", { name: /Claude Code/i }));
  expect(continueButton).toBeEnabled();

  const panel = screen.getByRole("complementary");
  expect(within(panel).getByText("Claude", { exact: true })).toBeInTheDocument();
  expect(within(panel).queryByLabelText(/API key/i)).not.toBeInTheDocument();
  expect(within(panel).getByPlaceholderText("Select repository")).toBeInTheDocument();

  await user.click(continueButton);
}

async function advanceFromTriggersToFinalStep(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: /^Continue$/i }));
  await completeVersionControlStep(user);
  await completeCodingAgentStep(user);
  expect(screen.getByText(/Step 4 of 4/i)).toBeInTheDocument();
}

describe("FreshOrgLanding story smoke", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("renders densified split landing with outcome timeline and setup steps", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    expect(
      await screen.findByRole("heading", { name: "Ship PRs to a mergeable state" }, { timeout: 5000 }),
    ).toBeInTheDocument();
    expect(screen.getByText(/orchestrates cloud agents/i)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "What you get" })).toBeInTheDocument();
    expect(screen.getByText("Work is triggered")).toBeInTheDocument();
    expect(screen.getByLabelText("Setup steps")).toBeInTheDocument();
    expect(
      screen.getByText(/Select the triggers you want to use to kick off Software Factory work/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/Where the factory checks out code and opens pull or merge requests/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start setup/i })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /browse other starter apps/i }));
    expect(screen.getByText(/automation starters \(not software factory setup\)/i)).toBeInTheDocument();
  });

  it("opens Trigger setup with Required integrations panel driven by left selections", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);

    expect(screen.getByRole("heading", { name: "Software Factory setup" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Required integrations" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Manual prompt/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Mention SuperPlane in your pull or merge request/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText("GitHub Issues")).not.toBeInTheDocument();
    expect(screen.getByText(/Select triggers on the left/i)).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /Manual prompt/i }));
    expect(continueButton).toBeEnabled();
  });

  it("lists a single GitHub integration when issue tracker and PR host share GitHub", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);

    await user.click(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i }));
    expect(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByText("Issue tracker")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /GitHub Issues/i }));
    await user.click(screen.getByRole("button", { name: /Mention SuperPlane in your pull or merge request/i }));

    expect(screen.getByText("Pull request or merge request")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /GitHub pull request/i })).toHaveAttribute("aria-pressed", "true");

    const panel = screen.getByRole("complementary");
    expect(within(panel).getByText("GitHub", { exact: true })).toBeInTheDocument();
    expect(within(panel).queryByText("GitHub Issues")).not.toBeInTheDocument();
    expect(within(panel).getAllByRole("listitem")).toHaveLength(1);
    expect(within(panel).getByText("Not connected")).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeEnabled();
  });

  it("locks version control to GitHub from triggers and picks a simulated repo from the right panel", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);

    await user.click(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i }));
    await user.click(screen.getByRole("button", { name: /GitHub Issues/i }));
    await user.click(screen.getByRole("button", { name: /Mention SuperPlane in your pull or merge request/i }));
    await user.click(screen.getByRole("button", { name: /^Continue$/i }));

    expect(screen.getByRole("heading", { name: "Version control" })).toBeInTheDocument();
    expect(screen.getByText(/Using GitHub for version control/i)).toBeInTheDocument();
    expect(screen.getByText(/Based on the triggers you selected/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^GitLab$/i })).not.toBeInTheDocument();

    const panel = screen.getByRole("complementary");
    expect(within(panel).getByText("GitHub", { exact: true })).toBeInTheDocument();
    expect(within(panel).getByText("Default repository")).toBeInTheDocument();
    expect(within(panel).getByPlaceholderText("Select repository")).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeEnabled();

    await selectDefaultRepository(user, "acme/api");
    expect(within(panel).getByText("acme/api")).toBeInTheDocument();
    expect(continueButton).toBeEnabled();
  });

  it("keeps the repo picker on coding agent and routes harnesses and Open Code providers correctly", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);
    await user.click(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i }));
    await user.click(screen.getByRole("button", { name: /GitHub Issues/i }));
    await user.click(screen.getByRole("button", { name: /^Continue$/i }));
    await selectDefaultRepository(user, "acme/web");
    await completeVersionControlStep(user);

    expect(screen.getByRole("heading", { name: "Coding agent" })).toBeInTheDocument();
    const panel = screen.getByRole("complementary");
    expect(within(panel).getByText("acme/web")).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /^Cursor$/i }));
    expect(within(panel).getByText("Cursor", { exact: true })).toBeInTheDocument();
    expect(within(panel).queryByLabelText(/API key/i)).not.toBeInTheDocument();
    expect(continueButton).toBeEnabled();

    await user.click(screen.getByRole("button", { name: /Claude Code/i }));
    expect(within(panel).getByText("Claude", { exact: true })).toBeInTheDocument();
    expect(continueButton).toBeEnabled();

    await user.click(screen.getByRole("button", { name: /Open Code/i }));
    expect(screen.getByText("Free / local")).toBeInTheDocument();
    expect(screen.getByText("Model provider")).toBeInTheDocument();
    expect(continueButton).toBeDisabled();
    expect(within(panel).queryByText("Claude", { exact: true })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Ollama/i }));
    expect(within(panel).queryByLabelText(/API key/i)).not.toBeInTheDocument();
    expect(continueButton).toBeEnabled();

    await user.click(screen.getByRole("button", { name: /OpenCode Zen/i }));
    expect(within(panel).getByLabelText(/OpenCode Zen API key/i)).toBeInTheDocument();
    expect(continueButton).toBeEnabled();

    await user.click(screen.getByRole("button", { name: /Anthropic/i }));
    expect(within(panel).getByText("Claude", { exact: true })).toBeInTheDocument();
    expect(within(panel).queryByLabelText(/API key/i)).not.toBeInTheDocument();
    expect(continueButton).toBeEnabled();

    await user.click(screen.getByRole("button", { name: /Google Gemini/i }));
    expect(within(panel).queryByText("Claude", { exact: true })).not.toBeInTheDocument();
    expect(within(panel).getByLabelText(/Google Gemini API key/i)).toBeInTheDocument();
    expect(continueButton).toBeEnabled();
  });

  it("allows continuing without connecting, and only blocks Done on the final step", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);

    await user.click(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i }));
    await user.click(screen.getByRole("button", { name: /GitHub Issues/i }));
    expect(screen.getByText("Not connected")).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeEnabled();

    await advanceFromTriggersToFinalStep(user);

    const doneButton = screen.getByRole("button", { name: /^Done$/i });
    expect(doneButton).toBeDisabled();

    const panel = screen.getByRole("complementary");
    expect(panel).toHaveAttribute("data-emphasize", "true");
    expect(within(panel).getAllByText("Not connected").length).toBeGreaterThan(0);

    // Connect GitHub, then Claude (two required integrations for this path).
    await connectRequiredIntegration(user, "GitHub", "github-connection");
    expect(doneButton).toBeDisabled();
    await connectRequiredIntegration(user, "Claude", "claude-connection");
    expect(within(panel).getAllByText("Connected", { exact: true }).length).toBeGreaterThan(0);
    expect(doneButton).toBeEnabled();
    expect(panel).not.toHaveAttribute("data-emphasize", "true");
  });
});
