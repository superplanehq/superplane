import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { GitHubAccountConnection } from "./GitHubAccountConnection";

const showErrorToast = vi.fn();
const showInfoToast = vi.fn();
const showSuccessToast = vi.fn();

vi.mock("@/lib/toast", () => ({
  showErrorToast: (message: string) => showErrorToast(message),
  showInfoToast: (message: string) => showInfoToast(message),
  showSuccessToast: (message: string) => showSuccessToast(message),
}));

describe("GitHubAccountConnection", () => {
  beforeEach(() => {
    showErrorToast.mockReset();
    showInfoToast.mockReset();
    showSuccessToast.mockReset();
  });

  it("starts account linking and returns to the current settings page", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/settings/profile"]}>
        <GitHubAccountConnection providers={[]} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("connect-github-profile")).toHaveAttribute(
      "href",
      "/account/providers/github/connect?redirect=%2Forg-1%2Fsettings%2Fprofile",
    );
  });

  it("shows the connected GitHub username instead of a connect action", () => {
    render(
      <MemoryRouter>
        <GitHubAccountConnection
          providers={[
            {
              provider: "github",
              username: "shiroyasha",
              display_name: "Igor Šarčević",
              email: "igisar@gmail.com",
              avatar_url: "https://avatars.example/shiroyasha",
            },
          ]}
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("@shiroyasha")).toBeInTheDocument();
    expect(screen.getByText("Connected")).toBeInTheDocument();
    expect(screen.queryByTestId("connect-github-profile")).not.toBeInTheDocument();
  });

  it("reports a successful callback and removes its result from the URL", async () => {
    render(
      <MemoryRouter initialEntries={["/org-1/settings/profile?provider=github&provider_link=success"]}>
        <GitHubAccountConnection providers={[]} />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("GitHub profile connected.");
    });
    expect(showErrorToast).not.toHaveBeenCalled();
  });
});
