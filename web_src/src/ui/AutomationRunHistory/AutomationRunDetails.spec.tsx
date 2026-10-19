import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { AutomationRunDetails } from "./AutomationRunDetails";
import { implementationRuns } from "./fixtures";
import type { RunActivity } from "./types";

describe("AutomationRunDetails live activity", () => {
  it("selects an available activity after a queue item disappears and preserves user selection", () => {
    const start = implementationRuns[0].activities[0];
    const execution = implementationRuns[0].activities[1];
    const queued: RunActivity = { ...execution, id: "queue:1", status: "Pending" };
    const props = {
      onBack: () => {},
      onRetryLogs: () => {},
      renderLogs: (activity: RunActivity) => <p>{`Logs for ${activity.id}`}</p>,
    };
    const view = (activities: RunActivity[]) => (
      <AutomationRunDetails {...props} run={{ ...implementationRuns[0], activities }} />
    );
    const { rerender } = render(view([start, queued]));
    expect(screen.getByText("Logs for queue:1")).toBeInTheDocument();
    rerender(view([start, execution]));
    expect(screen.getByText(`Logs for ${execution.id}`)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Run started/ }));
    rerender(view([start, { ...execution, status: "Failed" }]));
    expect(screen.getByText(`Logs for ${start.id}`)).toBeInTheDocument();
  });

  it("selects activity when an initially empty run receives it", () => {
    const props = { onBack: () => {}, onRetryLogs: () => {} };
    const { rerender } = render(<AutomationRunDetails {...props} run={{ ...implementationRuns[0], activities: [] }} />);
    rerender(<AutomationRunDetails {...props} run={implementationRuns[0]} />);
    expect(screen.getByRole("heading", { name: "Implement task" })).toBeInTheDocument();
  });
});
