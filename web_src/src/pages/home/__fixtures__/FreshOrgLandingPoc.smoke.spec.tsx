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
  expect(within(panel).getByText("Default repository")).toBeInTheDocument();

  await user.click(continueButton);
}

async function completeAgentSettingsStep(user: ReturnType<typeof userEvent.setup>) {
  expect(screen.getByRole("heading", { name: "Agent settings" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Planning" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Implementation" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "PR review loop" })).toBeInTheDocument();
  expect(screen.getAllByText("Model").length).toBeGreaterThanOrEqual(3);
  expect(screen.getAllByText("Machine").length).toBeGreaterThanOrEqual(3);
  expect(screen.getByLabelText(/PR review loop max retries/i)).toHaveValue(5);

  const continueButton = screen.getByRole("button", { name: /^Continue$/i });
  expect(continueButton).toBeEnabled();
  await user.click(continueButton);
}

async function advanceFromTriggersToFinalStep(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: /^Continue$/i }));
  await selectDefaultRepository(user, "acme/web");
  await completeVersionControlStep(user);
  await completeCodingAgentStep(user);
  await completeAgentSettingsStep(user);
  expect(screen.getByText(/Step 5 of 5/i)).toBeInTheDocument();
}

describe("FreshOrgLanding story smoke", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });

    // Some vitest/jsdom runs ship a broken localStorage (--localstorage-file without a path).
    // AppPage reads sidebar prefs on mount when Done navigates to the live canvas.
    if (typeof window.localStorage?.getItem !== "function") {
      const store = new Map<string, string>();
      Object.defineProperty(window, "localStorage", {
        configurable: true,
        value: {
          getItem: (key: string) => store.get(key) ?? null,
          setItem: (key: string, value: string) => {
            store.set(key, String(value));
          },
          removeItem: (key: string) => {
            store.delete(key);
          },
          clear: () => store.clear(),
          key: (index: number) => [...store.keys()][index] ?? null,
          get length() {
            return store.size;
          },
        },
      });
    }
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

  it("shows editable agent components on settings and only blocks Done on preview", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);

    await user.click(screen.getByRole("button", { name: /Assign SuperPlane bot to your issue/i }));
    await user.click(screen.getByRole("button", { name: /GitHub Issues/i }));
    expect(screen.getByText("Not connected")).toBeInTheDocument();

    const continueButton = screen.getByRole("button", { name: /^Continue$/i });
    expect(continueButton).toBeEnabled();

    await advanceFromTriggersToFinalStep(user);

    expect(screen.getByRole("heading", { name: "Preview" })).toBeInTheDocument();
    expect(screen.getByText("Factory workflow")).toBeInTheDocument();
    expect(screen.getByText("GitHub Issues")).toBeInTheDocument();
    expect(screen.getByText(/Assign @superplane on the GitHub issue/i)).toBeInTheDocument();
    expect(screen.getAllByText("acme/web").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("Planning")).toBeInTheDocument();
    expect(screen.getByText("Implementation")).toBeInTheDocument();
    expect(screen.getByText("Review loop")).toBeInTheDocument();
    expect(screen.getByText(/You review and merge/i)).toBeInTheDocument();

    const doneButton = screen.getByRole("button", { name: /^Done$/i });
    expect(doneButton).toBeDisabled();

    const panel = screen.getByRole("complementary");
    expect(panel).toHaveAttribute("data-emphasize", "true");
    expect(within(panel).getAllByText("Not connected").length).toBeGreaterThan(0);
    expect(screen.getByRole("status")).toHaveTextContent(/Hey, make sure you connect all the required tools/i);

    // Connect GitHub, then Claude (two required integrations for this path).
    await connectRequiredIntegration(user, "GitHub", "github-connection");
    expect(doneButton).toBeDisabled();
    await connectRequiredIntegration(user, "Claude", "claude-connection");
    expect(within(panel).getAllByText("Connected", { exact: true }).length).toBeGreaterThan(0);
    expect(doneButton).toBeEnabled();
    expect(panel).not.toHaveAttribute("data-emphasize", "true");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();

    await user.click(doneButton);
    expect(await screen.findByText("Software Factory", {}, { timeout: 5000 })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Preview" })).not.toBeInTheDocument();
  });

  it("lets users edit component settings and agent steps", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await startFactorySetup(user);
    await user.click(screen.getByRole("button", { name: /Manual prompt/i }));
    await user.click(screen.getByRole("button", { name: /^Continue$/i }));
    await user.click(screen.getByRole("button", { name: /^GitHub$/i }));
    await completeVersionControlStep(user);
    await completeCodingAgentStep(user);

    expect(screen.getByRole("heading", { name: "Agent settings" })).toBeInTheDocument();
    expect(screen.getAllByText("Claude Sonnet 4.6").length).toBeGreaterThanOrEqual(3);
    expect(screen.getAllByText(/Large · AMD64 · 8 vCPU \/ 16 GB/i).length).toBeGreaterThanOrEqual(3);

    const maxRetries = screen.getByLabelText(/PR review loop max retries/i);
    await user.tripleClick(maxRetries);
    await user.keyboard("3");
    expect(maxRetries).toHaveValue(3);

    const planBody = screen.getByLabelText(/Planning step 1 body/i);
    await user.clear(planBody);
    await user.type(planBody, "Write a concise plan for the assigned issue.");
    expect(planBody).toHaveValue("Write a concise plan for the assigned issue.");

    const planningAddButtons = screen.getAllByRole("button", { name: /Add prompt/i });
    await user.click(planningAddButtons[0]!);
    expect(screen.getByLabelText(/Planning step 2 title/i)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Remove New prompt/i }));
    expect(screen.queryByLabelText(/Planning step 2 title/i)).not.toBeInTheDocument();

    expect(screen.getByRole("button", { name: /^Continue$/i })).toBeEnabled();
  });
});
