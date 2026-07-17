import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { useCanvas } from "@/hooks/useCanvasData";

import { AppCanvasNodeFieldRenderer, filterAppCanvasNodes, resolveAppCanvasId } from "./AppCanvasNodeFieldRenderer";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: vi.fn(),
}));

const mockUseCanvas = vi.mocked(useCanvas);

function nodeField(): ConfigurationField {
  return {
    name: "node",
    label: "Node",
    type: "app-canvas-node",
    typeOptions: {
      appCanvasNode: {
        nodeTypes: ["trigger"],
        componentTypes: ["onRun"],
        parameters: [{ name: "app", valueFrom: { field: "app" } }],
      },
    },
  };
}

function ControlledRenderer({
  initialValue,
  allValues,
}: {
  initialValue?: string;
  allValues?: Record<string, unknown>;
}) {
  const [value, setValue] = useState<string | undefined>(initialValue);
  return (
    <>
      <span data-testid="current-value">{value ?? ""}</span>
      <AppCanvasNodeFieldRenderer
        field={nodeField()}
        value={value}
        onChange={setValue}
        allValues={allValues}
        organizationId="org-1"
      />
    </>
  );
}

beforeEach(() => {
  mockUseCanvas.mockReturnValue({
    data: {
      id: "canvas_target",
      spec: {
        nodes: [
          { id: "onRun-trigger", name: "On Run", type: "TYPE_TRIGGER", component: "onRun" },
          { id: "on-broadcast", name: "On Broadcast", type: "TYPE_TRIGGER", component: "onBroadcast" },
          { id: "send-email", name: "Send Email", type: "TYPE_ACTION", component: "sendEmail" },
        ],
      },
    },
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useCanvas>);
});

describe("AppCanvasNodeFieldRenderer helpers", () => {
  it("resolves the app id from parameter refs", () => {
    expect(
      resolveAppCanvasId([{ name: "app", valueFrom: { field: "app" } }], {
        app: "canvas_target",
      }),
    ).toBe("canvas_target");
  });

  it("filters nodes by type and component", () => {
    const nodes = filterAppCanvasNodes(
      [
        { id: "onRun-trigger", name: "On Run", type: "TYPE_TRIGGER", component: "onRun" },
        { id: "on-broadcast", name: "On Broadcast", type: "TYPE_TRIGGER", component: "onBroadcast" },
        { id: "send-email", name: "Send Email", type: "TYPE_ACTION", component: "sendEmail" },
      ],
      ["trigger"],
      ["onRun"],
    );

    expect(nodes.map((node) => node.id)).toEqual(["onRun-trigger"]);
  });
});

describe("AppCanvasNodeFieldRenderer", () => {
  it("prompts for an app before loading nodes", () => {
    render(<ControlledRenderer />);

    expect(screen.getByText("Choose the target app before selecting a node.")).toBeInTheDocument();
    expect(mockUseCanvas).toHaveBeenCalledWith("org-1", "", expect.objectContaining({ enabled: false }));
  });

  it("shows only matching nodes for the selected app", async () => {
    render(<ControlledRenderer allValues={{ app: "canvas_target" }} />);

    await userEvent.click(screen.getByRole("combobox"));
    expect(screen.getByText("On Run")).toBeInTheDocument();
    expect(screen.queryByText("On Broadcast")).not.toBeInTheDocument();
    expect(screen.queryByText("Send Email")).not.toBeInTheDocument();
  });

  it("stores the selected node id", async () => {
    render(<ControlledRenderer allValues={{ app: "canvas_target" }} />);

    await userEvent.click(screen.getByRole("combobox"));
    await userEvent.click(screen.getByText("On Run"));

    expect(screen.getByTestId("current-value")).toHaveTextContent("onRun-trigger");
  });

  it("clears stale node values when the app changes", async () => {
    const { rerender } = render(
      <ControlledRenderer initialValue="onRun-trigger" allValues={{ app: "canvas_target" }} />,
    );

    mockUseCanvas.mockReturnValue({
      data: {
        id: "canvas_other",
        spec: {
          nodes: [{ id: "other-trigger", name: "Other Trigger", type: "TYPE_TRIGGER", component: "onRun" }],
        },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useCanvas>);

    rerender(<ControlledRenderer initialValue="onRun-trigger" allValues={{ app: "canvas_other" }} />);

    await waitFor(() => {
      expect(screen.getByTestId("current-value")).toHaveTextContent("");
    });
  });

  it("requires organization context", () => {
    render(
      <AppCanvasNodeFieldRenderer
        field={nodeField()}
        value={undefined}
        onChange={vi.fn()}
        allValues={{ app: "canvas_target" }}
      />,
    );

    expect(screen.getByText("App canvas node field requires organization context.")).toBeInTheDocument();
  });
});
