import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";

import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import {
  buildWorkOrderTimelineView,
  type WorkOrderTimelineEvent,
  type WorkOrderTimelineStep,
} from "../lib/workOrderTimelineEvents";
import { stepExecutionEvent } from "../lib/workOrderTimelineFromEvents.testHelpers";
import { DispatchTimelineItem } from "./DispatchTimelineItem";

function dispatchEvent(
  comments: Array<{ body: string }>,
  execution: WorkOrderTimelineStep["execution"] = {
    id: "run-1",
    step: "Build",
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
  },
): WorkOrderTimelineEvent {
  return {
    id: "dispatch-1",
    kind: "dispatched",
    at: "2026-08-04T12:00:00.000Z",
    lineId: "line-1",
    lineName: "plan-and-implement",
    title: "Dispatched to plan-and-implement",
    steps: [
      {
        id: "step-1",
        stepName: "Build",
        at: "2026-08-04T12:00:00.000Z",
        startedAt: "2026-08-04T12:00:00.000Z",
        comments,
        execution,
      },
    ],
  };
}

describe("DispatchTimelineItem", () => {
  it("renders a step comment as plain body text, without repeating the automation name or run link", () => {
    // The step's title already names the automation line and links to its
    // run, so the comment itself must show only its body — no label and no
    // link — even when the underlying comment carries run details.
    render(
      <DispatchTimelineItem
        event={dispatchEvent([{ body: "Applying the fix now." }])}
        organizationId="org-1"
        factoryKey="factory-1"
        orderNumber="1"
        isLatestDispatch
      />,
    );

    expect(screen.getByText("Applying the fix now.")).toBeInTheDocument();
    expect(screen.queryByText("CI")).not.toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("explains a hosted credit failure and links to billing", () => {
    render(
      <MemoryRouter>
        <DispatchTimelineItem
          event={dispatchEvent([], {
            id: "run-1",
            step: "Build",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            failureReason: "no_hosted_credit",
          })}
          organizationId="org-1"
          factoryKey="factory-1"
          orderNumber="1"
          isLatestDispatch
        />
      </MemoryRouter>,
    );

    expect(
      screen.getByText("This agent run is blocked. The organization has no SuperPlane hosted credit.", {
        exact: false,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath("org-1", "factory-1", "organization", "billing"),
    );
  });

  it("shows the billing notice when the timeline joins a step event with the task execution", () => {
    const events = [stepExecutionEvent("step.execution.finished", "2026-08-04T12:00:00.000Z", "finished", "failed")];
    const fromEventsOnly = buildWorkOrderTimelineView(events);
    expect(fromEventsOnly.events[0]?.steps?.[0]?.execution.failureReason).toBeUndefined();

    const view = buildWorkOrderTimelineView(events, undefined, [
      {
        id: "execution-1",
        result: "RESULT_FAILED",
        failureReason: "no_hosted_credit",
        run: { id: "run-1" },
      },
    ]);
    const event = view.events[0];
    if (!event) {
      throw new Error("expected a dispatch event");
    }
    expect(event.steps?.[0]?.execution.failureReason).toBe("no_hosted_credit");

    render(
      <MemoryRouter>
        <DispatchTimelineItem
          event={event}
          organizationId="org-1"
          factoryKey="factory-1"
          orderNumber="1"
          isLatestDispatch
        />
      </MemoryRouter>,
    );

    expect(
      screen.getByText("This agent run is blocked. The organization has no SuperPlane hosted credit.", {
        exact: false,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath("org-1", "factory-1", "organization", "billing"),
    );
  });
});
