import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { AutomationRunDetails } from "./AutomationRunDetails";
import { implementationRuns } from "./fixtures";
import type { RunActivity } from "./types";

describe("AutomationRunDetails live activity", () => {
  it("opens event data for a source activity and keeps agent logs and outputs accessible", async () => {
    const original = implementationRuns[2];
    const run = {
      ...original,
      activities: original.activities.map((activity) =>
        activity.id === "start"
          ? {
              ...activity,
              logs: undefined,
              defaultTab: "outputs" as const,
              outputsLabel: "Event data",
            }
          : activity,
      ),
    };
    render(<AutomationRunDetails run={run} onBack={() => {}} onRetryLogs={() => {}} />);
    expect(screen.getByRole("tab", { name: "Event data", selected: true })).toBeInTheDocument();
    expect(screen.getByRole("tabpanel")).toHaveTextContent(original.task!.title);
    expect(screen.queryByRole("tab", { name: "Logs" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Implement task/ }));
    expect(screen.getByRole("tab", { name: "Logs", selected: true })).toBeInTheDocument();
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Changes committed and pushed.");
    await act(async () => {
      fireEvent.mouseDown(screen.getByRole("tab", { name: "Outputs" }), { button: 0, ctrlKey: false });
    });
    expect(screen.getByRole("tabpanel")).toHaveTextContent("completed");
  });
  it("shows the recorded instructions for the selected activity", async () => {
    const run = implementationRuns[0];
    const instructions = "Avoid new dependencies.\nInclude API examples in the PR.";
    render(
      <AutomationRunDetails
        run={{
          ...run,
          activities: run.activities.map((activity) =>
            activity.id === "implement" ? { ...activity, additionalInstructions: instructions } : activity,
          ),
        }}
        onBack={() => {}}
        onRetryLogs={() => {}}
      />,
    );
    const tab = screen.getByRole("tab", { name: "Instructions" });
    await act(async () => {
      fireEvent.mouseDown(tab, { button: 0, ctrlKey: false });
    });
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Avoid new dependencies.");
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Include API examples in the PR.");
    fireEvent.click(screen.getByRole("button", { name: /Run started/ }));
    expect(screen.queryByRole("tab", { name: "Instructions" })).not.toBeInTheDocument();
  });
  it("copies the task key through its adjacent action", async () => {
    const writeText = vi.fn(() => Promise.resolve());
    const clipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    const run = implementationRuns[0];
    try {
      render(<AutomationRunDetails run={run} onBack={() => {}} onRetryLogs={() => {}} />);
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: `Copy task ID ${run.task!.id}` }));
      });
      expect(writeText).toHaveBeenCalledWith(run.task!.id);
    } finally {
      if (clipboard) Object.defineProperty(navigator, "clipboard", clipboard);
      else Reflect.deleteProperty(navigator, "clipboard");
    }
  });
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
    expect(screen.queryByRole("tab", { name: "Logs" })).not.toBeInTheDocument();
    expect(screen.getByRole("tabpanel")).toHaveTextContent(implementationRuns[0].task!.title);
  });

  it("selects activity when an initially empty run receives it", () => {
    const props = { onBack: () => {}, onRetryLogs: () => {} };
    const { rerender } = render(<AutomationRunDetails {...props} run={{ ...implementationRuns[0], activities: [] }} />);
    rerender(<AutomationRunDetails {...props} run={implementationRuns[0]} />);
    expect(screen.getByRole("heading", { name: "Implement task" })).toBeInTheDocument();
  });
});
