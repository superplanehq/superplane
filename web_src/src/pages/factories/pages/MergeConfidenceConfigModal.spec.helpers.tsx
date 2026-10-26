import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { TooltipProvider } from "@/ui/tooltip";
import { confidenceGraph } from "./MergeConfidenceConfigModal.spec.fixtures";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

export function renderModal(
  onDelete: () => void = vi.fn(),
  onSaveNode = vi.fn(),
  graph: IntakeAutomationGraph = confidenceGraph("on-pr-feedback"),
  options: { initialNodeId?: string; onClose?: () => void } = {},
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(experimentalFeaturesKeys.registry(), {
    features: [{ id: FEATURE_WORKSPACE_SKILLS, released: false }],
  });
  if (graph.organizationId) {
    client.setQueryData(organizationKeys.details(graph.organizationId), {
      spec: { enabledExperimentalFeatures: [] },
    });
  }
  const element = (nextGraph: IntakeAutomationGraph = graph) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <MergeConfidenceConfigModal
              title="Merge confidence"
              graph={nextGraph}
              canvasId="merge-confidence"
              onClose={options.onClose ?? vi.fn()}
              onSaveNode={onSaveNode}
              onDelete={onDelete}
              initialNodeId={options.initialNodeId}
              factoryKey="SUPER"
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(element());
  return { ...view, rerenderGraph: (nextGraph: IntakeAutomationGraph) => view.rerender(element(nextGraph)) };
}
