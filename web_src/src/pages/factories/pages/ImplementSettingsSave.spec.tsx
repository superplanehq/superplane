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
import { useImplementSettingsDraft } from "./implementSettingsDraft";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";

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
  onSaveNode: (update: NodeConfigurationSave) => Promise<void> | void,
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

  it("saves changed nodes together and retries that write", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    let failSave = true;
    const onSaveNode = vi.fn(async () => {
      if (failSave) throw new Error("save failed");
    });
    renderImplementSettings(onSaveNode, onClose);

    await user.clear(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }));
    await user.type(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" }), "3");
    await user.click(screen.getByRole("switch", { name: "Create draft pull requests" }));
    await user.click(screen.getByTestId("implement-settings-save"));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not save Implement settings.");
    expect(onClose).not.toHaveBeenCalled();
    expect(onSaveNode).toHaveBeenCalledTimes(1);
    expect(onSaveNode).toHaveBeenCalledWith([
      expect.objectContaining({
        nodeId: "implementation-agent-no-issue",
        concurrency: { max: 3, key: "implementation" },
      }),
      expect.objectContaining({ nodeId: "create-pr", configuration: expect.objectContaining({ draft: false }) }),
    ]);
    failSave = false;
    onSaveNode.mockClear();
    await user.click(screen.getByTestId("implement-settings-save"));

    await waitFor(() => expect(onSaveNode).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledWith([
      expect.objectContaining({ nodeId: "implementation-agent-no-issue" }),
      expect.objectContaining({ nodeId: "create-pr" }),
    ]);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("locks fields while save is in progress", async () => {
    const user = userEvent.setup();
    let finishSave = () => {};
    const saving = new Promise<void>((resolve) => {
      finishSave = resolve;
    });
    const onClose = vi.fn();
    const onSaveNode = vi.fn(() => saving);
    renderImplementSettings(onSaveNode, onClose);

    await user.click(screen.getByRole("switch", { name: "Create draft pull requests" }));
    await user.click(screen.getByTestId("implement-settings-save"));

    const configuration = screen.getByRole("region", { name: "Implement configuration" });
    expect(configuration).toHaveAttribute("inert");
    expect(screen.getByRole("switch", { name: "Create draft pull requests" })).toBeDisabled();
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toBeDisabled();
    expect(screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" })).toBeDisabled();
    finishSave();

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledTimes(1);
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

  it("keeps the screen open when an edit arrives during save", async () => {
    const user = userEvent.setup();
    let finishSave = () => {};
    const saving = new Promise<void>((resolve) => {
      finishSave = resolve;
    });
    const onClose = vi.fn();
    const onSaveNode = vi.fn(() => saving);
    const graph = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Keep this prompt" }]);

    function Harness() {
      const settings = useImplementSettingsDraft(graph.specNodes, true);
      return (
        <>
          <button type="button" onClick={() => settings.updatePullRequestDraft(false)}>
            Mark ready
          </button>
          <button type="button" onClick={() => settings.updatePullRequestDraft(true)}>
            Mark draft
          </button>
          <button type="button" onClick={() => void settings.save(onSaveNode, onClose)}>
            Save draft
          </button>
          <span>{settings.saving ? "saving" : "idle"}</span>
        </>
      );
    }

    render(<Harness />);

    await user.click(screen.getByRole("button", { name: "Mark ready" }));
    await user.click(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByText("saving")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Mark draft" }));
    finishSave();

    expect(await screen.findByText("idle")).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
    onSaveNode.mockClear();
    await user.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledWith(
      expect.objectContaining({ nodeId: "create-pr", configuration: expect.objectContaining({ draft: true }) }),
    );
  });
});
