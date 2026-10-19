import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { AutomationRunHistory } from "./AutomationRunHistory";
import { implementationRuns } from "./fixtures";

// CopyButton owns clipboard feedback; this covers the history-to-copy wiring,
// where accidentally passing the displayed prefix would lose the real identity.
describe("AutomationRunHistory", () => {
  it("copies the complete run ID without selecting the run", async () => {
    const writeText = vi.fn(() => Promise.resolve());
    const clipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    const onSelectRun = vi.fn();
    const run = implementationRuns[0];
    try {
      render(<AutomationRunHistory runs={[run]} onSelectRun={onSelectRun} />);
      await act(async () => {
        fireEvent.click(screen.getByRole("button", { name: `Copy full run ID ${run.id}` }));
      });
      expect(writeText).toHaveBeenCalledWith(run.id);
      expect(onSelectRun).not.toHaveBeenCalled();
      fireEvent.click(screen.getByRole("button", { name: `Open run ${run.id}` }));
      expect(onSelectRun).toHaveBeenCalledWith(run);
    } finally {
      if (clipboard) Object.defineProperty(navigator, "clipboard", clipboard);
      else Reflect.deleteProperty(navigator, "clipboard");
    }
  });

  it("finds a run by its full ID and recovers from an empty search", () => {
    render(<AutomationRunHistory runs={implementationRuns} onSelectRun={() => {}} />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), {
      target: { value: implementationRuns[1].id },
    });
    expect(screen.getAllByRole("button", { name: /^Open run/ })).toHaveLength(1);
    expect(screen.getByRole("button", { name: `Open run ${implementationRuns[1].id}` })).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Search task or run ID" }), {
      target: { value: "missing-task" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getAllByRole("button", { name: /^Open run/ })).toHaveLength(implementationRuns.length);
  });
});
