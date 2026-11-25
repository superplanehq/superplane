import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import type { SuperplaneComponentsNode } from "@/api-client";
import { MergeConfidenceConfiguration } from "./MergeConfidenceConfiguration";
import { MERGE_CONFIDENCE_CHECKS } from "./mergeConfidenceChecks";
import type { PlanningReviewStep } from "./planningReviewMockup";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

const checkPermissions = vi.fn();
vi.mock("@/hooks/useIntegrations", () => ({
  integrationResourcesQueryOptions: () => ({ queryFn: checkPermissions }),
}));
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

function renderSettings(
  savedSteps = steps,
  onSave = vi.fn(),
  onClose = vi.fn(),
  publishing?: SuperplaneComponentsNode,
  options: { configuration?: Record<string, unknown>; publicationNodes?: SuperplaneComponentsNode[] } = {},
) {
  const agent: SuperplaneComponentsNode = {
    id: "assess-risk",
    name: "Assess risk",
    component: "runnerClaudeCode",
    configuration: { model: "sonnet", executionTimeoutSeconds: 3600, steps: savedSteps, ...options.configuration },
    concurrency: { max: 5, key: "pr" },
  };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(experimentalFeaturesKeys.registry(), { features: [] });
  const rendered = render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MergeConfidenceConfiguration
          agent={agent}
          publishing={publishing}
          publicationNodes={options.publicationNodes}
          organizationId="org"
          factoryId="factory"
          canvasId="automation"
          onSave={onSave}
          onClose={onClose}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onSave, onClose, unmount: rendered.unmount };
}

beforeEach(() => {
  loadDefaults.mockReset();
  checkPermissions.mockReset();
});

