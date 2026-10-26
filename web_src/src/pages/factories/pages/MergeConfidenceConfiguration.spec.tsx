import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import type { SuperplaneComponentsNode } from "@/api-client";
import { MergeConfidenceConfiguration } from "./MergeConfidenceConfiguration";
import { MERGE_CONFIDENCE_CHECKS } from "./mergeConfidenceChecks";
import type { PlanningReviewStep } from "./planningReviewMockup";

const loadDefaults = vi.fn();
vi.mock("../lib/loadDefaultAgentPrompt", () => ({ loadDefaultAgentPrompt: loadDefaults }));
vi.mock("@/hooks/useComponentData", () => ({
  useComponent: () => ({
    data: { configuration: [{ name: "model", label: "Model", type: "string" }] },
    isLoading: false,
  }),
  useComponents: () => ({
    data: [
      {
        name: "runnerClaudeCode",
        label: "Run Claude Code",
        configuration: [{ name: "model", label: "Model", type: "string" }],
      },
    ],
    isLoading: false,
  }),
}));
const steps: PlanningReviewStep[] = MERGE_CONFIDENCE_CHECKS.map((id) => ({
  name: id,
  type: "prompt",
  prompt: `Merge check: ${id}.\nSaved ${id} rules.`,
}));
const custom: PlanningReviewStep = { name: "Custom", type: "prompt", prompt: "Merge check: new-check.\nKeep this." };
const legacyReview = [
  "Review this pull request. The diff is in /tmp/pr.diff.",
  "Report each enabled check with the report_merge_check tool.",
  "Call the tool once for every check named in Enabled checks.",
  "Do not write a file.",
  "",
  "Enabled checks: risk, security.",
  "",
  "Risk. Use this section only when risk is enabled.",
  "A higher score means more risk.",
  "",
  "Security. Use this section only when security is enabled.",
  "Read the security practices.",
  "",
].join("\n");

function renderSettings(savedSteps = steps, onSave = vi.fn(), onClose = vi.fn()) {
  const agent: SuperplaneComponentsNode = {
    id: "assess-risk",
    name: "Assess risk",
    component: "runnerClaudeCode",
    configuration: { model: "sonnet", executionTimeoutSeconds: 3600, steps: savedSteps },
    concurrency: { max: 5, key: "pr" },
  };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(experimentalFeaturesKeys.registry(), { features: [] });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MergeConfidenceConfiguration
          agent={agent}
          organizationId="org"
          factoryId="factory"
          canvasId="automation"
          onSave={onSave}
          onClose={onClose}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onSave, onClose };
}

beforeEach(() => loadDefaults.mockReset());

