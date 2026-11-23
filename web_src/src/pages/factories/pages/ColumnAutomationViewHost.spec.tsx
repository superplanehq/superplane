import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as ApiClientModule from "@/api-client";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";

const materializeDefaults = vi.fn();

vi.mock("@/api-client", () => ({
  ...unmockedSrc<typeof ApiClientModule>("api-client"),
  factoriesMaterializeFactoryAutomationDefaults: materializeDefaults,
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

const { useCanvas, useTriggers, useComponents, useInfiniteCanvasRuns, updateCanvasVersion, commitCanvasStaging } =
  vi.hoisted(() => ({
    useCanvas: vi.fn(),
    useTriggers: vi.fn(),
    useComponents: vi.fn(),
    useInfiniteCanvasRuns: vi.fn(),
    updateCanvasVersion: vi.fn(),
    commitCanvasStaging: vi.fn(),
  }));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  ...unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData"),
  useCanvas,
  useTriggers,
  useWidgets: () => ({ data: [], isLoading: false }),
  useInfiniteCanvasRuns,
  useDescribeRun: () => ({ data: undefined, isLoading: false, isFetching: false, isPending: false, isError: false }),
  useEventExecutions: () => ({ data: { executions: [] }, isLoading: false }),
  useCanvasStaging: () => ({
    data: { hasStaging: false, stale: false },
    isPending: false,
    refetch: vi.fn().mockResolvedValue({ data: { hasStaging: false, stale: false } }),
  }),
  useUpdateCanvasVersion: () => ({ mutateAsync: updateCanvasVersion, isPending: false }),
  useCommitCanvasStaging: () => ({ mutateAsync: commitCanvasStaging, isPending: false }),
}));

vi.mock("@/hooks/useComponentData", () => ({
  ...unmockedSrc<typeof ComponentDataModule>("hooks/useComponentData"),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations"),
  useConnectedIntegrations: () => ({ data: [], isLoading: false, error: null }),
  useAvailableIntegrations: () => ({ data: [], isLoading: false }),
}));

const PULL_REQUEST_ACTIONS = {
  name: "actions",
  label: "Actions",
  type: "multi-select",
  typeOptions: {
    multiSelect: {
      options: [
        { label: "Opened", value: "opened" },
        { label: "Closed", value: "closed" },
        { label: "Merged", value: "merged" },
        { label: "Declined", value: "declined" },
      ],
    },
  },
};

const GITHUB_PULL_REQUEST_TRIGGER = {
  name: "github.onPullRequest",
  label: "On Pull Request",
  configuration: [
    { name: "customName", label: "Run title", type: "string" },
    { name: "repository", label: "Repository", type: "string", required: true },
    PULL_REQUEST_ACTIONS,
    {
      name: "ignoreDrafts",
      label: "Ignore draft pull requests",
      type: "boolean",
      description: "Do not start a run when the pull request is a draft.",
    },
    {
      name: "onlyFactoryPullRequests",
      label: "Only pull requests in this factory",
      type: "boolean",
      description: "Start a run only when the pull request belongs to this factory.",
    },
  ],
};

const BITBUCKET_PULL_REQUEST_TRIGGER = {
  name: "bitbucket.onPullRequest",
  label: "On Pull Request",
  configuration: [
    { name: "repository", label: "Repository", type: "string", required: true },
    PULL_REQUEST_ACTIONS,
    {
      name: "onlyFactoryPullRequests",
      label: "Only pull requests in this factory",
      type: "boolean",
      description: "Start a run only when the pull request belongs to this factory.",
    },
  ],
};

const UPDATE_TASK_STATUS = {
  name: "updateWorkOrderStatus",
  label: "Update Task Status",
  configuration: [
    { name: "orderId", label: "Task ID", type: "string", required: true },
    {
      name: "status",
      label: "Status",
      type: "select",
      required: true,
      typeOptions: {
        select: {
          options: [
            { label: "Draft", value: "draft" },
            { label: "Open", value: "open" },
            { label: "Closed", value: "closed" },
          ],
        },
      },
    },
  ],
};

function closureCanvas(component: "github.onPullRequest" | "bitbucket.onPullRequest", actions: string[]) {
  return {
    metadata: { id: "app-pr-closure", name: "PR Closure", factoryId: "factory-1", liveVersionId: "version-1" },
    spec: {
      nodes: [
        {
          id: "on-pr-closed",
          name: "On Pull Request Closed",
          type: "TYPE_TRIGGER",
          component,
          configuration: {
            actions,
            onlyFactoryPullRequests: true,
            repository: "{{ install_params.appRepository }}",
          },
        },
        {
          id: "complete-work-order",
          name: "Complete Task",
          type: "TYPE_ACTION",
          component: "updateWorkOrderStatus",
          configuration: {
            orderId: "{{ root().data.workOrder.id }}",
            status: "open",
          },
        },
      ],
      edges: [{ channel: "default", sourceId: "on-pr-closed", targetId: "complete-work-order" }],
    },
  };
}