describe("Merge confidence configuration", () => {
  it.each(["granted", "missing", "unavailable"])(
    "checks GitHub access before enabling PR checks: %s",
    async (access) => {
      const body: SuperplaneComponentsNode = {
        id: "publish-merge-confidence",
        component: "github.updatePullRequestBodySection",
        configuration: { enabled: true },
      };
      const start: SuperplaneComponentsNode = {
        id: "start-assessment-check",
        component: "github.setPullRequestCheckRun",
        integration: { id: "github" },
        configuration: { enabled: false },
      };
      let finishCheck!: (resources: { id: string; name?: string }[]) => void;
      let failCheck!: (error: Error) => void;
      checkPermissions.mockImplementation(
        () =>
          new Promise((resolve, reject) => {
            finishCheck = resolve;
            failCheck = reject;
          }),
      );
      const { onSave } = renderSettings(steps, vi.fn(), vi.fn(), body, { publicationNodes: [body, start] });
      const toggle = screen.getByRole("switch", { name: "Publish PR check" });
      expect(toggle).not.toBeChecked();
      fireEvent.click(toggle);
      expect(toggle).toBeDisabled();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
      await act(async () => {
        if (access === "unavailable") failCheck(new Error("GitHub unavailable"));
        else finishCheck([{ id: access, name: "Accept the new GitHub App permissions, then try again." }]);
      });
      expect(toggle).not.toBeDisabled();
      if (access !== "granted") {
        expect(toggle).not.toBeChecked();
        expect(screen.getByRole("alert")).toHaveTextContent(
          access === "missing" ? "Accept the new GitHub App permissions" : "Could not verify GitHub permissions",
        );
        expect(onSave).not.toHaveBeenCalled();
        checkPermissions.mockResolvedValue([{ id: "granted" }]);
        await act(async () => fireEvent.click(toggle));
        expect(toggle).toBeChecked();
      }
      expect(toggle).toBeChecked();
      fireEvent.click(screen.getByRole("button", { name: "Save" }));
      await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
      expect(
        onSave.mock.calls[0][0].find((node: NodeConfigurationUpdate) => node.nodeId === "start-assessment-check")
          ?.configuration,
      ).toEqual(expect.objectContaining({ enabled: true, activationGeneration: 2 }));
    },
  );

  it("saves independent controls, reloads choices, and fences earlier activations", async () => {
    const body: SuperplaneComponentsNode = {
      id: "publish-merge-confidence",
      component: "github.updatePullRequestBodySection",
      configuration: { enabled: true },
    };
    const nodes: SuperplaneComponentsNode[] = [
      body,
      {
        id: "comment-merge-confidence",
        component: "github.publishPullRequestComment",
        configuration: { enabled: false },
      },
      { id: "start-assessment-check", component: "github.setPullRequestCheckRun", configuration: { enabled: true } },
      { id: "finish-assessment-check", component: "github.setPullRequestCheckRun", configuration: {} },
    ];
    const onSave = vi.fn();
    let rendered = renderSettings(steps, onSave, vi.fn(), body, { publicationNodes: nodes });
    fireEvent.click(screen.getByRole("switch", { name: "Publish PR check" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "PR body" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "PR comment" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const first = onSave.mock.calls[0][0] as NodeConfigurationUpdate[];
    expect(first[0].configuration).toEqual(
      expect.objectContaining({
        publishPRCheck: false,
        publishResults: true,
        resultsBody: false,
        resultsComment: true,
      }),
    );
    expect(first.find((node) => node.nodeId === "comment-merge-confidence")?.configuration).toEqual(
      expect.objectContaining({ enabled: true, activationGeneration: 2 }),
    );
    expect(first.find((node) => node.nodeId === "start-assessment-check")?.configuration).toEqual(
      expect.objectContaining({ enabled: false }),
    );
    rendered.unmount();
    const reload = (updates: NodeConfigurationUpdate[]) => {
      const savedNodes = updates.slice(1).map((update) => ({
        ...nodes.find((node) => node.id === update.nodeId),
        id: update.nodeId,
        configuration: update.configuration,
      }));
      return renderSettings(steps, onSave, vi.fn(), savedNodes[0], {
        configuration: updates[0].configuration,
        publicationNodes: savedNodes,
      });
    };
    rendered = reload(first);
    expect(screen.getByRole("checkbox", { name: "PR comment" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "PR body" })).not.toBeChecked();
    fireEvent.click(screen.getByRole("switch", { name: "Publish results" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(2));
    const off = onSave.mock.calls[1][0] as NodeConfigurationUpdate[];
    expect(off[0].configuration).toEqual(
      expect.objectContaining({ publishResults: false, resultsBody: false, resultsComment: true }),
    );
    rendered.unmount();
    rendered = reload(off);
    expect(screen.getByRole("switch", { name: "Publish PR check" })).not.toBeChecked();
    expect(screen.queryByRole("checkbox", { name: "PR comment" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("switch", { name: "Publish results" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(3));
    const enabled = onSave.mock.calls[2][0] as NodeConfigurationUpdate[];
    expect(enabled.find((node) => node.nodeId === "comment-merge-confidence")?.configuration).toEqual(
      expect.objectContaining({ enabled: true, activationGeneration: 3 }),
    );
    expect(loadDefaults).not.toHaveBeenCalled();
    rendered.unmount();
  });
  it.each(["github", "bitbucket"])(
    "upgrades existing %s completion rules when results are disabled",
    async (provider) => {
      const predicate =
        '$["Assess Merge Confidence"].data.mergeConfidence.complete == true && previous().data.status in ["published", "unchanged"]';
      const body: SuperplaneComponentsNode = {
        id: "publish-merge-confidence",
        component: `${provider}.updatePullRequestBodySection`,
        configuration: { enabled: true },
      };
      const completion: SuperplaneComponentsNode = {
        id: "complete-merge-confidence",
        component: "filter",
        configuration: { expression: predicate },
      };
      const finish: SuperplaneComponentsNode = {
        id: "finish-assessment-check",
        component: "github.setPullRequestCheckRun",
        configuration: {
          state: `{{ ${predicate} ? "success" : "failure" }}`,
          reason: `{{ ${predicate} ? "Complete" : "Incomplete" }}`,
        },
      };
      const nodes =
        provider === "github"
          ? [
              body,
              completion,
              finish,
              {
                id: "start-assessment-check",
                component: "github.setPullRequestCheckRun",
                configuration: {},
              },
            ]
          : [body, completion];
      const { onSave } = renderSettings(steps, vi.fn(), vi.fn(), body, { publicationNodes: nodes });
      fireEvent.click(screen.getByRole("switch", { name: "Publish results" }));
      fireEvent.click(screen.getByRole("button", { name: "Save" }));
      await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
      const updates = onSave.mock.calls[0][0] as NodeConfigurationUpdate[];
      const upgraded = predicate.replace('["published", "unchanged"]', '["published", "unchanged", "disabled"]');
      expect(updates.find((node) => node.nodeId === completion.id)?.configuration.expression).toBe(upgraded);
      if (provider === "github") {
        const final = updates.filter((node) => node.nodeId === finish.id);
        expect(final).toHaveLength(1);
        expect(final[0].configuration).toEqual(
          expect.objectContaining({
            state: `{{ ${upgraded} ? "success" : "failure" }}`,
            reason: `{{ ${upgraded} ? "Complete" : "Incomplete" }}`,
            activationNodeId: "start-assessment-check",
          }),
        );
      }
      expect(completion.configuration?.expression).toBe(predicate);
    },
  );
  it("validates destinations and preserves choices while publication is off", async () => {
    const { onSave } = renderSettings(steps, vi.fn(), vi.fn(), {
      id: "publish-merge-confidence",
      component: "github.updatePullRequestBodySection",
      configuration: { sectionKey: "merge-confidence" },
    });
    fireEvent.click(screen.getByRole("checkbox", { name: "PR body" }));
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    fireEvent.click(screen.getByRole("switch", { name: "Publish results" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave.mock.calls[0][0][0].configuration).toEqual(
      expect.objectContaining({
        publishResults: false,
        resultsBody: false,
        resultsComment: false,
      }),
    );
  });
  it("loads a saved opt-out and permits publishing again", async () => {
    const { onSave } = renderSettings(steps, vi.fn(), vi.fn(), {
      id: "publication",
      component: "bitbucket.updatePullRequestBodySection",
      configuration: { sectionKey: "merge-confidence", enabled: false },
    });
    const publishing = screen.getByRole("switch", { name: "Publish results" });
    expect(publishing).not.toBeChecked();
    fireEvent.click(publishing);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenCalledWith([
      expect.objectContaining({ nodeId: "assess-risk" }),
      expect.objectContaining({
        nodeId: "publication",
        configuration: { sectionKey: "merge-confidence", enabled: true, activationGeneration: 2 },
      }),
    ]);
  });

  it("saves publication with agent settings and retains the opt-out after a failed save", async () => {
    const onSave = vi.fn().mockRejectedValueOnce(new Error("Network error")).mockResolvedValue(undefined);
    const { onClose } = renderSettings(steps, onSave, vi.fn(), {
      id: "publish-merge-confidence",
      name: "Publish results",
      component: "github.updatePullRequestBodySection",
      configuration: { sectionKey: "merge-confidence", body: "{{ event.data.body }}", pullRequestId: "pr" },
    });
    const publishing = screen.getByRole("switch", { name: "Publish results" });
    expect(publishing).toBeChecked();
    fireEvent.click(publishing);
    fireEvent.change(screen.getByDisplayValue("sonnet"), { target: { value: "opus" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(publishing).not.toBeChecked();
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    expect(onSave).toHaveBeenLastCalledWith([
      expect.objectContaining({ configuration: expect.objectContaining({ model: "opus" }) }),
      expect.objectContaining({
        nodeId: "publish-merge-confidence",
        configuration: {
          enabled: false,
          activationGeneration: 1,
          sectionKey: "merge-confidence",
          body: "{{ event.data.body }}",
          pullRequestId: "pr",
        },
      }),
    ]);
  });

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
    expect(await screen.findByRole("alert")).toBeVisible();
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
