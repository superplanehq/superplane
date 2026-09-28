import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";

import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import type { WorkOrderTimelineEvent, WorkOrderTimelineStep } from "../lib/workOrderTimelineEvents";
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
      screen.getByText("This step did not start. The organization has no SuperPlane hosted credit.", {
        exact: false,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add credits" })).toHaveAttribute(
      "href",
      factorySettingsSectionPath("org-1", "factory-1", "organization", "billing"),
    );
  });
});
