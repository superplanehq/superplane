import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HostedGitHubInstallPicker } from "./HostedGitHubInstallPicker";

describe("HostedGitHubInstallPicker", () => {
  it("shows account buttons and the install link", () => {
    render(
      <HostedGitHubInstallPicker
        state="csrf"
        appSlug="superplane"
        installations={[
          { id: "11", accountLogin: "acme" },
          { id: "22", accountLogin: "octo" },
        ]}
      />,
    );

    expect(screen.getByRole("heading", { name: "Select a GitHub account" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use acme" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use octo" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Install on a different account" })).toHaveAttribute(
      "href",
      "https://github.com/apps/superplane/installations/new?state=csrf",
    );
  });
});
