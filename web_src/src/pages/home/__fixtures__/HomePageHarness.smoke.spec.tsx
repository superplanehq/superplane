import { render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { client } from "@/api-client/client.gen";

import { HomePageHarness } from "./HomePageHarness";
import { emptyHomePageFixture } from "./homePageResponses";

describe("HomePageHarness story smoke", () => {
  beforeAll(() => {
    // jsdom/undici cannot construct `new Request("/relative")` without a base.
    // Storybook's real browser location makes this unnecessary at preview time.
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("renders populated current homepage without redirecting to apps/new", async () => {
    render(<HomePageHarness />);

    expect(await screen.findByRole("heading", { name: "Apps" }, { timeout: 5000 })).toBeInTheDocument();
    expect(await screen.findByText("Software Factory")).toBeInTheDocument();
    expect(screen.getByText("Automation")).toBeInTheDocument();
    expect(screen.getByText("Releases")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /create new app/i })).not.toBeInTheDocument();
  });

  it("redirects a fresh org to the create / onboarding zero state", async () => {
    render(<HomePageHarness fixture={emptyHomePageFixture} />);

    expect(await screen.findByRole("heading", { name: "Create New App" }, { timeout: 5000 })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start from scratch/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Apps" })).not.toBeInTheDocument();
  });
});
