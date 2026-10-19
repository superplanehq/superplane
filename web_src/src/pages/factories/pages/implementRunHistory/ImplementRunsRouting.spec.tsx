import { cloneElement, createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MergeConfidenceConfigShell } from "../MergeConfidenceConfigShell";
import { DEFAULT_SPLIT_CONFIG_DELETE_COPY } from "../splitConfigDeleteCopy";

const access = vi.hoisted(() => ({
  enabled: false,
  isLoading: false,
  organizationReady: true,
  retryOrganization: vi.fn(),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ ...access, has: (id: string) => id === "factory_run_history" && access.enabled }),
}));

vi.mock("./ImplementRunHistory", () => ({
  ImplementRunHistory: ({ onSelectRun }: { onSelectRun: (id: string | null) => void }) => (
    <button data-testid="connected-history" onClick={() => onSelectRun("selected-run")}>
      Open run
    </button>
  ),
}));
vi.mock("../SettingsAutomationWorkspace", () => ({
  SettingsAutomationHeaderRow: ({ tabs }: { tabs: React.ReactNode }) => <>{tabs}</>,
  SettingsAutomationWorkspace: () => <div data-testid="existing-canvas" />,
}));

describe("Implement Runs routing", () => {
  it.each([
    {
      nodeId: "implementation-agent-no-issue",
      enabled: false,
      isLoading: false,
      organizationReady: true,
      expected: false,
    },
    {
      nodeId: "implementation-agent-no-issue",
      enabled: true,
      isLoading: true,
      organizationReady: true,
      expected: false,
    },
    {
      nodeId: "implementation-agent-no-issue",
      enabled: true,
      isLoading: false,
      organizationReady: false,
      expected: false,
    },
    {
      nodeId: "implementation-agent-no-issue",
      enabled: true,
      isLoading: false,
      organizationReady: true,
      expected: true,
    },
    { nodeId: "assess-risk", enabled: true, isLoading: false, organizationReady: true, expected: false },
  ])(
    "gates Implement history while preserving the old view: %j",
    ({ nodeId, enabled, isLoading, organizationReady, expected }) => {
      Object.assign(access, { enabled, isLoading, organizationReady });
      access.retryOrganization.mockClear();
      const onSelectRun = vi.fn();
      const view = (
        <MergeConfidenceConfigShell
          title="Automation"
          page
          onClose={() => {}}
          deleteCopy={DEFAULT_SPLIT_CONFIG_DELETE_COPY}
          deletePending={false}
          confirmDelete={false}
          onConfirmDelete={() => {}}
          onConfirmDeleteChange={() => {}}
          tab="runs"
          showRunsTab
          onTabChange={() => {}}
          containerRef={createRef<HTMLDivElement>()}
          organizationId="org"
          factoryId="factory"
          widthPercent={50}
          isResizing={false}
          onResizeStart={() => {}}
          onSaveNode={() => {}}
          graph={{
            organizationId: "org",
            factoryId: "factory",
            nodes: [{ id: nodeId, position: { x: 0, y: 0 }, data: {} }],
            edges: [],
            specNodes: [{ id: nodeId }],
          }}
          loading={false}
          error={false}
          canvasId="canvas"
          selectedRunId={null}
          onSelectedRunIdChange={onSelectRun}
          onNodeSelect={() => {}}
          selectedNodeId={null}
          focusNonce={0}
          layoutFitNonce={null}
        />
      );
      const { rerender } = render(view);
      if (isLoading || !organizationReady) {
        if (isLoading) expect(screen.getByRole("status")).toBeInTheDocument();
        else {
          expect(screen.getByRole("alert")).toBeInTheDocument();
          fireEvent.click(screen.getByRole("button", { name: "Try again" }));
          expect(access.retryOrganization).toHaveBeenCalledTimes(1);
        }
        expect(screen.queryByTestId("existing-canvas")).not.toBeInTheDocument();
        expect(onSelectRun).not.toHaveBeenCalled();
        Object.assign(access, { isLoading: false, organizationReady: true });
        rerender(cloneElement(view));
        expect(screen.getByTestId("connected-history")).toBeInTheDocument();
        expect(onSelectRun).not.toHaveBeenCalled();
        return;
      }
      if (expected) {
        fireEvent.click(screen.getByTestId("connected-history"));
        expect(onSelectRun).toHaveBeenCalledWith("selected-run");
        expect(screen.queryByTestId("existing-canvas")).not.toBeInTheDocument();
      } else {
        expect(screen.getByTestId("existing-canvas")).toBeInTheDocument();
        expect(screen.queryByTestId("connected-history")).not.toBeInTheDocument();
      }
    },
  );
});
