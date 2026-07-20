import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./HomePageHarness";
import { emptyHomePageFixture } from "./homePageResponses";

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
    expect(screen.getByText("Agent plans and codes")).toBeInTheDocument();
    expect(screen.getByText("Opens a pull request")).toBeInTheDocument();
    expect(screen.getByText("Keeps checks passing")).toBeInTheDocument();
    expect(screen.getByText("Waits for your review")).toBeInTheDocument();
    expect(screen.getByText("Addresses review comments")).toBeInTheDocument();
    expect(screen.getByText("Gets to a mergeable state")).toBeInTheDocument();
    expect(screen.getByLabelText("Setup steps")).toBeInTheDocument();
    expect(screen.getByText("Trigger")).toBeInTheDocument();
    expect(screen.getByText("Version control")).toBeInTheDocument();
    expect(screen.getByText("Coding agent")).toBeInTheDocument();
    expect(screen.getByText("Preview and tweak")).toBeInTheDocument();
    expect(screen.getByText(/Manual prompt, issue from a tracker, and\/or PR tag/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start setup/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create a blank app/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /browse other starter apps/i })).toBeInTheDocument();
    expect(screen.queryByText(/automation starters/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /browse other starter apps/i }));
    expect(screen.getByText(/automation starters \(not software factory setup\)/i)).toBeInTheDocument();
  });

  it("shows accurate setup stub steps after Start setup", async () => {
    const user = userEvent.setup();
    render(<HomePageHarness fixture={emptyHomePageFixture} pathSuffix="apps/welcome" />);

    await screen.findByRole("heading", { name: "Ship PRs to a mergeable state" }, { timeout: 5000 });
    await user.click(screen.getByRole("button", { name: /start setup/i }));

    expect(screen.getByRole("heading", { name: "Software Factory setup" })).toBeInTheDocument();
    expect(screen.getByText("GitHub Issues")).toBeInTheDocument();
    expect(screen.getByText("GitLab Issues")).toBeInTheDocument();
    expect(screen.getByText("Linear")).toBeInTheDocument();
    expect(screen.getByText("Jira")).toBeInTheDocument();
    expect(screen.getByText(/Connect GitHub or GitLab for checkout and PR\/MR/i)).toBeInTheDocument();
    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("Codex")).toBeInTheDocument();
    expect(screen.getByText("Open Code")).toBeInTheDocument();
    expect(screen.getByText(/Adjust prompts, SSH commands/i)).toBeInTheDocument();
  });
});
