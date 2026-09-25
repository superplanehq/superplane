import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";

import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPullRequest } from "@/api-client";

import { pullRequestMentionNote } from "./lib/pullRequestMentionNote";
import { WorkOrderStatusNote } from "./WorkOrderStatusNote";

const OPEN_PULL_REQUEST: FactoriesFactoryPullRequest = {
  id: "pr-1",
  workOrderId: "wo-1",
  repository: "acme/payments",
  url: "https://github.com/acme/payments/pull/42",
  number: "42",
  state: "STATE_OPEN",
};

function handler(overrides: Partial<FactoriesFactoryPrFeedbackHandler> = {}): FactoriesFactoryPrFeedbackHandler {
  return {
    id: "handler-1",
    factoryId: "factory-1",
    source: "SOURCE_PULL_REQUEST_DISCUSSION",
    healthy: true,
    settings: { subject: { repository: "acme/payments" } },
    ...overrides,
  };
}

/**
 * Renders the note exactly as the task page assembles it: the mention note
 * built from the tracked pull request and the factory's PR feedback
 * handlers, rendered through the same `WorkOrderStatusNote` card every
 * other status note uses.
 */
function renderMentionNote(handlers: FactoriesFactoryPrFeedbackHandler[] | undefined) {
  const note = pullRequestMentionNote([OPEN_PULL_REQUEST], "wo-1", handlers);
  if (!note) {
    return { note, container: render(<></>).container };
  }
  const { container } = render(
    <MemoryRouter>
      <WorkOrderStatusNote
        note={note}
        organizationId="org-1"
        factoryKey="acme"
        canClose={false}
        canManage={false}
        isBusy={false}
        statusActions={[]}
        onClose={() => {}}
        onStatusChange={async () => {}}
      />
    </MemoryRouter>,
  );
  return { note, container };
}

describe("task page pull request mention note", () => {
  it("shows the note text, mention, and pull-request link for a matching healthy handler", () => {
    renderMentionNote([handler()]);

    expect(screen.getByRole("heading", { name: "Ask for changes in the pull request" })).toBeInTheDocument();
    expect(
      screen.getByText("Mention @superplaneagent in a pull request comment or review to request changes."),
    ).toBeInTheDocument();
    const link = screen.getByRole("link", { name: "Open pull request" });
    expect(link).toHaveAttribute("href", "https://github.com/acme/payments/pull/42");
  });

  it("is absent without a matching handler", () => {
    const { note } = renderMentionNote([]);

    expect(note).toBeUndefined();
    expect(screen.queryByRole("heading", { name: "Ask for changes in the pull request" })).not.toBeInTheDocument();
  });

  it("is absent when the only handler is unhealthy", () => {
    const { note } = renderMentionNote([handler({ healthy: false })]);

    expect(note).toBeUndefined();
    expect(screen.queryByRole("heading", { name: "Ask for changes in the pull request" })).not.toBeInTheDocument();
  });
});
