import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { IntakeWebhookFailureNotice } from "./IntakeWebhookFailureNotice";

describe("IntakeWebhookFailureNotice", () => {
  it("shows the hook limit and a link the user can open", () => {
    render(
      <IntakeWebhookFailureNotice
        message="GitHub did not create the webhook. This repository already has 20 webhooks for issues. Remove unused webhooks in the repository settings, then try again."
        repository="mytmlt/sentry-intake-test-project"
        onRetry={vi.fn()}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("already has 20 webhooks");
    expect(screen.getByRole("link", { name: "Open repository webhooks" })).toHaveAttribute(
      "href",
      "https://github.com/mytmlt/sentry-intake-test-project/settings/hooks",
    );
  });

  it("opens repository webhooks on the GitHub Enterprise host", () => {
    render(
      <IntakeWebhookFailureNotice
        message="GitHub did not create the webhook."
        repository="acme/app"
        host="acme.ghe.com"
        onRetry={vi.fn()}
      />,
    );

    expect(screen.getByRole("link", { name: "Open repository webhooks" })).toHaveAttribute(
      "href",
      "https://acme.ghe.com/acme/app/settings/hooks",
    );
  });

  it("tries again when the user asks", async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();
    render(
      <IntakeWebhookFailureNotice
        message="GitHub did not create the webhook."
        repository="acme/app"
        onRetry={onRetry}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("does not link a repository name that is not owner and name", () => {
    render(
      <IntakeWebhookFailureNotice
        message="GitHub did not create the webhook."
        repository="not a repo"
        onRetry={vi.fn()}
      />,
    );

    expect(screen.queryByRole("link", { name: "Open repository webhooks" })).not.toBeInTheDocument();
  });
});
