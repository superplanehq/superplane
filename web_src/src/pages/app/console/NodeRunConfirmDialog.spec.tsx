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
  it("renders a read-only payload preview when the template declares no parameters", () => {
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
    const preview = screen.getByTestId("node-run-confirm-parameters");
    expect(preview.textContent).toContain('"template": "manual"');
  });

  it("keeps long parameter values inside a horizontally scrollable preview", () => {
    render(
      <NodeRunConfirmDialog
        open
        onOpenChange={() => undefined}
        resolved={resolvedFor({
          ...NODE_NO_PARAMS,
          configuration: {
            templates: [{ name: "manual", payload: { token: "a".repeat(200) } }],
          },
        })}
        templateName="manual"
        onConfirm={vi.fn().mockResolvedValue(undefined)}
      />,
    );

    const preview = screen.getByTestId("node-run-confirm-parameters");
    expect(preview.getAttribute("class")).toContain("overflow-x-auto");
    expect(preview.getAttribute("class")).toContain("whitespace-pre");
    expect(preview.getAttribute("class")).not.toContain("break-all");
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

  it("keeps the dialog open and shows the error when onConfirm rejects", async () => {
    const onConfirm = vi.fn().mockRejectedValue(new Error("boom"));
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
    await waitFor(() => expect(onConfirm).toHaveBeenCalledTimes(1));
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByTestId("node-run-confirm-submit")).toBeTruthy();
    expect(screen.getByTestId("node-run-confirm-error").textContent).toBe("boom");
  });
});
