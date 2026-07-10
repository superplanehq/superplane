import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunInspectorChrome } from "./RunInspectorChrome";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

describe("RunInspectorChrome", () => {
  it("navigates between loaded sidebar runs", () => {
    const onNavigateRun = vi.fn();

    render(
      <RunInspectorChrome
        runId="run-1"
        newerRunId="run-newer"
        olderRunId="run-older"
        onNavigateRun={onNavigateRun}
        onClose={() => {}}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Newer run" }));
    fireEvent.click(screen.getByRole("button", { name: "Older run" }));

    expect(onNavigateRun).toHaveBeenNthCalledWith(1, "run-newer");
    expect(onNavigateRun).toHaveBeenNthCalledWith(2, "run-older");
  });

  it("copies a run-only link", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    window.history.pushState({}, "", "/org-1/apps/app-1?run=old-run&node=action-1");

    render(<RunInspectorChrome runId="run-1" onClose={() => {}} />);

    fireEvent.click(screen.getByRole("button", { name: "Copy run link" }));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith("http://localhost:3000/org-1/apps/app-1?run=run-1");
    });
  });
});
