import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrder, FactoriesWorkOrderArtifact, FactoriesWorkOrderEvent } from "@/api-client";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({ data: [] }),
}));

import { WorkOrderActivityTimeline } from "./WorkOrderActivityTimeline";

const ORDER: FactoriesWorkOrder = {
  id: "wo-1",
  title: "Refund retry",
  executions: [],
};

const ARTIFACT_ADDED_EVENT: FactoriesWorkOrderEvent = {
  type: "order.artifact.added",
  timestamp: "2026-08-01T12:00:00.000Z",
  event: {
    order: { id: ORDER.id, title: ORDER.title },
    artifact: {
      id: "art-pr-1",
      type: "pr",
      data: {
        url: "https://github.com/example/repo/pull/42",
        title: "Draft implementation",
        state: "open",
      },
    },
  },
};

function renderTimeline(artifacts?: FactoriesWorkOrderArtifact[]) {
  return render(
    <MemoryRouter>
      <WorkOrderActivityTimeline
        organizationId="org-1"
        factoryKey="factory-1"
        order={ORDER}
        events={[ARTIFACT_ADDED_EVENT]}
        artifacts={artifacts}
      />
    </MemoryRouter>,
  );
}

describe("WorkOrderActivityTimeline artifact chip", () => {
  it("renders the event's own snapshot when no live artifact data is provided", () => {
    renderTimeline();
    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });

  it("overlays live artifact data over the stale event-time snapshot", () => {
    renderTimeline([
      {
        id: "art-pr-1",
        type: "TYPE_PR",
        data: {
          url: "https://github.com/example/repo/pull/42",
          title: "Draft implementation",
          state: "merged",
        },
      },
    ]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-purple-600");
  });

  it("falls back to the snapshot when the artifact isn't in the current list", () => {
    renderTimeline([{ id: "some-other-artifact", type: "TYPE_PR", data: { url: "https://x", state: "merged" } }]);

    const link = screen.getByRole("link");
    expect(link.querySelector("svg")).toHaveClass("text-emerald-600");
  });
});

const COMMENT_EVENT: FactoriesWorkOrderEvent = {
  id: "comment-event-1",
  type: "order.comment.added",
  timestamp: "2026-08-01T12:00:00.000Z",
  event: {
    order: { id: ORDER.id, title: ORDER.title },
    body: "Looks great!",
    author: { kind: "user", userId: "user-1" },
    reactions: [{ emoji: "+1", count: 1, reactedByMe: true }],
  },
};

describe("WorkOrderActivityTimeline comment reactions", () => {
  it("renders existing reaction pills and calls the remove handler when the caller's own pill is clicked", async () => {
    const user = userEvent.setup();
    const onRemoveCommentReaction = vi.fn();

    render(
      <MemoryRouter>
        <WorkOrderActivityTimeline
          organizationId="org-1"
          factoryKey="factory-1"
          order={ORDER}
          events={[COMMENT_EVENT]}
          canReactToComments
          onAddCommentReaction={vi.fn()}
          onRemoveCommentReaction={onRemoveCommentReaction}
        />
      </MemoryRouter>,
    );

    const pill = screen.getByTestId("work-order-comment-reaction-+1");
    expect(pill).toHaveTextContent("1");
    await user.click(pill);

    expect(onRemoveCommentReaction).toHaveBeenCalledWith("comment-event-1", "+1");
  });

  it("adds a new reaction from the picker", async () => {
    const user = userEvent.setup();
    const onAddCommentReaction = vi.fn();

    render(
      <MemoryRouter>
        <WorkOrderActivityTimeline
          organizationId="org-1"
          factoryKey="factory-1"
          order={ORDER}
          events={[COMMENT_EVENT]}
          canReactToComments
          onAddCommentReaction={onAddCommentReaction}
          onRemoveCommentReaction={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByTestId("work-order-comment-add-reaction"));
    await user.click(screen.getByTestId("work-order-comment-reaction-picker-rocket"));

    expect(onAddCommentReaction).toHaveBeenCalledWith("comment-event-1", "rocket");
  });

  it("does not call the handlers when the viewer can't react", async () => {
    const user = userEvent.setup();
    const onRemoveCommentReaction = vi.fn();

    render(
      <MemoryRouter>
        <WorkOrderActivityTimeline
          organizationId="org-1"
          factoryKey="factory-1"
          order={ORDER}
          events={[COMMENT_EVENT]}
          canReactToComments={false}
          onAddCommentReaction={vi.fn()}
          onRemoveCommentReaction={onRemoveCommentReaction}
        />
      </MemoryRouter>,
    );

    const pill = screen.getByTestId("work-order-comment-reaction-+1");
    expect(pill).toBeDisabled();
    await user.click(pill);

    expect(onRemoveCommentReaction).not.toHaveBeenCalled();
    expect(screen.getByTestId("work-order-comment-add-reaction")).toBeDisabled();
  });

  it("hides the reaction bar entirely for a comment with no reactions when the viewer can't react", () => {
    const noReactionsEvent: FactoriesWorkOrderEvent = {
      ...COMMENT_EVENT,
      event: { ...COMMENT_EVENT.event, reactions: [] },
    };

    render(
      <MemoryRouter>
        <WorkOrderActivityTimeline
          organizationId="org-1"
          factoryKey="factory-1"
          order={ORDER}
          events={[noReactionsEvent]}
          canReactToComments={false}
        />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("work-order-comment-reactions")).not.toBeInTheDocument();
  });
});
