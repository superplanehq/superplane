import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { DispatchTimelineItem } from "./DispatchTimelineItem";

function dispatchEvent(
  comments: Array<{ body: string; label?: string; sourceRunId?: string; sourceAppId?: string }>,
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
  it("renders a step comment's automation label as a link to its run", () => {
    render(
      <DispatchTimelineItem
        event={dispatchEvent([{ body: "Applying the fix now.", label: "CI", sourceRunId: "run-42", sourceAppId: "app-1" }])}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="order-1"
        isLatestDispatch
      />,
    );

    const link = screen.getByRole("link", { name: /CI/ });
    expect(link).toHaveAttribute("href", expect.stringContaining("/apps/app-1"));
    expect(link).toHaveAttribute("href", expect.stringContaining("run=run-42"));
    expect(screen.getByText(/Applying the fix now\./)).toBeInTheDocument();
  });

  it("renders a step comment without run info as plain text", () => {
    render(
      <DispatchTimelineItem
        event={dispatchEvent([{ body: "Heads up, retrying.", label: "CI" }])}
        organizationId="org-1"
        factoryId="factory-1"
        orderId="order-1"
        isLatestDispatch
      />,
    );

    expect(screen.getByText("CI")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /CI/ })).not.toBeInTheDocument();
  });
});
