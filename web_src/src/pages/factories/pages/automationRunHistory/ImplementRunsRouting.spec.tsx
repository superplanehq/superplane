import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { createRef } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MergeConfidenceConfigShell } from "../MergeConfidenceConfigShell";
import { DEFAULT_SPLIT_CONFIG_DELETE_COPY } from "../splitConfigDeleteCopy";

vi.mock("./FactoryAutomationRunHistory", () => ({
  FactoryAutomationRunHistory: ({ onSelectRun }: { onSelectRun: (id: string | null) => void }) => (
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
  it.each(["implementation-agent-no-issue", "on-pr-risk"])(
    "opens %s history without an experimental feature lookup",
    (nodeId) => {
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
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      client.setQueryData(experimentalFeaturesKeys.registry(), { features: [] });
      client.setQueryData(organizationKeys.details("org"), { spec: { enabledExperimentalFeatures: [] } });
      render(<QueryClientProvider client={client}>{view}</QueryClientProvider>);
      fireEvent.click(screen.getByTestId("connected-history"));
      expect(onSelectRun).toHaveBeenCalledWith("selected-run");
      expect(screen.queryByTestId("existing-canvas")).not.toBeInTheDocument();
    },
  );
});
