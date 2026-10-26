import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { RunHistoryTable, type RunHistoryRowData } from "./RunHistoryTable";

const runs: RunHistoryRowData[] = [
  {
    id: "c2388973-c0af-4847-93dd-3d95c372df3f",
    subject: { key: "project#123", title: "Keep the source issue attached" },
    status: "Task created",
    statusContent: <span>Task created</span>,
    startedAt: new Date().toISOString(),
    duration: "1s",
    searchText: "SUPER-987",
  },
  {
    id: "ceaa564d-c55a-4936-82d0-dad09f3e0551",
    subject: { key: "SUPER-988", title: "Use the selected model" },
    status: "Running",
    statusContent: <span>Running</span>,
    startedAt: new Date().toISOString(),
    duration: "2m",
  },
];

describe("RunHistoryTable", () => {
  it("opens runs from the row while copying the complete ID leaves the list open", async () => {
    const onSelectRun = vi.fn();
    const writeText = vi.fn(() => Promise.resolve());
    const clipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    try {
      render(<RunHistoryTable runs={runs} onSelectRun={onSelectRun} onRetry={() => {}} />);
      fireEvent.click(screen.getByText("1s"));
      expect(onSelectRun).toHaveBeenCalledTimes(1);
      expect(onSelectRun).toHaveBeenLastCalledWith(runs[0].id);
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: `Copy full run ID ${runs[0].id}` }));
      });
      expect(writeText).toHaveBeenCalledWith(runs[0].id);
      expect(onSelectRun).toHaveBeenCalledTimes(1);
      fireEvent.click(screen.getByRole("button", { name: `Open run ${runs[1].id}` }));
      expect(onSelectRun).toHaveBeenLastCalledWith(runs[1].id);
    } finally {
      if (clipboard) Object.defineProperty(navigator, "clipboard", clipboard);
      else Reflect.deleteProperty(navigator, "clipboard");
    }
  });

  it("finds source references, linked task IDs, and full run IDs using the same search", () => {
    render(<RunHistoryTable runs={runs} onSelectRun={() => {}} onRetry={() => {}} />);
    const search = screen.getByRole("textbox", { name: "Search task, source, or run ID" });
    for (const query of ["project#123", "SUPER-987", runs[0].id]) {
      fireEvent.change(search, { target: { value: query } });
      expect(within(screen.getByRole("table")).getAllByRole("button", { name: /^Open run / })).toHaveLength(1);
      expect(screen.getByRole("button", { name: `Open run ${runs[0].id}` })).toBeInTheDocument();
    }
    fireEvent.change(search, { target: { value: "missing item" } });
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(within(screen.getByRole("table")).getAllByRole("button", { name: /^Open run / })).toHaveLength(2);
  });
});
