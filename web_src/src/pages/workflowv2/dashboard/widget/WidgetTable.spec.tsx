import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";

import type { SuperplaneComponentsNode } from "@/api-client";
import { DashboardContextProvider } from "../DashboardContext";
import { WidgetTable } from "./WidgetTable";
import type { WidgetTableRender } from "./types";

const START_NODE: SuperplaneComponentsNode = {
  id: "start-id",
  name: "start",
  type: "TYPE_TRIGGER",
  configuration: {
    templates: [{ name: "default", payload: { issue: { number: 0 } } }],
  },
};

const ROWS = [
  { id: "mem-1", namespace: "environments", service: "api", status: "failed", pr_number: "42" },
  { id: "mem-2", namespace: "environments", service: "web", status: "passed", pr_number: "7" },
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
      node: "start",
      show: 'status == "failed"',
    },
  ],
};

function renderTable({
  canRunNodes,
  onTriggerNode,
}: {
  canRunNodes: boolean;
  onTriggerNode?: (nodeId: string, options?: { hookName?: string; successLabel?: string }) => Promise<void>;
}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <DashboardContextProvider
          canvasId="canvas-1"
          organizationId="org-1"
          nodes={[START_NODE]}
          canRunNodes={canRunNodes}
          onTriggerNode={onTriggerNode ? (id, opts) => void onTriggerNode(id, opts) : undefined}
        >
          <WidgetTable render={RENDER} rows={ROWS} isLoading={false} />
        </DashboardContextProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("WidgetTable row actions — permission gating", () => {
  it("invokes the trigger callback when canRunNodes is true", async () => {
    const onTrigger = vi.fn().mockResolvedValue(undefined);
    renderTable({ canRunNodes: true, onTriggerNode: onTrigger });
    const triggers = screen.getAllByTestId("widget-row-action-start");
    expect(triggers).toHaveLength(1);
    expect(triggers[0]).not.toBeDisabled();
    fireEvent.click(triggers[0]);
    expect(onTrigger).toHaveBeenCalledWith(
      "start-id",
      expect.objectContaining({
        hookName: "run",
        successLabel: "Redeploy",
      }),
    );
  });

  it("renders trigger disabled when canRunNodes is false", () => {
    const onTrigger = vi.fn();
    renderTable({ canRunNodes: false, onTriggerNode: onTrigger });
    const trigger = screen.getByTestId("widget-row-action-start");
    expect(trigger).toBeDisabled();
    fireEvent.click(trigger);
    expect(onTrigger).not.toHaveBeenCalled();
  });

  it("evaluates per-row show expressions", () => {
    renderTable({ canRunNodes: true });
    expect(screen.queryAllByTestId("widget-row-action-start")).toHaveLength(1);
  });
});
