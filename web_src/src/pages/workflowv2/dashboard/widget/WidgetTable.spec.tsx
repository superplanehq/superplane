import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";
import { DashboardContextProvider } from "../DashboardContext";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

const DEPLOY_NODE: SuperplaneComponentsNode = {
  id: "deploy-id",
  name: "deploy",
  type: "TYPE_TRIGGER",
};

const ROWS = [
  { id: "exec-1", service: "api", status: "failed", nodeId: "deploy-id" },
  { id: "exec-2", service: "web", status: "passed", nodeId: "deploy-id" },
];

const RENDER: WidgetTableRender = {
  kind: "table",
  columns: [
    { field: "service", label: "Service" },
    { field: "status", format: "status" },
  ],
  rowActions: [
    {
      kind: "trigger",
      label: "Redeploy",
      target: "nodeId",
      show: 'row.status == "failed"',
    },
    {
      kind: "cancel",
      label: "Cancel",
      target: "id",
    },
  ],
};

function renderTable({
  canRunNodes,
  onTriggerNode,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: { templateName?: string; triggerName?: string }) => void;
}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DashboardContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[DEPLOY_NODE]}
          canRunNodes={canRunNodes}
          onTriggerNode={onTriggerNode}
        >
          <WidgetTable render={RENDER} rows={ROWS} isLoading={false} />
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable row actions — permission gating", () => {
  it("invokes the trigger callback when canRunNodes is true", () => {
    const onTrigger = vi.fn();
    renderTable({ canRunNodes: true, onTriggerNode: onTrigger });
    const triggers = screen.getAllByTestId("widget-row-action-trigger");
    // Only the failed row matches the `show` expression.
    expect(triggers).toHaveLength(1);
    expect(triggers[0]).not.toBeDisabled();
    fireEvent.click(triggers[0]);
    expect(onTrigger).toHaveBeenCalledWith("deploy-id", { templateName: undefined });
  });

  it("renders trigger and cancel actions disabled when canRunNodes is false", () => {
    const onTrigger = vi.fn();
    renderTable({ canRunNodes: false, onTriggerNode: onTrigger });
    const trigger = screen.getByTestId("widget-row-action-trigger");
    const cancels = screen.getAllByTestId("widget-row-action-cancel");
    expect(trigger).toBeDisabled();
    cancels.forEach((c) => expect(c).toBeDisabled());
    expect(trigger).toHaveAttribute("title", expect.stringMatching(/do not have permission/i));
    fireEvent.click(trigger);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("evaluates per-row `show` expressions even when canRunNodes is true", () => {
    renderTable({ canRunNodes: true });
    // The non-failed row should not show the trigger button.
    const triggers = screen.queryAllByTestId("widget-row-action-trigger");
    expect(triggers).toHaveLength(1);
  });
});