function renderHost() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/org-1/workspaces/rf/lines/line-plan"]}>
        <ThemeProvider>
          <TooltipProvider>
            <ColumnAutomationViewHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="RF"
              lineId="line-plan"
              canvasId="app-pr-closure"
              title="PR Closure"
              onClose={vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function expectRunsOnly() {
  const dialog = screen.getByTestId("merge-confidence-config");
  expect(screen.queryByTestId("column-automation-view")).not.toBeInTheDocument();
  expect(within(dialog).queryByRole("tab", { name: "Automation" })).not.toBeInTheDocument();
  expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
  expect(within(dialog).queryByRole("heading", { name: "On Pull Request Closed" })).not.toBeInTheDocument();
  return dialog;
}

describe("ColumnAutomationViewHost", () => {
  beforeEach(() => {
    useCanvas.mockReturnValue({
      data: closureCanvas("github.onPullRequest", ["closed"]),
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    useTriggers.mockReturnValue({
      data: [GITHUB_PULL_REQUEST_TRIGGER, BITBUCKET_PULL_REQUEST_TRIGGER],
      isLoading: false,
    });
    useComponents.mockReturnValue({
      data: [UPDATE_TASK_STATUS],
      isLoading: false,
    });
    updateCanvasVersion.mockReset();
    updateCanvasVersion.mockResolvedValue(undefined);
    commitCanvasStaging.mockReset();
    commitCanvasStaging.mockResolvedValue(undefined);
    useInfiniteCanvasRuns.mockReturnValue({
      data: { pages: [{ runs: [] }] },
      isPending: false,
      isError: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
      refetch: vi.fn(),
    });
    localStorage.clear();
  });

  it("opens pull request closure on runs without the automation tab", () => {
    renderHost();

    expectRunsOnly();
  });

  it("edits and saves a custom automation agent without requesting bundled defaults", async () => {
    const user = userEvent.setup();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    materializeDefaults.mockReset();
    materializeDefaults.mockRejectedValue(new Error("factory app has no bundled defaults"));
    useCanvas.mockReturnValue({
      data: {
        metadata: { id: "app-pr-closure", name: "Custom automation", liveVersionId: "version-1" },
        spec: {
          nodes: [
            {
              id: "on-task",
              type: "TYPE_TRIGGER",
              component: "onWorkOrder",
              metadata: { factoryTemplate: { id: "custom-verification", version: 1 } },
            },
            {
              id: "verify",
              name: "Custom agent",
              type: "TYPE_ACTION",
              component: "runnerSuperPlane",
              configuration: { steps: [{ name: "Verify task", type: "prompt", prompt: "Custom prompt" }] },
            },
          ],
          edges: [{ sourceId: "on-task", targetId: "verify", channel: "default" }],
        },
      },
      isPending: false,
      isError: false,
      refetch: vi.fn(),
    });
    try {
      renderHost();
      await user.click(screen.getByRole("tab", { name: "Agent" }));
      await act(async () => {
        await Promise.resolve();
      });
      expect(materializeDefaults).not.toHaveBeenCalled();
      expect(warn).not.toHaveBeenCalled();
      expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
      expect(screen.queryByTestId("planning-review-restore-default-prompt-error")).not.toBeInTheDocument();
      expect(screen.queryByTestId("planning-review-restore-default-prompt-retry")).not.toBeInTheDocument();

      await user.click(screen.getByTestId("planning-review-step-toggle-0"));
      const prompt = screen.getByTestId("planning-review-step-body-0");
      await user.clear(prompt);
      await user.type(prompt, "Updated custom prompt");
      await user.click(screen.getByRole("button", { name: "Save Agent" }));
      await waitFor(() => expect(commitCanvasStaging).toHaveBeenCalledWith("Update agent"));
      const parsed = parseCanvasYamlForImport(updateCanvasVersion.mock.calls[0][0].canvasYaml);
      expect(parsed.ok && parsed.spec.nodes?.find((node) => node.id === "verify")?.configuration?.steps).toEqual([
        expect.objectContaining({ prompt: "Updated custom prompt" }),
      ]);
    } finally {
      warn.mockRestore();
    }
  });
});
