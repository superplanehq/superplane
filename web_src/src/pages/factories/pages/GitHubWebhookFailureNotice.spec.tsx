import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { GitHubWebhookFailureNotice } from "./GitHubWebhookFailureNotice";

const failedWebhook = {
  id: "hook-1",
  repository: "acme/app",
  eventTypes: ["pull_request", "check_run"],
  affectedAutomations: [
    { id: "one", name: "Review" },
    { id: "two", name: "Verify" },
  ],
  state: "failed",
  message:
    "GitHub reached the webhook limit for pull requests. Remove unused webhooks in the repository settings, then try again.",
};

describe("GitHubWebhookFailureNotice", () => {
  it("identifies the shared registration and opens repository settings on the Enterprise host", () => {
    render(
      <GitHubWebhookFailureNotice
        webhook={{ ...failedWebhook, host: "acme.ghe.com" }}
        onRetry={vi.fn()}
        onRefresh={vi.fn()}
      />,
    );

    const warning = screen.getByRole("alert");
    expect(warning).toHaveTextContent("acme/app");
    expect(warning).toHaveTextContent("pull requests");
    expect(warning).toHaveTextContent("Review");
    expect(warning).toHaveTextContent("Verify");
    expect(screen.getByRole("link", { name: "Open repository webhooks" })).toHaveAttribute(
      "href",
      "https://acme.ghe.com/acme/app/settings/hooks",
    );
  });

  it("keeps the warning visible during retry and restores the action after failure", async () => {
    const retry = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(
      <GitHubWebhookFailureNotice webhook={failedWebhook} onRetry={retry} onRefresh={vi.fn()} />,
    );
    await user.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();

    rerender(
      <GitHubWebhookFailureNotice
        webhook={{ ...failedWebhook, state: "pending" }}
        retrying
        onRetry={retry}
        onRefresh={vi.fn()}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Trying again");
    expect(screen.queryByRole("button", { name: "Try again" })).not.toBeInTheDocument();

    rerender(<GitHubWebhookFailureNotice webhook={failedWebhook} onRetry={retry} onRefresh={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
  });

  it("offers refresh after the polling window and clears after recovery", async () => {
    const refresh = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(
      <GitHubWebhookFailureNotice webhook={failedWebhook} retryTimedOut onRetry={vi.fn()} onRefresh={refresh} />,
    );
    await user.click(screen.getByRole("button", { name: "Refresh status" }));
    expect(refresh).toHaveBeenCalledOnce();

    rerender(
      <GitHubWebhookFailureNotice
        webhook={{ ...failedWebhook, state: "ready" }}
        onRetry={vi.fn()}
        onRefresh={refresh}
      />,
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
