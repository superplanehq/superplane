import type { ComponentProps } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { describe, expect, it, vi } from "bun:test";
import type { MergeConfidenceConfigShell } from "./MergeConfidenceConfigShell";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";

vi.mock("./MergeConfidenceConfigShell", () => ({
  MergeConfidenceConfigShell: (props: ComponentProps<typeof MergeConfidenceConfigShell>) => (
    <>
      <button data-testid="automation" onClick={() => props.onTabChange("automation")}>
        Configuration
      </button>
      <button data-testid="runs" onClick={() => props.onTabChange("runs")}>
        Runs
      </button>
      {props.tab === "runs" && (
        <>
          <button data-testid="open-run" onClick={() => props.onSelectedRunIdChange("run-123")}>
            Open run
          </button>
          <button data-testid="history" onClick={() => props.onSelectedRunIdChange(null)}>
            History
          </button>
        </>
      )}
      <output data-testid="view">
        {props.tab}:{props.selectedRunId ?? "index"}
      </output>
    </>
  ),
}));

function BrowserControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="url">
        {location.pathname}
        {location.search}
      </output>
      <button data-testid="back" onClick={() => navigate(-1)}>
        Back
      </button>
      <button data-testid="forward" onClick={() => navigate(1)}>
        Forward
      </button>
    </>
  );
}

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <BrowserControls />
      <MergeConfidenceConfigModal title="Implement" canvasId="canvas" onClose={() => {}} onSaveNode={() => {}} />
    </MemoryRouter>,
  );
}

describe("automation run navigation", () => {
  it("keeps tab and run selection in shareable URLs and restores them with browser navigation", async () => {
    const base = "/org/workspaces/sp/lines/line?automationView=canvas&keep=1";
    const view = renderAt(base);
    fireEvent.click(screen.getByTestId("runs"));
    expect(screen.getByTestId("url").textContent).toContain("automationTab=runs");
    fireEvent.click(screen.getByTestId("open-run"));
    const permalink = screen.getByTestId("url").textContent!;
    expect(permalink).toContain("automationRun=run-123");
    expect(permalink).toContain("keep=1");
    fireEvent.click(screen.getByTestId("back"));
    await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent("runs:index"));
    fireEvent.click(screen.getByTestId("back"));
    await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent("automation:index"));
    fireEvent.click(screen.getByTestId("forward"));
    await waitFor(() => expect(screen.getByTestId("view")).toHaveTextContent("runs:index"));
    view.unmount();
    renderAt(permalink);
    expect(screen.getByTestId("view")).toHaveTextContent("runs:run-123");
    fireEvent.click(screen.getByTestId("history"));
    expect(screen.getByTestId("url").textContent).not.toContain("automationRun=");
  });
});
