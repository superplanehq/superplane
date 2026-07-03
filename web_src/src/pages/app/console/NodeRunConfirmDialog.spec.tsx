import { render, screen, fireEvent, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";

import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";

const NODE_NO_PARAMS: SuperplaneComponentsNode = {
  id: "node-1",
  name: "deploy-prod",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "manual", payload: { reason: "console" } }],
  },
};

const NODE_WITH_PARAMS: SuperplaneComponentsNode = {
  id: "node-2",
  name: "deploy-stg",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [
      {
        name: "manual",
        payload: { reason: "console" },
        parameters: [
          { name: "branch", type: "string", defaultString: "main" },
          { name: "approve", type: "boolean", defaultBoolean: true },
        ],
      },
    ],
  },
};

function resolvedFor(node: SuperplaneComponentsNode) {
  return { node, label: node.name ?? node.id ?? "" };
}

describe("NodeRunConfirmDialog", () => {
  it("renders a bare confirmation (no fields, no payload preview) when the template declares no parameters", () => {
    render(
      <NodeRunConfirmDialog
        open
        onOpenChange={() => undefined}
        resolved={resolvedFor(NODE_NO_PARAMS)}
        templateName="manual"
        onConfirm={vi.fn().mockResolvedValue(undefined)}
      />,
    );
    expect(screen.queryByTestId("node-run-confirm-fields")).toBeNull();
    expect(screen.queryByTestId("node-run-confirm-parameters")).toBeNull();
    expect(screen.queryByTestId("node-run-confirm-payload-toggle")).toBeNull();
    expect(screen.getByTestId("node-run-confirm-confirm-message").textContent).toContain("deploy-prod");
  });

  it("submits coerced parameter values along with the template name", async () => {
    const onConfirm = vi.fn().mockResolvedValue(undefined);
    const onOpenChange = vi.fn();
    render(
      <NodeRunConfirmDialog
        open
        onOpenChange={onOpenChange}
        resolved={resolvedFor(NODE_WITH_PARAMS)}
        templateName="manual"
        onConfirm={onConfirm}
      />,
    );

    const branchInput = screen.getByLabelText("branch") as HTMLInputElement;
    expect(branchInput.value).toBe("main");
    fireEvent.change(branchInput, { target: { value: "release/v2" } });

    const approveCheckbox = screen.getByLabelText("approve");
    fireEvent.click(approveCheckbox);

    await act(async () => {
      fireEvent.click(screen.getByTestId("node-run-confirm-submit"));
    });

    await waitFor(() => expect(onConfirm).toHaveBeenCalledTimes(1));
    expect(onConfirm).toHaveBeenCalledWith({
      template: "manual",
      branch: "release/v2",
      approve: false,
    });
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("hands off the template and closes immediately on confirm (no internal loading state)", async () => {
    const onConfirm = vi.fn();
    const onOpenChange = vi.fn();
    render(
      <NodeRunConfirmDialog
        open
        onOpenChange={onOpenChange}
        resolved={resolvedFor(NODE_NO_PARAMS)}
        templateName="manual"
        onConfirm={onConfirm}
      />,
    );
    await act(async () => {
      fireEvent.click(screen.getByTestId("node-run-confirm-submit"));
    });
    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onConfirm).toHaveBeenCalledWith({ template: "manual" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
