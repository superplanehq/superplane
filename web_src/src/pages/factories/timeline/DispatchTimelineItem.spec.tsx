import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import type { WorkOrderTimelineBroadcast, WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { DispatchTimelineItem } from "./DispatchTimelineItem";

function dispatchEvent(
  comments: Array<{ body: string }>,
  broadcasts: WorkOrderTimelineBroadcast[] = [],
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
        broadcasts,
        execution: {
          id: "run-1",
          step: "Build",
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
        },
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

  it("expands a step broadcast to show the preview URL", () => {
    render(
      <DispatchTimelineItem
        event={dispatchEvent(
          [],
          [
            {
              summary: "Preview environment is ready",
              body: "The preview environment is available.",
              url: "https://preview.example.com/orders/12",
              urlLabel: "Preview",
            },
          ],
        )}
        organizationId="org-1"
        factoryKey="factory-1"
        orderNumber="1"
        isLatestDispatch
      />,
    );

    expect(screen.queryByRole("link", { name: "Preview" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Preview environment is ready" }));
    expect(screen.getByRole("link", { name: "Preview" })).toHaveAttribute(
      "href",
      "https://preview.example.com/orders/12",
    );
    expect(screen.getByText("The preview environment is available.")).toBeInTheDocument();
  });
});
