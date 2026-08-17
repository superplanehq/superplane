import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoryApp } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";

import { FactoryLineStepEditor } from "./FactoryLineStepEditor";
import type { DraftStep } from "./lib/factoryLineFormShared";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: vi.fn(),
}));

const mockUseCanvas = vi.mocked(useCanvas);

const app: FactoryApp = { id: "app-1", name: "Deploy App" };
const apps = [app];
const appById = new Map([[app.id!, app]]);

function step(overrides: Partial<DraftStep> = {}): DraftStep {
  return { name: "step", appId: "app-1", entrypoint: "", ...overrides };
}

function renderEditor(draftStep: DraftStep) {
  return render(
    <FactoryLineStepEditor
      organizationId="org-1"
      index={0}
      step={draftStep}
      apps={apps}
      appById={appById}
      onChange={vi.fn()}
    />,
  );
}

function triggerSelectTrigger() {
  return screen.getByLabelText("Trigger");
}

describe("FactoryLineStepEditor trigger states", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows a load-error message and disables the trigger select when the canvas request fails", () => {
    mockUseCanvas.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("network error"),
    } as unknown as ReturnType<typeof useCanvas>);

    renderEditor(step());

    expect(screen.getByText("Failed to load triggers for this app.")).toBeInTheDocument();
    expect(screen.queryByText("Deploy App has no triggers yet.")).not.toBeInTheDocument();
    expect(triggerSelectTrigger()).toHaveAttribute("data-disabled");
  });

  it("shows the existing empty-state message when the canvas genuinely has no triggers", () => {
    mockUseCanvas.mockReturnValue({
      data: { id: "canvas-1", spec: { nodes: [] } },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useCanvas>);

    renderEditor(step());

    expect(screen.getByText("Deploy App has no triggers yet.")).toBeInTheDocument();
  });

  it("flags a saved entrypoint that is no longer present in the loaded canvas", () => {
    mockUseCanvas.mockReturnValue({
      data: {
        id: "canvas-1",
        spec: { nodes: [{ id: "build", name: "Build", type: "TYPE_TRIGGER" }] },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useCanvas>);

    renderEditor(step({ entrypoint: "deploy" }));

    expect(
      screen.getByText("The selected trigger is no longer available. Choose another trigger."),
    ).toBeInTheDocument();
  });

  it("shows no warning once triggers load normally and the saved entrypoint is still present", () => {
    mockUseCanvas.mockReturnValue({
      data: {
        id: "canvas-1",
        spec: { nodes: [{ id: "deploy", name: "Deploy", type: "TYPE_TRIGGER" }] },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useCanvas>);

    renderEditor(step({ entrypoint: "deploy" }));

    expect(screen.queryByText("Failed to load triggers for this app.")).not.toBeInTheDocument();
    expect(screen.queryByText("Deploy App has no triggers yet.")).not.toBeInTheDocument();
    expect(
      screen.queryByText("The selected trigger is no longer available. Choose another trigger."),
    ).not.toBeInTheDocument();
    expect(triggerSelectTrigger()).not.toHaveAttribute("data-disabled");
  });
});
