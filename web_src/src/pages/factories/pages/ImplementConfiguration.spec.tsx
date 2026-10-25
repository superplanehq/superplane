import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";
import type { SuperplaneComponentsNode } from "@/api-client";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { ImplementConfiguration } from "./ImplementConfiguration";
import { implementConfigGraph } from "./MergeConfidenceConfigModal.spec.fixtures";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";

function renderSettings(agent: SuperplaneComponentsNode, onSave: (update: NodeConfigurationSave) => void) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(experimentalFeaturesKeys.registry(), { features: [] });
  const settings = (node: SuperplaneComponentsNode) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ImplementConfiguration agent={node} nodes={[node]} onSave={onSave} />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(settings(agent));
  return { ...view, rerenderSettings: (node: SuperplaneComponentsNode) => view.rerender(settings(node)) };
}

describe("ImplementConfiguration", () => {
  it.each([false, true])("accepts refreshed instructions unless locally edited: %s", async (hasLocalEdits) => {
    const onSave = vi.fn();
    const agent = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Managed prompt" }])
      .specNodes![1];
    agent.configuration = { ...agent.configuration, additionalInstructions: "Original instructions" };
    const view = renderSettings(agent, onSave);
    if (hasLocalEdits)
      fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), {
        target: { value: "Unsaved local instructions" },
      });
    const updated = {
      ...agent,
      configuration: { ...agent.configuration, additionalInstructions: "Instructions saved in another tab" },
    };
    view.rerenderSettings(updated);
    const expected = hasLocalEdits ? "Unsaved local instructions" : "Instructions saved in another tab";
    expect(screen.getByRole("textbox", { name: "Additional instructions" })).toHaveValue(expected);
    const duration = screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" });
    fireEvent.change(duration, { target: { value: "120" } });
    fireEvent.blur(duration);
    expect(onSave).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("implement-settings-save"));
    await waitFor(() =>
      expect(onSave).toHaveBeenLastCalledWith(
        expect.objectContaining({
          configuration: { ...updated.configuration, additionalInstructions: expected, executionTimeoutSeconds: 7200 },
        }),
      ),
    );
  });

  it("saves additional instructions only with Save and supports clearing them", async () => {
    const onSave = vi.fn();
    const agent = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Managed prompt" }])
      .specNodes![1];
    renderSettings(agent, onSave);
    const instructions = screen.getByRole("textbox", { name: "Additional instructions" });
    expect(instructions).toHaveValue("");
    fireEvent.change(instructions, { target: { value: "Avoid new dependencies.\nInclude API examples in PRs." } });
    fireEvent.blur(instructions);
    expect(onSave).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("implement-settings-save"));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenLastCalledWith(
      expect.objectContaining({
        configuration: {
          ...agent.configuration,
          additionalInstructions: "Avoid new dependencies.\nInclude API examples in PRs.",
        },
      }),
    );
    fireEvent.change(instructions, { target: { value: "" } });
    fireEvent.click(screen.getByTestId("implement-settings-save"));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(2));
    expect(onSave).toHaveBeenLastCalledWith(
      expect.objectContaining({ configuration: { ...agent.configuration, additionalInstructions: "" } }),
    );
  });

  it("keeps instructions after a failed save and retries with Save", async () => {
    const onSave = vi.fn().mockRejectedValueOnce(new Error("Network error")).mockResolvedValue(undefined);
    const agent = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Managed prompt" }])
      .specNodes![1];
    renderSettings(agent, onSave);
    const instructions = screen.getByRole("textbox", { name: "Additional instructions" });
    fireEvent.change(instructions, { target: { value: "Use existing test helpers." } });
    fireEvent.click(screen.getByTestId("implement-settings-save"));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(instructions).toHaveValue("Use existing test helpers.");
    fireEvent.click(screen.getByTestId("implement-settings-save"));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
  });

  it("discards instructions when leaving without Save", () => {
    const onSave = vi.fn();
    const agent = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Managed prompt" }])
      .specNodes![1];
    const view = renderSettings(agent, onSave);
    const instructions = screen.getByRole("textbox", { name: "Additional instructions" });
    fireEvent.change(instructions, { target: { value: "Keep changes focused." } });
    fireEvent.blur(instructions);
    view.unmount();
    expect(onSave).not.toHaveBeenCalled();
  });

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
      expect(screen.getByRole("alert")).toHaveTextContent("Enter a duration from 1 second to 1440 minutes.");
      expect(screen.getByTestId("implement-settings-save")).toBeDisabled();
      expect(onSaveNode).not.toHaveBeenCalled();
    }
    fireEvent.change(duration, { target: { value: "120" } });
    expect(onSaveNode).not.toHaveBeenCalled();
    fireEvent.blur(duration);
    fireEvent.keyDown(duration, { key: "Enter" });
    expect(onSaveNode).not.toHaveBeenCalled();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("implement-settings-save"));
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

  it.each(["120", "0", ""])("does not save duration edits on blur or unmount: %s", (minutes) => {
    const onSaveNode = vi.fn();
    const graph = implementConfigGraph([{ name: "Implementation", type: "prompt", prompt: "Keep this prompt" }]);
    const agent = graph.specNodes![1];
    const view = renderSettings(agent, onSaveNode);
    const duration = screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" });
    fireEvent.change(duration, { target: { value: minutes } });
    fireEvent.blur(duration);
    fireEvent.keyDown(duration, { key: "Enter" });
    expect(onSaveNode).not.toHaveBeenCalled();
    view.unmount();
    expect(onSaveNode).not.toHaveBeenCalled();
  });
});
