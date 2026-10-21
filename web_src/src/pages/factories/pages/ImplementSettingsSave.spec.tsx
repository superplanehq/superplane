import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { FEATURE_FACTORY_RUN_HISTORY } from "@/lib/experimentalFeatures";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { implementConfigGraph } from "./MergeConfidenceConfigModal.spec.fixtures";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData");
  return {
    ...actual,
    useInfiniteCanvasRuns: () => ({
      data: { pages: [{ runs: [] }] },
      isPending: false,
      isError: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
      refetch: vi.fn(),
    }),
    useDescribeRun: () => ({ data: undefined, isLoading: false, isFetching: false, isPending: false, isError: false }),
    useTriggers: () => ({ data: [], isLoading: false }),
    useWidgets: () => ({ data: [], isLoading: false }),
  };
});

vi.mock("@/hooks/useComponentData", () => ({
  useComponents: () => ({ data: [], isLoading: false }),
  useComponent: () => ({
    data: { configuration: [{ name: "model", label: "Model", type: "string" }] },
    isLoading: false,
  }),
}));

function renderImplementSettings(
  onSaveNode: (update: { nodeId: string }) => Promise<void> | void,
  onClose: () => void,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const graph = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Keep this prompt" }]);
  client.setQueryData(experimentalFeaturesKeys.registry(), {
    features: [{ id: FEATURE_FACTORY_RUN_HISTORY, released: false }],
  });
  client.setQueryData(organizationKeys.details(graph.organizationId ?? ""), {
    spec: { enabledExperimentalFeatures: [FEATURE_FACTORY_RUN_HISTORY] },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <MergeConfidenceConfigModal
              title="Implement"
              graph={graph}
              canvasId="implement"
              onClose={onClose}
              onSaveNode={onSaveNode}
              factoryKey="SUPER"
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Implement settings save", () => {
  it("closes without a request when nothing changed", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSaveNode = vi.fn();
    renderImplementSettings(onSaveNode, onClose);

    await user.click(screen.getByTestId("implement-settings-save"));

    expect(onSaveNode).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("retries only the nodes that did not save", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    let failPullRequest = true;
    const onSaveNode = vi.fn(async (update: { nodeId: string }) => {
      if (failPullRequest && update.nodeId === "create-pr") throw new Error("save failed");
    });
    renderImplementSettings(onSaveNode, onClose);

    await user.clear(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }));
    await user.type(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }), "3");
    await user.click(screen.getByRole("switch", { name: "Create draft pull requests" }));
    await user.click(screen.getByTestId("implement-settings-save"));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not save Implement settings.");
    expect(onClose).not.toHaveBeenCalled();
    expect(onSaveNode).toHaveBeenCalledTimes(2);
    failPullRequest = false;
    onSaveNode.mockClear();
    await user.click(screen.getByTestId("implement-settings-save"));

    await waitFor(() => expect(onSaveNode).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledWith(expect.objectContaining({ nodeId: "create-pr" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("discards edits when the screen closes", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSaveNode = vi.fn();
    renderImplementSettings(onSaveNode, onClose);

    await user.click(screen.getByRole("switch", { name: "Create draft pull requests" }));
    await user.click(screen.getByRole("button", { name: "Close" }));

    expect(onSaveNode).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
