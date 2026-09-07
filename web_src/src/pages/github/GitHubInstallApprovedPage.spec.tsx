import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";

import {
  GITHUB_INSTALL_APPROVED_ACTION,
  GITHUB_INSTALL_APPROVED_BODY,
  GITHUB_INSTALL_APPROVED_TITLE,
} from "@/lib/githubInstallRequestCopy";

import { GitHubInstallApprovedPage } from "./GitHubInstallApprovedPage";

describe("GitHubInstallApprovedPage", () => {
  it("explains that the GitHub App request is approved", () => {
    render(
      <MemoryRouter>
        <GitHubInstallApprovedPage />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("github-install-approved")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: GITHUB_INSTALL_APPROVED_TITLE })).toBeInTheDocument();
    expect(screen.getByText(GITHUB_INSTALL_APPROVED_BODY)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: GITHUB_INSTALL_APPROVED_ACTION })).toHaveAttribute("href", "/");
  });
});
