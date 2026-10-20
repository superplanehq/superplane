import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";
import type { SuperplaneComponentsNode } from "@/api-client";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { ImplementConfiguration } from "./ImplementConfiguration";
import { implementConfigGraph } from "./MergeConfidenceConfigModal.spec.fixtures";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

function renderSettings(agent: SuperplaneComponentsNode, onSave: (update: NodeConfigurationUpdate) => void) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(experimentalFeaturesKeys.registry(), { features: [] });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ImplementConfiguration agent={agent} nodes={[agent]} onSave={onSave} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ImplementConfiguration", () => {
  it.each([undefined, 0, 5400])("edits the Implement timeout in minutes from %s seconds", async (timeout) => {
    const onSaveNode = vi.fn();
    const graph = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Keep this prompt" }]);
    const agent = graph.specNodes![1];
    agent.configuration = { ...agent.configuration, executionTimeoutSeconds: timeout, machineType: "custom-runner" };
    renderSettings(agent, onSaveNode);

    const duration = screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" });
    expect(duration).toHaveValue(timeout ? timeout / 60 : 60);
    expect(onSaveNode).not.toHaveBeenCalled();

    for (const invalid of ["", "0", "-1", "1441"]) {
      fireEvent.change(duration, { target: { value: invalid } });
      fireEvent.blur(duration);
      expect(duration).toHaveAttribute("aria-invalid", "true");
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(onSaveNode).not.toHaveBeenCalled();
    }
    fireEvent.change(duration, { target: { value: "120" } });
    expect(onSaveNode).not.toHaveBeenCalled();
    fireEvent.blur(duration);

    await waitFor(() => expect(onSaveNode).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledWith(
      expect.objectContaining({
        nodeId: agent.id,
        configuration: { ...agent.configuration, executionTimeoutSeconds: 7200 },
        concurrency: agent.concurrency,
      }),
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it.each(["120", "0", ""])("preserves only valid duration edits when dismissed without blur: %s", async (minutes) => {
    const onSaveNode = vi.fn();
    const graph = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Keep this prompt" }]);
    const agent = graph.specNodes![1];
    const view = renderSettings(agent, onSaveNode);
    fireEvent.change(screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" }), {
      target: { value: minutes },
    });
    expect(onSaveNode).not.toHaveBeenCalled();
    view.unmount();

    if (minutes !== "120") {
      expect(onSaveNode).not.toHaveBeenCalled();
      return;
    }
    await waitFor(() => expect(onSaveNode).toHaveBeenCalledTimes(1));
    expect(onSaveNode).toHaveBeenCalledWith(
      expect.objectContaining({
        configuration: { ...graph.specNodes![1].configuration, executionTimeoutSeconds: 7200 },
      }),
    );
  });
});
