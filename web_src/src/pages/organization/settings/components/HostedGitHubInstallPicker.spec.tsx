import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { HostedGitHubInstallPicker } from "./HostedGitHubInstallPicker";

describe("HostedGitHubInstallPicker", () => {
  it("shows repositories grouped by account and the install link", () => {
    render(
      <HostedGitHubInstallPicker
        state="csrf"
        appSlug="superplane"
        installations={[
          { id: "11", accountLogin: "acme", repositories: [{ id: "101", name: "acme/api" }] },
          { id: "22", accountLogin: "octo", repositories: [{ id: "202", name: "octo/web" }] },
        ]}
      />,
    );

    expect(screen.getByRole("heading", { name: "Select a GitHub repository" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use acme/api" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use octo/web" })).toBeInTheDocument();
    expect(screen.getByText("Do not see your GitHub account or organization?")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Install the GitHub App there." })).toHaveAttribute(
      "href",
      "https://github.com/apps/superplane/installations/new?state=csrf",
    );
  });
});