describe("Merge confidence configuration", () => {
  it("saves runner options without rewriting stored prompts and retries a failed save", async () => {
    const stored = [steps[2], custom, { name: "Setup", type: "bash" as const, command: "git status" }, steps[0]];
    let rejectSave = (_error: Error) => {};
    const pending = new Promise<void>((_resolve, reject) => {
      rejectSave = reject;
    });
    const onSave = vi.fn().mockReturnValueOnce(pending).mockResolvedValue(undefined);
    const { onClose } = renderSettings(stored, onSave);
    fireEvent.change(screen.getByDisplayValue("sonnet"), { target: { value: "opus" } });
    const concurrency = screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" });
    fireEvent.change(concurrency, { target: { value: "3" } });
    fireEvent.blur(concurrency);
    expect(onSave).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("button", { name: "Saving" })).toBeDisabled();
    expect(screen.getByRole("switch", { name: "Security" })).toBeDisabled();
    expect(concurrency).toBeDisabled();
    rejectSave(new Error("Network error"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not save Merge confidence settings");
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByDisplayValue("opus")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenLastCalledWith(
      expect.objectContaining({
        configuration: { model: "opus", executionTimeoutSeconds: 3600, steps: stored },
        concurrency: { max: 3, key: "pr" },
      }),
    );
    expect(loadDefaults).not.toHaveBeenCalled();
  });

  it("shows scores from an older combined prompt and saves them as separate checks", async () => {
    const checkout: PlanningReviewStep = { name: "Checkout Pull Request", type: "bash", command: "git clone" };
    const { onSave } = renderSettings([
      checkout,
      { name: "Review Pull Request", type: "prompt", prompt: legacyReview, workingDirectory: "repo" },
    ]);
    expect(screen.getByRole("switch", { name: "Blast radius" })).toBeChecked();
    expect(screen.getByRole("switch", { name: "Security" })).toBeChecked();
    expect(screen.getByRole("switch", { name: "Performance" })).not.toBeChecked();
    expect(screen.getByRole("switch", { name: "Drift from Specification" })).not.toBeChecked();
    expect(screen.getByRole("switch", { name: "Reversibility" })).not.toBeChecked();

    fireEvent.change(screen.getByDisplayValue("sonnet"), { target: { value: "opus" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const saved = onSave.mock.calls[0]?.[0] as { configuration: { model: string; steps: PlanningReviewStep[] } };
    expect(saved.configuration.model).toBe("opus");
    expect(saved.configuration.steps.map((step) => step.name)).toEqual([
      "Checkout Pull Request",
      "Blast radius",
      "Security",
    ]);
    expect(saved.configuration.steps.map((step) => step.prompt).join("\n")).not.toContain("Enabled checks:");
    expect(saved.configuration.steps[1]?.prompt).toContain("Merge check: risk.");
    expect(saved.configuration.steps[2]?.prompt).toContain("Merge check: security.");
    expect(loadDefaults).not.toHaveBeenCalled();
  });

  it("removes a score stored only in an older combined prompt", async () => {
    const { onSave } = renderSettings([
      { name: "Review Pull Request", type: "prompt", prompt: legacyReview, workingDirectory: "repo" },
    ]);
    fireEvent.click(screen.getByRole("switch", { name: "Security" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const saved = onSave.mock.calls[0]?.[0] as { configuration: { steps: PlanningReviewStep[] } };
    expect(saved.configuration.steps.map((step) => step.name)).toEqual(["Blast radius"]);
    expect(saved.configuration.steps[0]?.prompt).toContain("Merge check: risk.");
    expect(saved.configuration.steps.map((step) => step.prompt).join("\n")).not.toContain("Enabled checks:");
    expect(saved.configuration.steps.map((step) => step.prompt).join("\n")).not.toContain("Merge check: security.");
  });

  it("removes Security only on Save and preserves custom checks and runner settings", async () => {
    const { onSave, onClose } = renderSettings([...steps, custom]);
    fireEvent.click(screen.getByRole("switch", { name: "Security" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), {
      target: { value: "Review API compatibility." },
    });
    const duration = screen.getByRole("spinbutton", { name: "Maximum run duration (minutes)" });
    fireEvent.change(duration, { target: { value: "120" } });
    fireEvent.blur(duration);
    expect(onSave).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        nodeId: "assess-risk",
        concurrency: { max: 5, key: "pr" },
        configuration: {
          model: "sonnet",
          executionTimeoutSeconds: 7200,
          additionalInstructions: "Review API compatibility.",
          steps: [...steps.filter((step) => step.name !== "security"), custom],
        },
      }),
    );
    expect(loadDefaults).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it("loads and inserts a newly enabled default in canonical order", async () => {
    const security: PlanningReviewStep = {
      name: "Default Security",
      type: "prompt",
      prompt: "Merge check: security.\nBundled rules.",
    };
    loadDefaults.mockResolvedValue([security]);
    const { onSave } = renderSettings(steps.filter((step) => step.name !== "security"));
    fireEvent.click(screen.getByRole("switch", { name: "Security" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(loadDefaults).toHaveBeenCalledWith({
      organizationId: "org",
      factoryId: "factory",
      automationId: "automation",
      agentNodeId: "assess-risk",
    });
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        configuration: expect.objectContaining({ steps: [steps[0], steps[1], security, steps[3], steps[4]] }),
      }),
    );
  });

  it("keeps the form open after a defaults failure and permits retry", async () => {
    loadDefaults.mockRejectedValueOnce(new Error("Network error")).mockResolvedValue([]);
    const { onSave, onClose } = renderSettings([]);
    fireEvent.click(screen.getByRole("switch", { name: "Security" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(onSave).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Security"));
    expect(onSave).not.toHaveBeenCalled();
  });

  it.each([false, true])("turns all known checks off while preserving custom checks: %s", async (hasCustom) => {
    const { onSave } = renderSettings(hasCustom ? [...steps, custom] : steps);
    for (const control of screen.getAllByRole("switch")) fireEvent.click(control);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        configuration: expect.objectContaining({
          steps: hasCustom ? [custom] : [expect.objectContaining({ name: "No checks enabled", type: "prompt" })],
        }),
      }),
    );
  });
});
